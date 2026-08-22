// Copyright 2023 Marco Zaccaro. All Rights Reserved.
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

//go:build !uncommented_test

package json

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"reflect"
	"runtime"
	"testing"
)

func TestUnmarshalFixtures(t *testing.T) {
	t.Parallel()

	UnmarshalOK(t, _small, Small{})
	UnmarshalOK(t, _medium, Medium{})
}

func UnmarshalOK[T DataType](t testing.TB, data []byte, initial T) {
	t.Helper()
	got := initial
	if err := Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	FieldsValue(t, got)
}

func TestUnmarshalJSONCProfile(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		// line
		"array": [1, 2, /* block */],
		"url": "https://example.test/a/*b*/",
	}`)
	var got struct {
		Array []int  `json:"array"`
		URL   string `json:"url"`
	}
	if err := Unmarshal(input, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := struct {
		Array []int  `json:"array"`
		URL   string `json:"url"`
	}{
		Array: []int{1, 2},
		URL:   "https://example.test/a/*b*/",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unmarshal() = %#v, want %#v", got, want)
	}
}

func TestUnmarshalStrictBehaviorMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{name: "object", data: []byte(`{"a":1,"b":[true,null,"x"]}`)},
		{name: "top-level scalar", data: []byte(`123.5`)},
		{name: "invalid UTF-8 in string", data: []byte{'{', '"', 's', '"', ':', '"', 0xff, '"', '}'}},
		{name: "invalid trailing data", data: []byte(`{"a":1} false`)},
		{name: "invalid token", data: []byte(`tru`)},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got any
			var want any
			gotErr := Unmarshal(tt.data, &got)
			wantErr := stdjson.Unmarshal(tt.data, &want)
			if !sameError(gotErr, wantErr) {
				t.Fatalf("Unmarshal() error = %T %v, encoding/json = %T %v", gotErr, gotErr, wantErr, wantErr)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Unmarshal() value = %#v, encoding/json = %#v", got, want)
			}
		})
	}
}

func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return reflect.TypeOf(got) == reflect.TypeOf(want) && got.Error() == want.Error()
}

func TestUnmarshalMalformedUTF8OutsideCommentUsesEncodingJSON(t *testing.T) {
	t.Parallel()

	input := append(bytes.Clone(_small), _invalidChar...)
	var got Small
	err := Unmarshal(input, &got)
	if err == nil {
		t.Fatal("Unmarshal() error = nil, want standard syntax error")
	}
	if errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("Unmarshal() error = %v, malformed byte is outside a comment", err)
	}

	normalized, sanitizeErr := Sanitize(input)
	if sanitizeErr != nil {
		t.Fatalf("Sanitize() error = %v", sanitizeErr)
	}
	var want Small
	wantErr := stdjson.Unmarshal(normalized, &want)
	if !sameError(err, wantErr) {
		t.Fatalf("Unmarshal() error = %T %v, encoding/json = %T %v", err, err, wantErr, wantErr)
	}
}

func TestUnmarshalJSONCErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		want error
	}{
		{name: "unterminated block", data: []byte(`{"a":1} /*`), want: ErrUnterminatedBlockComment},
		{name: "invalid line UTF-8", data: []byte{'/', '/', 0xff, '\n', '1'}, want: ErrInvalidUTF8},
		{name: "invalid block UTF-8", data: []byte{'/', '*', 0xff, '*', '/', '1'}, want: ErrInvalidUTF8},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var destination any
			err := Unmarshal(tt.data, &destination)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Unmarshal() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
		})
	}
}

type recordingUnmarshaler struct {
	data []byte
}

func (r *recordingUnmarshaler) UnmarshalJSON(data []byte) error {
	r.data = append(r.data[:0], data...)
	return nil
}

func TestUnmarshalCallbacksReceiveNormalizedJSON(t *testing.T) {
	t.Parallel()

	input := []byte(`{"value":[1,/*c*/]}`)
	var got struct {
		Value recordingUnmarshaler `json:"value"`
	}
	if err := Unmarshal(input, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := "[1" + string(bytes.Repeat([]byte{' '}, 6)) + "]"
	if string(got.Value.data) != want {
		t.Fatalf("UnmarshalJSON() data = %q, want %q", got.Value.data, want)
	}

	var raw RawMessage
	if err := Unmarshal([]byte(`/*c*/[1,]`), &raw); err != nil {
		t.Fatalf("Unmarshal(RawMessage) error = %v", err)
	}
	if string(raw) != "[1 ]" {
		t.Fatalf("RawMessage = %q, want %q", raw, "[1 ]")
	}
}

func TestUnmarshalSyntaxOffsetMapsToOriginalInput(t *testing.T) {
	t.Parallel()

	input := []byte("{\n// c\n\"a\": nope\n}")
	normalized, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	var got any
	gotErr := Unmarshal(input, &got)
	var want any
	wantErr := stdjson.Unmarshal(normalized, &want)
	if !sameError(gotErr, wantErr) {
		t.Fatalf("Unmarshal() error = %T %v, encoding/json = %T %v", gotErr, gotErr, wantErr, wantErr)
	}
	var syntaxErr *SyntaxError
	if !errors.As(gotErr, &syntaxErr) {
		t.Fatalf("Unmarshal() error = %T, want *SyntaxError", gotErr)
	}
	if syntaxErr.Offset <= 0 || syntaxErr.Offset > int64(len(input)) {
		t.Fatalf("SyntaxError.Offset = %d, input length = %d", syntaxErr.Offset, len(input))
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	benchmarks := []struct {
		name string
		data []byte
	}{
		{name: "Small/Commented", data: _small},
		{name: "Small/Strict", data: _smallUncommented},
		{name: "Medium/Commented", data: _medium},
		{name: "Medium/Strict", data: _mediumUncommented},
	}

	for _, benchmark := range benchmarks {
		benchmark := benchmark
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				var destination any
				if err := Unmarshal(benchmark.data, &destination); err != nil {
					b.Fatalf("Unmarshal() error = %v", err)
				}
				runtime.KeepAlive(destination)
			}
		})
	}
}
