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
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestStreamNormalizerMatchesSanitizeAcrossChunks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "strict", input: []byte(`{"url":"https://example.com/a/*b*/"}`)},
		{name: "line comments", input: []byte("// π  remains a comment\r\n1")},
		{name: "block comment", input: []byte("/***/\r\n1")},
		{name: "unicode block comment", input: []byte("[1,/* 🌿 */]")},
		{name: "nested trailing commas", input: []byte(`[{"a":[true,],},]`)},
		{name: "comments around trailing comma", input: []byte(`{"a":1/* value */,/* tail */}`)},
		{name: "invalid empty comma", input: []byte(`[/* no value */,]`)},
		{name: "double comma", input: []byte(`[1,,]`)},
		{name: "token splitting", input: []byte(`tr/* split */ue`)},
		{name: "escaped quote parity", input: []byte(`{"s":"\\\"// not a comment"}`)},
		{name: "standalone slash", input: []byte(`/`)},
	}

	readers := []struct {
		name string
		new  func([]byte) io.Reader
	}{
		{name: "buffered", new: func(data []byte) io.Reader { return bytes.NewReader(data) }},
		{name: "one byte", new: func(data []byte) io.Reader {
			return iotest.OneByteReader(bytes.NewReader(data))
		}},
		{name: "half", new: func(data []byte) io.Reader {
			return iotest.HalfReader(bytes.NewReader(data))
		}},
		{name: "data and EOF", new: func(data []byte) io.Reader {
			return iotest.DataErrReader(bytes.NewReader(data))
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			want, err := Sanitize(test.input)
			if err != nil {
				t.Fatalf("Sanitize returned unexpected error: %v", err)
			}
			for _, reader := range readers {
				reader := reader
				t.Run(reader.name, func(t *testing.T) {
					t.Parallel()
					got, err := io.ReadAll(newStreamNormalizer(reader.new(test.input)))
					if err != nil {
						t.Fatalf("stream normalization returned error: %v", err)
					}
					if !bytes.Equal(got, want) {
						t.Fatalf("normalized bytes differ\n got: %q\nwant: %q", got, want)
					}
					if len(got) != len(test.input) {
						t.Fatalf("length changed: got %d, want %d", len(got), len(test.input))
					}
				})
			}
		})
	}
}

func TestStreamNormalizerErrorsAcrossChunks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  []byte
		kind   error
		offset int64
	}{
		{name: "unterminated block", input: []byte(`/* open`), kind: ErrUnterminatedBlockComment, offset: 1},
		{name: "invalid line UTF-8", input: []byte{'/', '/', ' ', 0xc2, '\n', '1'}, kind: ErrInvalidUTF8, offset: 4},
		{name: "invalid block UTF-8", input: []byte{'/', '*', 0xff, '*', '/', '1'}, kind: ErrInvalidUTF8, offset: 3},
		{name: "truncated UTF-8 in block", input: []byte{'/', '*', 0xe2}, kind: ErrInvalidUTF8, offset: 3},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			for _, reader := range []io.Reader{
				bytes.NewReader(test.input),
				iotest.OneByteReader(bytes.NewReader(test.input)),
				iotest.DataErrReader(bytes.NewReader(test.input)),
			} {
				_, err := io.ReadAll(newStreamNormalizer(reader))
				if !errors.Is(err, test.kind) {
					t.Fatalf("error = %v, want errors.Is(_, %v)", err, test.kind)
				}
				var syntaxErr *JSONCSyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Fatalf("error type = %T, want *JSONCSyntaxError", err)
				}
				if syntaxErr.Offset != test.offset {
					t.Fatalf("offset = %d, want %d", syntaxErr.Offset, test.offset)
				}
			}
		})
	}
}

func TestStreamNormalizerTransfersLargePendingBuffer(t *testing.T) {
	t.Parallel()

	const bodySize = 1 << 20
	input := make([]byte, 0, bodySize+8)
	input = append(input, "[1,/*"...)
	input = append(input, bytes.Repeat([]byte{'x'}, bodySize)...)
	input = append(input, "*/]"...)

	normalizer := newStreamNormalizer(bytes.NewReader(input))
	buffer := make([]byte, 4<<10)
	var got bytes.Buffer

	count, err := normalizer.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	got.Write(buffer[:count])
	if got.String() != "[1" {
		t.Fatalf("first committed prefix = %q, want %q", got.Bytes(), "[1")
	}

	count, err = normalizer.Read(buffer)
	if err != nil {
		t.Fatal(err)
	}
	got.Write(buffer[:count])

	largeBuffers := 0
	if cap(normalizer.ready) > bodySize/2 {
		largeBuffers++
	}
	if cap(normalizer.pending) > bodySize/2 {
		largeBuffers++
	}
	if largeBuffers != 1 {
		t.Fatalf("large normalizer buffers = %d, want 1 (ready cap %d, pending cap %d)",
			largeBuffers, cap(normalizer.ready), cap(normalizer.pending))
	}
	if normalizer.pending != nil {
		t.Fatalf("resolved pending allocation was retained with capacity %d", cap(normalizer.pending))
	}

	rest, err := io.ReadAll(normalizer)
	if err != nil {
		t.Fatal(err)
	}
	got.Write(rest)
	want, err := Sanitize(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatal("ownership transfer changed normalized output")
	}
}

func TestDecoderTokenMoreAndOffsetsMatchNormalizedStandard(t *testing.T) {
	t.Parallel()

	input := []byte(`/* head */{"n":[1,/* array tail */],/* object tail */}`)
	normalized, err := Sanitize(input)
	if err != nil {
		t.Fatal(err)
	}

	got := NewDecoder(iotest.OneByteReader(bytes.NewReader(input)))
	want := stdjson.NewDecoder(iotest.OneByteReader(bytes.NewReader(normalized)))
	got.UseNumber()
	want.UseNumber()

	compareToken := func() {
		t.Helper()
		gotToken, gotErr := got.Token()
		wantToken, wantErr := want.Token()
		if !sameDecoderError(gotErr, wantErr) {
			t.Fatalf("Token errors differ: got %v, want %v", gotErr, wantErr)
		}
		if !reflect.DeepEqual(gotToken, wantToken) {
			t.Fatalf("Token differs: got %T(%v), want %T(%v)", gotToken, gotToken, wantToken, wantToken)
		}
		if got.InputOffset() != want.InputOffset() {
			t.Fatalf("offset after Token = %d, want %d", got.InputOffset(), want.InputOffset())
		}
	}
	compareMore := func() bool {
		t.Helper()
		gotMore := got.More()
		wantMore := want.More()
		if gotMore != wantMore {
			t.Fatalf("More = %v, want %v", gotMore, wantMore)
		}
		if got.InputOffset() != want.InputOffset() {
			t.Fatalf("offset after More = %d, want %d", got.InputOffset(), want.InputOffset())
		}
		return gotMore
	}

	compareToken() // {
	if !compareMore() {
		t.Fatal("object unexpectedly empty")
	}
	compareToken() // n
	compareToken() // [
	if !compareMore() {
		t.Fatal("array unexpectedly empty")
	}
	var gotNumber, wantNumber any
	gotErr := got.Decode(&gotNumber)
	wantErr := want.Decode(&wantNumber)
	if !sameDecoderError(gotErr, wantErr) || !reflect.DeepEqual(gotNumber, wantNumber) {
		t.Fatalf("Decode differs: got %T(%v), %v; want %T(%v), %v",
			gotNumber, gotNumber, gotErr, wantNumber, wantNumber, wantErr)
	}
	if got.InputOffset() != want.InputOffset() {
		t.Fatalf("offset after Decode = %d, want %d", got.InputOffset(), want.InputOffset())
	}
	if compareMore() {
		t.Fatal("trailing comma was exposed as another array value")
	}
	compareToken() // ]
	if compareMore() {
		t.Fatal("trailing comma was exposed as another object member")
	}
	compareToken() // }
	compareToken() // EOF
}

func TestDecoderOptionsPersistAcrossValues(t *testing.T) {
	t.Parallel()

	type target struct {
		OK Number `json:"ok"`
	}

	decoder := NewDecoder(iotest.OneByteReader(strings.NewReader(
		`{"bad":1,} /* between */ {"ok":9007199254740993,}`,
	)))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()

	var first target
	if err := decoder.Decode(&first); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("first Decode error = %v, want unknown-field error", err)
	}

	var second target
	if err := decoder.Decode(&second); err != nil {
		t.Fatalf("second Decode returned sticky semantic error: %v", err)
	}
	if second.OK != Number("9007199254740993") {
		t.Fatalf("UseNumber did not persist: got %q", second.OK)
	}
}

func TestDecoderLexicalErrorsAreSticky(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		kind  error
	}{
		{name: "unterminated", input: []byte(`/* open`), kind: ErrUnterminatedBlockComment},
		{name: "invalid UTF-8", input: []byte{'/', '/', 0xff}, kind: ErrInvalidUTF8},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			decoder := NewDecoder(iotest.OneByteReader(bytes.NewReader(test.input)))
			for attempt := 0; attempt < 2; attempt++ {
				var value any
				err := decoder.Decode(&value)
				if !errors.Is(err, test.kind) {
					t.Fatalf("Decode attempt %d error = %v, want %v", attempt+1, err, test.kind)
				}
			}
		})
	}
}

func TestDecoderDefersInvalidTrailingData(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(strings.NewReader(`{} /* open`))
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("first Decode = %v, want nil", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		err := decoder.Decode(&value)
		if !errors.Is(err, ErrUnterminatedBlockComment) {
			t.Fatalf("trailing Decode attempt %d error = %v", attempt+1, err)
		}
	}
}

func TestDecoderPreservesUnderlyingDataError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("reader failed")
	decoder := NewDecoder(&dataErrorReader{data: []byte("1 "), err: sentinel})
	var value float64
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("first Decode = %v, want nil", err)
	}
	if value != 1 {
		t.Fatalf("decoded value = %v, want 1", value)
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := decoder.Decode(&value); err != sentinel {
			t.Fatalf("Decode attempt %d error = %v, want sentinel", attempt+2, err)
		}
	}
}

func TestDecoderBufferedNormalizedView(t *testing.T) {
	t.Parallel()

	t.Run("resolved JSONC", func(t *testing.T) {
		input := []byte(`{"a":1,}/* hidden */ 2`)
		normalized, err := Sanitize(input)
		if err != nil {
			t.Fatal(err)
		}
		decoder := NewDecoder(bytes.NewReader(input))
		var first map[string]int
		if err := decoder.Decode(&first); err != nil {
			t.Fatal(err)
		}
		offset := decoder.InputOffset()
		buffered, err := io.ReadAll(decoder.Buffered())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(buffered, normalized[offset:]) {
			t.Fatalf("Buffered = %q, want %q", buffered, normalized[offset:])
		}
		if decoder.InputOffset() != offset {
			t.Fatal("reading Buffered advanced InputOffset")
		}
	})

	t.Run("unresolved comma is not committed", func(t *testing.T) {
		source := &chunkSequenceReader{chunks: [][]byte{[]byte(`[1,`), []byte(`]`)}}
		decoder := NewDecoder(source)
		token, err := decoder.Token()
		if err != nil || token != Delim('[') {
			t.Fatalf("first Token = %v, %v", token, err)
		}
		buffered, err := io.ReadAll(decoder.Buffered())
		if err != nil {
			t.Fatal(err)
		}
		if string(buffered) != "1" {
			t.Fatalf("Buffered = %q, want committed prefix %q", buffered, "1")
		}
		token, err = decoder.Token()
		if err != nil || token != float64(1) {
			t.Fatalf("number Token = %T(%v), %v", token, token, err)
		}
		if decoder.More() {
			t.Fatal("trailing comma created another array value")
		}
		token, err = decoder.Token()
		if err != nil || token != Delim(']') {
			t.Fatalf("closing Token = %v, %v", token, err)
		}
	})
}

func TestDecoderCallbacksReceiveNormalizedJSON(t *testing.T) {
	t.Parallel()

	input := []byte(`{"a":/* comment */1,}`)
	want, err := Sanitize(input)
	if err != nil {
		t.Fatal(err)
	}

	var captured decoderCapture
	if err := NewDecoder(iotest.OneByteReader(bytes.NewReader(input))).Decode(&captured); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(captured, want) {
		t.Fatalf("UnmarshalJSON received %q, want %q", captured, want)
	}

	var raw RawMessage
	if err := NewDecoder(bytes.NewReader(input)).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, want) {
		t.Fatalf("RawMessage received %q, want %q", raw, want)
	}
}

func TestDecoderSyntaxOffsetUsesOriginalInput(t *testing.T) {
	t.Parallel()

	input := []byte(`[1,/* comment */2x]`)
	normalized, err := Sanitize(input)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	gotErr := NewDecoder(iotest.OneByteReader(bytes.NewReader(input))).Decode(&gotValue)
	wantErr := stdjson.NewDecoder(iotest.OneByteReader(bytes.NewReader(normalized))).Decode(&wantValue)
	if !sameDecoderError(gotErr, wantErr) {
		t.Fatalf("syntax errors differ: got %v, want %v", gotErr, wantErr)
	}
	var gotSyntax, wantSyntax *SyntaxError
	if !errors.As(gotErr, &gotSyntax) || !errors.As(wantErr, &wantSyntax) {
		t.Fatalf("errors are not SyntaxError: got %T, want %T", gotErr, wantErr)
	}
	if gotSyntax.Offset != wantSyntax.Offset {
		t.Fatalf("syntax offset = %d, want original offset %d", gotSyntax.Offset, wantSyntax.Offset)
	}
}

func TestDecoderCompletesTrailingCommaWithoutReaderEOF(t *testing.T) {
	t.Parallel()

	reader, writer := net.Pipe()
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()

	writeDone := make(chan error, 1)
	go func() {
		_, err := writer.Write([]byte(`[1,/* tail */]`))
		writeDone <- err
	}()

	decodeDone := make(chan error, 1)
	go func() {
		var value []int
		decodeDone <- NewDecoder(reader).Decode(&value)
	}()

	select {
	case err := <-decodeDone:
		if err != nil {
			t.Fatalf("Decode = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Decode waited for EOF after a complete JSONC value")
	}

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("Write = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not complete")
	}
}

type decoderCapture []byte

func (c *decoderCapture) UnmarshalJSON(data []byte) error {
	*c = append((*c)[:0], data...)
	return nil
}

type dataErrorReader struct {
	data []byte
	err  error
	done bool
}

func (r *dataErrorReader) Read(dst []byte) (int, error) {
	if r.done {
		return 0, r.err
	}
	r.done = true
	return copy(dst, r.data), r.err
}

type chunkSequenceReader struct {
	chunks [][]byte
}

func (r *chunkSequenceReader) Read(dst []byte) (int, error) {
	if len(r.chunks) == 0 {
		return 0, io.EOF
	}
	chunk := r.chunks[0]
	count := copy(dst, chunk)
	if count == len(chunk) {
		r.chunks = r.chunks[1:]
	} else {
		r.chunks[0] = chunk[count:]
	}
	return count, nil
}

func sameDecoderError(got, want error) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return reflect.TypeOf(got) == reflect.TypeOf(want) && got.Error() == want.Error()
}
