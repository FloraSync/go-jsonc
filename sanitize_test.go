// Copyright 2023 Marco Zaccaro. All Rights Reserved.
// This file was modified by FloraSync in 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeExactOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strict JSON and comment-looking strings",
			input: `{"url":"https://x","block":"/* x */","values":[1,2]}`,
			want:  `{"url":"https://x","block":"/* x */","values":[1,2]}`,
		},
		{
			name:  "odd backslash parity keeps marker in string",
			input: `{"s":"quote: \\\" // not comment"}`,
			want:  `{"s":"quote: \\\" // not comment"}`,
		},
		{
			name:  "even backslash parity closes string",
			input: `{"s":"backslash \\\\" /* comment */}`,
			want: `{"s":"backslash \\\\"` +
				strings.Repeat(" ", len(" /* comment */")) + "}",
		},
		{name: "line LF", input: "// c\n1", want: "    \n1"},
		{name: "line CR", input: "// c\r1", want: "    \r1"},
		{name: "line CRLF", input: "// c\r\n1", want: "    \r\n1"},
		{name: "line EOF", input: "1// eof", want: "1      "},
		{name: "line backslash does not continue", input: "// a\\\n1", want: "     \n1"},
		{name: "empty block", input: "/**/1", want: "    1"},
		{name: "repeated block stars", input: "/****/1", want: "      1"},
		{
			name:  "block preserves CRLF",
			input: "/* line\r\nnext */1",
			want:  "       \r\n       1",
		},
		{
			name:  "block does not nest",
			input: "/* outer /* inner */ 1",
			want:  strings.Repeat(" ", len("/* outer /* inner */")) + " 1",
		},
		{
			name:  "unicode line body including U+2028",
			input: "// π x\n1",
			want:  strings.Repeat(" ", len("// π x")) + "\n1",
		},
		{
			name:  "unicode block body including NUL",
			input: "/*\x00π*/1",
			want:  strings.Repeat(" ", len("/*\x00π*/")) + "1",
		},
		{name: "token split literal", input: "tr/*x*/ue", want: "tr     ue"},
		{name: "token split number", input: "1/*x*/.5", want: "1     .5"},
		{name: "token split line", input: "nu//x\nll", want: "nu   \nll"},
		{name: "lone slash preserved", input: "1/", want: "1/"},
		{name: "lone star preserved", input: "1*", want: "1*"},
		{name: "stray block closer preserved", input: "*/1", want: "*/1"},
		{name: "array trailing comma", input: "[1,]", want: "[1 ]"},
		{name: "object trailing comma", input: `{"a":1,}`, want: `{"a":1 }`},
		{
			name:  "nested trailing commas",
			input: `[{"a":[1,],},]`,
			want:  `[{"a":[1 ] } ]`,
		},
		{
			name:  "comment after trailing comma",
			input: "[1, /* note */]",
			want:  "[1" + strings.Repeat(" ", 12) + "]",
		},
		{
			name:  "adjacent comment after trailing comma",
			input: "[1,/*c*/]",
			want:  "[1" + strings.Repeat(" ", 6) + "]",
		},
		{name: "empty array comma stays invalid", input: "[,]", want: "[,]"},
		{name: "empty object comma stays invalid", input: "{,}", want: "{,}"},
		{name: "repeated array comma stays invalid", input: "[1,,]", want: "[1,,]"},
		{name: "repeated object comma stays invalid", input: `{"a":1,,}`, want: `{"a":1,,}`},
		{name: "wrong closer stays invalid", input: "[1,}", want: "[1,}"},
		{name: "root comma stays invalid", input: "1,", want: "1,"},
		{name: "object key without value stays invalid", input: `{"a",}`, want: `{"a",}`},
		{name: "non JSON whitespace cancels trailing comma", input: "[1,\v]", want: "[1,\v]"},
		{
			name:  "comment does not make empty comma valid",
			input: "[/*c*/,]",
			want:  "[     ,]",
		},
		{
			name:  "comments do not hide repeated comma",
			input: "[1,/*c*/,]",
			want:  "[1,     ,]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(tt.input)
			before := bytes.Clone(input)

			got, err := Sanitize(input)
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("Sanitize() = %q, want %q", got, tt.want)
			}
			if len(got) != len(input) {
				t.Fatalf("Sanitize() length = %d, want %d", len(got), len(input))
			}
			if !bytes.Equal(input, before) {
				t.Fatalf("Sanitize() mutated input: got %q, want %q", input, before)
			}
		})
	}
}

func TestSanitizeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		want    error
		offset  int64
		message string
	}{
		{
			name:    "unterminated at start",
			input:   []byte("/*"),
			want:    ErrUnterminatedBlockComment,
			offset:  1,
			message: "jsonc: unterminated block comment at byte 1",
		},
		{
			name:    "unterminated after value",
			input:   []byte("1 /* missing"),
			want:    ErrUnterminatedBlockComment,
			offset:  3,
			message: "jsonc: unterminated block comment at byte 3",
		},
		{
			name:    "malformed UTF-8 in line comment",
			input:   []byte{'/', '/', 0xff, '\n', '1'},
			want:    ErrInvalidUTF8,
			offset:  3,
			message: "jsonc: invalid UTF-8 in comment at byte 3",
		},
		{
			name:    "malformed UTF-8 in block comment",
			input:   []byte{'/', '*', 0xff, '*', '/', '1'},
			want:    ErrInvalidUTF8,
			offset:  3,
			message: "jsonc: invalid UTF-8 in comment at byte 3",
		},
		{
			name:    "malformed UTF-8 after multibyte rune",
			input:   append([]byte("/*π"), 0xff, '*', '/'),
			want:    ErrInvalidUTF8,
			offset:  5,
			message: "jsonc: invalid UTF-8 in comment at byte 5",
		},
		{
			name:    "malformed UTF-8 takes precedence while scanning unterminated block",
			input:   []byte{'/', '*', 0xff},
			want:    ErrInvalidUTF8,
			offset:  3,
			message: "jsonc: invalid UTF-8 in comment at byte 3",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Sanitize(tt.input)
			if got != nil {
				t.Fatalf("Sanitize() output = %q, want nil", got)
			}
			if !errors.Is(err, tt.want) {
				t.Fatalf("errors.Is(%v, %v) = false", err, tt.want)
			}
			var syntaxErr *JSONCSyntaxError
			if !errors.As(err, &syntaxErr) {
				t.Fatalf("errors.As(%T, *JSONCSyntaxError) = false", err)
			}
			if syntaxErr.Offset != tt.offset {
				t.Fatalf("JSONCSyntaxError.Offset = %d, want %d", syntaxErr.Offset, tt.offset)
			}
			if err.Error() != tt.message {
				t.Fatalf("error = %q, want %q", err, tt.message)
			}
		})
	}
}

func TestSanitizeMalformedUTF8OutsideComments(t *testing.T) {
	t.Parallel()

	tests := [][]byte{
		{'[', '"', 0xff, '"', ']'},
		{0xff, '/', '/', 'o', 'k'},
	}
	for _, input := range tests {
		input := input
		t.Run(strings.ReplaceAll(string(input), "/", "slash"), func(t *testing.T) {
			t.Parallel()
			before := bytes.Clone(input)
			got, err := Sanitize(input)
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}
			if len(got) != len(input) {
				t.Fatalf("Sanitize() length = %d, want %d", len(got), len(input))
			}
			if input[0] == 0xff {
				want := append([]byte{0xff}, []byte("    ")...)
				if !bytes.Equal(got, want) {
					t.Fatalf("Sanitize() = %v, want %v", got, want)
				}
			} else if !bytes.Equal(got, input) {
				t.Fatalf("Sanitize() = %v, want %v", got, input)
			}
			if !bytes.Equal(input, before) {
				t.Fatalf("Sanitize() mutated input: got %v, want %v", input, before)
			}
		})
	}
}

func TestSanitizeTokenSplitsRemainInvalid(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		"tr/*x*/ue",
		"1/*x*/.5",
		"nu//x\nll",
		"[1,,]",
		"[,]",
		"{,}",
		"[1,}",
	} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, err := Sanitize([]byte(input))
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}
			if stdjson.Valid(got) {
				t.Fatalf("Sanitize(%q) repaired invalid input to %q", input, got)
			}
		})
	}
}

func TestSanitizeFixtures(t *testing.T) {
	t.Parallel()

	var small Small
	got, err := Sanitize(_small)
	if err != nil {
		t.Fatalf("Sanitize(small) error = %v", err)
	}
	if err := stdjson.Unmarshal(got, &small); err != nil {
		t.Fatalf("Unmarshal(sanitized small) error = %v", err)
	}
	FieldsValue(t, small)

	var medium Medium
	got, err = Sanitize(_medium)
	if err != nil {
		t.Fatalf("Sanitize(medium) error = %v", err)
	}
	if err := stdjson.Unmarshal(got, &medium); err != nil {
		t.Fatalf("Unmarshal(sanitized medium) error = %v", err)
	}
	FieldsValue(t, medium)
}

func TestSanitizeStrictFastPathAllocations(t *testing.T) {
	input := []byte(`{"url":"https://x","comment":"/* text */","values":[1,2,3]}`)
	got, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	if !bytes.Equal(got, input) {
		t.Fatalf("Sanitize() = %q, want unchanged %q", got, input)
	}

	allocs := testing.AllocsPerRun(1_000, func() {
		got, err = Sanitize(input)
	})
	runtime.KeepAlive(got)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	if allocs != 0 {
		t.Fatalf("Sanitize(strict JSON) allocations = %g, want 0", allocs)
	}
}

func BenchmarkSanitize(b *testing.B) {
	benchmarks := []struct {
		name string
		data []byte
	}{
		{name: "Strict", data: _mediumUncommented},
		{name: "Commented", data: _medium},
		{name: "TrailingComma", data: []byte(`{"value":[1,2,3,],}`)},
	}

	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			var output []byte
			var err error
			for i := 0; i < b.N; i++ {
				output, err = Sanitize(benchmark.data)
			}
			if err != nil {
				b.Fatalf("Sanitize() error = %v", err)
			}
			runtime.KeepAlive(output)
		})
	}
}

func TestSanitizeOutputMatchesExpectedValue(t *testing.T) {
	t.Parallel()

	input := []byte(`{"a":1,"b":[true,null,],}`)
	got, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	var value map[string]any
	if err := stdjson.Unmarshal(got, &value); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := map[string]any{"a": float64(1), "b": []any{true, nil}}
	if !reflect.DeepEqual(value, want) {
		t.Fatalf("value = %#v, want %#v", value, want)
	}
}
