// Copyright 2023 Marco Zaccaro. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json_test

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/FloraSync/go-jsonc"
)

func TestStrictTopLevelFacadeMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	inputs := [][]byte{
		[]byte(`{"name":"flora","values":[1,true,null],"nested":{"x":"y"}}`),
		[]byte(`"https://example.test/a/*literal*/"`),
		[]byte(`"\ud800"`),
		[]byte{`"`[0], 0xff, `"`[0]},
		[]byte(`{"broken":]`),
		[]byte(`1 2`),
	}

	for _, input := range inputs {
		input := input
		t.Run(string(input), func(t *testing.T) {
			var want any
			wantErr := stdjson.Unmarshal(input, &want)

			var got any
			gotErr := json.Unmarshal(input, &got)

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Unmarshal value mismatch:\n got: %#v\nwant: %#v", got, want)
			}
			assertMatchingError(t, gotErr, wantErr)
			if got, want := json.Valid(input), stdjson.Valid(input); got != want {
				t.Fatalf("Valid() = %v, want %v", got, want)
			}
		})
	}
}

func TestEncodingFacadeMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	value := struct {
		HTML   string         `json:"html"`
		Values map[string]any `json:"values"`
	}{
		HTML:   "<script>&\u2028",
		Values: map[string]any{"nil": nil, "number": 42.5},
	}

	got, gotErr := json.Marshal(value)
	want, wantErr := stdjson.Marshal(value)
	assertMatchingError(t, gotErr, wantErr)
	if !bytes.Equal(got, want) {
		t.Fatalf("Marshal() = %q, want %q", got, want)
	}

	got, gotErr = json.MarshalIndent(value, "P", "  ")
	want, wantErr = stdjson.MarshalIndent(value, "P", "  ")
	assertMatchingError(t, gotErr, wantErr)
	if !bytes.Equal(got, want) {
		t.Fatalf("MarshalIndent() = %q, want %q", got, want)
	}

	_, gotErr = json.Marshal(make(chan int))
	_, wantErr = stdjson.Marshal(make(chan int))
	assertMatchingError(t, gotErr, wantErr)
}

func TestEncoderFacadeMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	values := []any{
		map[string]any{"html": "<tag>", "n": 1},
		[]string{"a", "b"},
	}

	var got bytes.Buffer
	gotEncoder := json.NewEncoder(&got)
	gotEncoder.SetEscapeHTML(false)
	gotEncoder.SetIndent("P", "  ")

	var want bytes.Buffer
	wantEncoder := stdjson.NewEncoder(&want)
	wantEncoder.SetEscapeHTML(false)
	wantEncoder.SetIndent("P", "  ")

	for _, value := range values {
		assertMatchingError(t, gotEncoder.Encode(value), wantEncoder.Encode(value))
	}
	if got.String() != want.String() {
		t.Fatalf("encoded stream mismatch:\n got: %q\nwant: %q", got.String(), want.String())
	}
}

func TestFormattingFacadeUsesNormalizedView(t *testing.T) {
	t.Parallel()

	source := []byte("/*head*/ {\"a\": [1, /*tail*/],}\n")
	normalized, err := json.Sanitize(source)
	if err != nil {
		t.Fatalf("Sanitize() error: %v", err)
	}

	for _, test := range []struct {
		name string
		run  func(*bytes.Buffer, []byte) error
	}{
		{name: "compact", run: json.Compact},
		{name: "indent", run: func(dst *bytes.Buffer, src []byte) error {
			return json.Indent(dst, src, "P", "  ")
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			got := bytes.NewBufferString("prefix:")
			if err := test.run(got, source); err != nil {
				t.Fatalf("JSONC formatter error: %v", err)
			}

			want := bytes.NewBufferString("prefix:")
			var err error
			if test.name == "compact" {
				err = stdjson.Compact(want, normalized)
			} else {
				err = stdjson.Indent(want, normalized, "P", "  ")
			}
			if err != nil {
				t.Fatalf("standard formatter oracle error: %v", err)
			}
			if got.String() != want.String() {
				t.Fatalf("formatter output = %q, want %q", got.String(), want.String())
			}
		})
	}
}

func TestHTMLEscapeMatchesEncodingJSON(t *testing.T) {
	t.Parallel()

	source := []byte(`{"html":"<script>&\u2028"}`)
	got := bytes.NewBufferString("prefix:")
	json.HTMLEscape(got, source)
	want := bytes.NewBufferString("prefix:")
	stdjson.HTMLEscape(want, source)
	if got.String() != want.String() {
		t.Fatalf("HTMLEscape() = %q, want %q", got.String(), want.String())
	}
}

func TestValidStrictFastPathDoesNotAllocate(t *testing.T) {
	input := []byte(`{"url":"https://example.test/a/*literal*/","n":1}`)
	if !json.Valid(input) {
		t.Fatal("strict fixture is unexpectedly invalid")
	}

	if got := testing.AllocsPerRun(100, func() {
		if !json.Valid(input) {
			panic("unexpected invalid result")
		}
	}); got != 0 {
		t.Fatalf("Valid(strict JSON) allocated %v times per call, want 0", got)
	}
}

func assertMatchingError(t *testing.T, got, want error) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("error mismatch: got %v, want %v", got, want)
	}
	if got == nil {
		return
	}
	if reflect.TypeOf(got) != reflect.TypeOf(want) {
		t.Fatalf("error type = %T, want %T", got, want)
	}
	if got.Error() != want.Error() {
		t.Fatalf("error text = %q, want %q", got, want)
	}

	var gotSyntax *stdjson.SyntaxError
	var wantSyntax *stdjson.SyntaxError
	if errors.As(got, &gotSyntax) && errors.As(want, &wantSyntax) && gotSyntax.Offset != wantSyntax.Offset {
		t.Fatalf("syntax offset = %d, want %d", gotSyntax.Offset, wantSyntax.Offset)
	}
}
