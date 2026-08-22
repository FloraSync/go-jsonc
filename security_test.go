// Copyright 2026 FloraSync. All Rights Reserved.
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

package json

import (
	"bytes"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestDecoderRejectsInvalidReaderCounts(t *testing.T) {
	tests := []struct {
		name   string
		reader *invalidCountReader
	}{
		{name: "larger than buffer", reader: &invalidCountReader{adjustment: 1}},
		{name: "negative", reader: &invalidCountReader{returnNegative: true}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			decoder := NewDecoder(test.reader)
			var value any

			firstErr := decoder.Decode(&value)
			want := fmt.Sprintf("jsonc: reader returned invalid count %d", test.reader.returned)
			if firstErr == nil || firstErr.Error() != want {
				t.Fatalf("first Decode() error = %v, want %q", firstErr, want)
			}

			secondErr := decoder.Decode(&value)
			if secondErr != firstErr {
				t.Fatalf("second Decode() error = %v, want same sticky error %v", secondErr, firstErr)
			}
		})
	}
}

type invalidCountReader struct {
	adjustment     int
	returnNegative bool
	returned       int
}

func (r *invalidCountReader) Read(data []byte) (int, error) {
	if r.returnNegative {
		r.returned = -1
		return r.returned, nil
	}
	r.returned = len(data) + r.adjustment
	return r.returned, nil
}

func TestSliceNormalizerTinyInputs(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{name: "nil", input: nil},
		{name: "empty", input: []byte{}},
		{name: "lone slash", input: []byte(`/`)},
		{name: "line opener at EOF", input: []byte(`//`)},
		{name: "block opener at EOF", input: []byte(`/*`), wantErr: ErrUnterminatedBlockComment},
		{name: "stray block closer", input: []byte(`*/`)},
		{name: "repeated stars", input: []byte(`/****/`)},
		{name: "incomplete string escape", input: []byte(`"\`)},
		{name: "truncated comment rune", input: []byte{'/', '*', 0xe2}, wantErr: ErrInvalidUTF8},
		{name: "binary outside comment", input: []byte{0xff, '/', '*', '*', '/'}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			before := bytes.Clone(test.input)
			_ = HasCommentRunes(test.input)
			_ = Valid(test.input)
			_ = stdjson.Valid(test.input)

			got, err := Sanitize(test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Sanitize() error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if err == nil && len(got) != len(test.input) {
				t.Fatalf("Sanitize() length = %d, want %d", len(got), len(test.input))
			}
			if !bytes.Equal(test.input, before) {
				t.Fatalf("Sanitize() mutated input: got %v, want %v", test.input, before)
			}
		})
	}
}

func TestNormalizationAtNestingLimit(t *testing.T) {
	tests := []struct {
		name              string
		depth             int
		wantNormalized    bool
		wantStandardValid bool
	}{
		{name: "below limit", depth: maxTrackedNestingDepth - 1, wantNormalized: true, wantStandardValid: true},
		{name: "at limit", depth: maxTrackedNestingDepth, wantNormalized: true, wantStandardValid: true},
		{name: "above limit", depth: maxTrackedNestingDepth + 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			input := make([]byte, 0, test.depth*2+2)
			input = append(input, bytes.Repeat([]byte{'['}, test.depth)...)
			input = append(input, '0', ',')
			commaOffset := len(input) - 1
			input = append(input, bytes.Repeat([]byte{']'}, test.depth)...)

			got, err := Sanitize(input)
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}
			if normalized := got[commaOffset] == ' '; normalized != test.wantNormalized {
				t.Fatalf("trailing comma normalized = %v, want %v", normalized, test.wantNormalized)
			}
			if valid := stdjson.Valid(got); valid != test.wantStandardValid {
				t.Fatalf("encoding/json.Valid() = %v, want %v", valid, test.wantStandardValid)
			}

			streamed, streamErr := io.ReadAll(newStreamNormalizer(bytes.NewReader(input)))
			if streamErr != nil {
				t.Fatalf("stream normalization error = %v", streamErr)
			}
			if !bytes.Equal(streamed, got) {
				t.Fatal("slice and stream normalization differ at nesting boundary")
			}
		})
	}
}

func TestStreamNormalizerReturnsTemporaryNoProgress(t *testing.T) {
	input := []byte("// pause\n1")
	source := &scriptedReader{results: []readResult{
		{data: []byte(`/`)},
		{},
		{data: []byte("/ pause\n1"), err: io.EOF},
	}}
	normalizer := newStreamNormalizer(source)

	buffer := make([]byte, 8)
	count, err := normalizer.Read(buffer)
	if count != 0 || err != nil {
		t.Fatalf("Read() = %d, %v, want 0, nil", count, err)
	}
	if len(source.results) != 1 {
		t.Fatalf("Read() consumed %d scripted results after no progress, want 2", 3-len(source.results))
	}

	got, err := io.ReadAll(normalizer)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	want, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("stream normalization = %q, want %q", got, want)
	}
}

func TestStreamNormalizerResumesStateAfterTemporaryNoProgress(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		results []readResult
	}{
		{
			name:  "block comment UTF-8",
			input: []byte("/*€*/1"),
			results: []readResult{
				{data: []byte{'/', '*', 0xe2}},
				{},
				{data: []byte{0x82, 0xac, '*', '/', '1'}, err: io.EOF},
			},
		},
		{
			name:  "pending comma",
			input: []byte(`[1,/* tail */]`),
			results: []readResult{
				{data: []byte(`[1,`)},
				{},
				{data: []byte(`/* tail */]`), err: io.EOF},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			got, err := io.ReadAll(newStreamNormalizer(&scriptedReader{results: test.results}))
			if err != nil {
				t.Fatalf("stream normalization error = %v", err)
			}
			want, err := Sanitize(test.input)
			if err != nil {
				t.Fatalf("Sanitize() error = %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("stream normalization = %q, want %q", got, want)
			}
		})
	}
}

func TestStreamNormalizerSourceErrorPrecedence(t *testing.T) {
	sourceErr := errors.New("source failed")
	tests := []struct {
		name    string
		input   []byte
		want    []byte
		wantErr error
	}{
		{name: "pending slash", input: []byte(`/`), want: []byte(`/`), wantErr: sourceErr},
		{name: "pending comma", input: []byte(`[1,`), want: []byte(`[1,`), wantErr: sourceErr},
		{name: "block star", input: []byte(`/**`), want: []byte(`   `), wantErr: sourceErr},
		{name: "incomplete UTF-8", input: []byte{'/', '*', 0xe2}, want: []byte(`   `), wantErr: sourceErr},
		{name: "lexical error wins", input: []byte{'/', '*', 0xff}, want: []byte(`  `), wantErr: ErrInvalidUTF8},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			source := &scriptedReader{results: []readResult{{data: test.input, err: sourceErr}}}
			got, err := io.ReadAll(newStreamNormalizer(source))
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("stream normalization error = %v, want errors.Is(_, %v)", err, test.wantErr)
			}
			if !bytes.Equal(got, test.want) {
				t.Fatalf("stream normalization = %q, want %q", got, test.want)
			}
		})
	}
}

func TestStreamNormalizerReleasesPendingBufferOnLexicalError(t *testing.T) {
	const bodySize = 1 << 20
	input := make([]byte, 0, bodySize+6)
	input = append(input, "[1,/*"...)
	input = append(input, bytes.Repeat([]byte{'x'}, bodySize)...)
	input = append(input, 0xff)

	normalizer := newStreamNormalizer(bytes.NewReader(input))
	_, err := io.ReadAll(normalizer)
	if !errors.Is(err, ErrInvalidUTF8) {
		t.Fatalf("stream normalization error = %v, want %v", err, ErrInvalidUTF8)
	}
	if normalizer.commaPending {
		t.Fatal("terminal lexical error retained pending-comma state")
	}
	if normalizer.pending != nil {
		t.Fatalf("terminal lexical error retained %d bytes with capacity %d", len(normalizer.pending), cap(normalizer.pending))
	}
}

func TestDecoderLexicalErrorsAreStickyAcrossOperations(t *testing.T) {
	decoder := NewDecoder(bytes.NewReader([]byte(`/* open`)))
	_, tokenErr := decoder.Token()
	if !errors.Is(tokenErr, ErrUnterminatedBlockComment) {
		t.Fatalf("Token() error = %v, want %v", tokenErr, ErrUnterminatedBlockComment)
	}
	offset := decoder.InputOffset()

	var value any
	decodeErr := decoder.Decode(&value)
	if !errors.Is(decodeErr, ErrUnterminatedBlockComment) {
		t.Fatalf("Decode() error = %v, want %v", decodeErr, ErrUnterminatedBlockComment)
	}
	if decoder.InputOffset() != offset {
		t.Fatalf("InputOffset() = %d after sticky error, want %d", decoder.InputOffset(), offset)
	}
	if _, err := io.ReadAll(decoder.Buffered()); err != nil {
		t.Fatalf("Buffered() error = %v", err)
	}
}

type readResult struct {
	data []byte
	err  error
}

type scriptedReader struct {
	results []readResult
}

func (r *scriptedReader) Read(data []byte) (int, error) {
	if len(r.results) == 0 {
		return 0, io.EOF
	}

	result := &r.results[0]
	count := copy(data, result.data)
	result.data = result.data[count:]
	if len(result.data) != 0 {
		return count, nil
	}

	err := result.err
	r.results = r.results[1:]
	return count, err
}
