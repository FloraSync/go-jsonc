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
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	maxSanitizeFuzzInput = 64 << 10
	maxFacadeFuzzInput   = 32 << 10
	maxDecoderFuzzData   = 16 << 10
	maxChunkPlanBytes    = 64
	maxActionPlanBytes   = 64
	maxDecoderOperations = 256
)

const (
	flagDataErr         uint8 = 1 << 0
	flagSourceErr       uint8 = 1 << 1
	flagUseNumber       uint8 = 1 << 2
	flagDisallowUnknown uint8 = 1 << 3
)

var errFuzzSource = errors.New("fuzz reader source error")

type fuzzSeed struct {
	name string
	data []byte
}

// phase4FuzzSeeds is source-visible seed corpus shared by all three targets.
// Keep large depth and allocation boundaries in deterministic tests so every
// fuzz iteration stays cheap.
var phase4FuzzSeeds = []fuzzSeed{
	{name: "empty", data: []byte{}},
	{name: "strict null", data: []byte(`null`)},
	{name: "strict object", data: []byte(`{"url":"https://example.test/a/*literal*/","values":[1,true,null]}`)},
	{name: "line LF", data: []byte("// comment\n1")},
	{name: "line CR", data: []byte("// comment\r1")},
	{name: "line CRLF", data: []byte("// comment\r\n1")},
	{name: "line EOF", data: []byte("1// comment")},
	{name: "unicode separators", data: []byte("// \u2028 \u2029\n1")},
	{name: "empty block", data: []byte(`/**/1`)},
	{name: "repeated stars", data: []byte(`/****/1`)},
	{name: "first block terminator", data: []byte(`/* outer /* inner */ 1`)},
	{name: "unterminated block", data: []byte(`/* unterminated`)},
	{name: "lone slash", data: []byte(`/`)},
	{name: "stray block closer", data: []byte(`*/1`)},
	{name: "string markers", data: []byte(`{"line":"// text","block":"/* text */"}`)},
	{name: "escape parity", data: []byte(`{"s":"\\\"// still string"}`)},
	{name: "unicode comment", data: []byte("[1,/* π🌿 */]")},
	{name: "invalid line UTF-8", data: []byte{'/', '/', 0xff, '\n', '1'}},
	{name: "invalid block UTF-8", data: []byte{'/', '*', 0xff, '*', '/', '1'}},
	{name: "truncated comment UTF-8", data: []byte{'/', '*', 0xe2}},
	{name: "binary outside comment", data: []byte{0xff, '[', '1', ',', ']'}},
	{name: "token split literal", data: []byte(`tr/*x*/ue`)},
	{name: "token split number", data: []byte(`1/*x*/.5`)},
	{name: "token split line", data: []byte("nu//x\nll")},
	{name: "array trailing comma", data: []byte(`[1,]`)},
	{name: "object trailing comma", data: []byte(`{"known":1,}`)},
	{name: "commented trailing comma", data: []byte(`[1, /* tail */]`)},
	{name: "nested trailing commas", data: []byte(`[{"known":[true,],},]`)},
	{name: "empty comma", data: []byte(`[,]`)},
	{name: "repeated comma", data: []byte(`[1,,]`)},
	{name: "wrong closer", data: []byte(`[1,}`)},
	{name: "malformed numbers", data: []byte(`[01,+1,1.,1e,NaN]`)},
	{name: "malformed literals", data: []byte(`tru fals nullx`)},
	{name: "concatenated values", data: []byte(`1 /* between */ {"known":2,} [3,]`)},
	{name: "trailing lexical error", data: []byte(`{} /* open`)},
	{name: "NUL comment", data: []byte{'/', '*', 0, '*', '/', '1'}},
}

func FuzzSanitize(f *testing.F) {
	for _, seed := range phase4FuzzSeeds {
		f.Add(bytes.Clone(seed.data))
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > maxSanitizeFuzzInput {
			t.Skip()
		}

		before := bytes.Clone(input)
		_ = HasCommentRunes(input)
		normalized, err := Sanitize(input)
		if !bytes.Equal(input, before) {
			t.Fatalf("Sanitize mutated input\n got: %q\nwant: %q", input, before)
		}

		repeated, repeatedErr := Sanitize(input)
		requireEquivalentFuzzError(t, repeatedErr, err)
		if !bytes.Equal(repeated, normalized) {
			t.Fatalf("Sanitize is nondeterministic\nfirst:  %q\nsecond: %q", normalized, repeated)
		}

		if err != nil {
			if normalized != nil {
				t.Fatalf("Sanitize output on error = %q, want nil", normalized)
			}
			requireJSONCLexicalFuzzError(t, err, len(input))
			if Valid(input) {
				t.Fatal("Valid returned true after Sanitize failed")
			}
			return
		}

		if len(normalized) != len(input) {
			t.Fatalf("Sanitize length = %d, want %d", len(normalized), len(input))
		}
		for i := range input {
			if normalized[i] != input[i] && normalized[i] != ' ' {
				t.Fatalf("Sanitize changed byte %d from %#x to %#x, want ASCII space", i, input[i], normalized[i])
			}
		}

		idempotent, idempotentErr := Sanitize(normalized)
		if idempotentErr != nil {
			t.Fatalf("Sanitize(normalized) error = %v", idempotentErr)
		}
		if !bytes.Equal(idempotent, normalized) {
			t.Fatalf("Sanitize is not idempotent\n once: %q\ntwice: %q", normalized, idempotent)
		}
		if got, want := Valid(input), stdjson.Valid(normalized); got != want {
			t.Fatalf("Valid = %v, want %v for normalized %q", got, want, normalized)
		}
	})
}

type fuzzFixedStruct struct {
	A int            `json:"a"`
	B string         `json:"b"`
	C bool           `json:"c"`
	D []int          `json:"d"`
	E map[string]int `json:"e"`
}

type fuzzRecordingUnmarshaler struct {
	data []byte
}

func (u *fuzzRecordingUnmarshaler) UnmarshalJSON(b []byte) error {
	u.data = append([]byte(nil), b...)
	return nil
}

func FuzzFacadeDifferential(f *testing.F) {
	for _, seed := range phase4FuzzSeeds {
		f.Add(bytes.Clone(seed.data))
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > maxFacadeFuzzInput {
			t.Skip()
		}

		normalized, sanitizeErr := Sanitize(input)
		if got, want := Valid(input), sanitizeErr == nil && stdjson.Valid(normalized); got != want {
			t.Fatalf("Valid(%q) = %v, want %v (sanitizeErr = %v)", input, got, want, sanitizeErr)
		}

		if sanitizeErr != nil {
			var gotAny any
			gotErr := Unmarshal(input, &gotAny)
			if gotErr == nil {
				t.Fatalf("Unmarshal succeeded on invalid JSONC input: %q", input)
			}
			requireEquivalentFuzzError(t, gotErr, sanitizeErr)

			gotCompact := bytes.NewBufferString("prefix:")
			compactErr := Compact(gotCompact, input)
			if compactErr == nil {
				t.Fatalf("Compact succeeded on invalid JSONC input: %q", input)
			}
			requireEquivalentFuzzError(t, compactErr, sanitizeErr)
			if gotCompact.String() != "prefix:" {
				t.Fatalf("Compact mutated destination on error: got %q, want %q", gotCompact.String(), "prefix:")
			}

			gotIndent := bytes.NewBufferString("prefix:")
			indentErr := Indent(gotIndent, input, "  ", "\t")
			if indentErr == nil {
				t.Fatalf("Indent succeeded on invalid JSONC input: %q", input)
			}
			requireEquivalentFuzzError(t, indentErr, sanitizeErr)
			if gotIndent.String() != "prefix:" {
				t.Fatalf("Indent mutated destination on error: got %q, want %q", gotIndent.String(), "prefix:")
			}
			return
		}

		// Unmarshal into any
		var gotAny, wantAny any
		gotErr := Unmarshal(input, &gotAny)
		wantErr := stdjson.Unmarshal(normalized, &wantAny)
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !reflect.DeepEqual(gotAny, wantAny) {
			t.Fatalf("Unmarshal(any) value mismatch:\n got: %#v\nwant: %#v", gotAny, wantAny)
		}

		// Unmarshal into fixed struct
		var gotStruct, wantStruct fuzzFixedStruct
		gotErr = Unmarshal(input, &gotStruct)
		wantErr = stdjson.Unmarshal(normalized, &wantStruct)
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !reflect.DeepEqual(gotStruct, wantStruct) {
			t.Fatalf("Unmarshal(struct) value mismatch:\n got: %#v\nwant: %#v", gotStruct, wantStruct)
		}

		// Unmarshal into RawMessage
		var gotRaw, wantRaw RawMessage
		gotErr = Unmarshal(input, &gotRaw)
		wantErr = stdjson.Unmarshal(normalized, &wantRaw)
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !bytes.Equal(gotRaw, wantRaw) {
			t.Fatalf("Unmarshal(RawMessage) mismatch:\n got: %q\nwant: %q", gotRaw, wantRaw)
		}

		// Unmarshal into recording Unmarshaler
		var gotRec, wantRec fuzzRecordingUnmarshaler
		gotErr = Unmarshal(input, &gotRec)
		wantErr = stdjson.Unmarshal(normalized, &wantRec)
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !bytes.Equal(gotRec.data, wantRec.data) {
			t.Fatalf("Unmarshal(recording) mismatch:\n got: %q\nwant: %q", gotRec.data, wantRec.data)
		}

		// Compact with prefix
		gotCompact := bytes.NewBufferString("prefix:")
		gotErr = Compact(gotCompact, input)
		wantCompact := bytes.NewBufferString("prefix:")
		wantErr = stdjson.Compact(wantCompact, normalized)
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !bytes.Equal(gotCompact.Bytes(), wantCompact.Bytes()) {
			t.Fatalf("Compact mismatch:\n got: %q\nwant: %q", gotCompact.Bytes(), wantCompact.Bytes())
		}

		// Indent with prefix
		gotIndent := bytes.NewBufferString("prefix:")
		gotErr = Indent(gotIndent, input, "  ", "\t")
		wantIndent := bytes.NewBufferString("prefix:")
		wantErr = stdjson.Indent(wantIndent, normalized, "  ", "\t")
		requireEquivalentFuzzError(t, gotErr, wantErr)
		if !bytes.Equal(gotIndent.Bytes(), wantIndent.Bytes()) {
			t.Fatalf("Indent mismatch:\n got: %q\nwant: %q", gotIndent.Bytes(), wantIndent.Bytes())
		}
	})
}

type fuzzReader struct {
	data       []byte
	pos        int
	chunkPlan  []byte
	planPos    int
	pausesLeft int
	dataAndEOF bool
	sourceErr  error
}

func newFuzzReader(data, chunkPlan []byte, flags uint8) *fuzzReader {
	var srcErr error
	if flags&flagSourceErr != 0 {
		srcErr = errFuzzSource
	}
	return &fuzzReader{
		data:       data,
		chunkPlan:  chunkPlan,
		pausesLeft: 8,
		dataAndEOF: flags&flagDataErr != 0,
		sourceErr:  srcErr,
	}
}

func (r *fuzzReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.pos >= len(r.data) {
		if r.sourceErr != nil {
			return 0, r.sourceErr
		}
		return 0, io.EOF
	}

	if r.planPos < len(r.chunkPlan) {
		planByte := r.chunkPlan[r.planPos]
		r.planPos++
		if planByte == 0 && r.pausesLeft > 0 {
			r.pausesLeft--
			return 0, nil
		}
		chunkLimit := int(planByte%64) + 1
		if len(p) > chunkLimit {
			p = p[:chunkLimit]
		}
	}

	available := len(r.data) - r.pos
	toCopy := len(p)
	if toCopy > available {
		toCopy = available
	}
	n := copy(p, r.data[r.pos:r.pos+toCopy])
	r.pos += n

	if r.pos >= len(r.data) {
		if r.dataAndEOF {
			if r.sourceErr != nil {
				return n, r.sourceErr
			}
			return n, io.EOF
		}
	}
	return n, nil
}

func FuzzDecoder(f *testing.F) {
	for _, seed := range phase4FuzzSeeds {
		f.Add(bytes.Clone(seed.data), []byte{1}, []byte{0}, uint8(0))
		f.Add(bytes.Clone(seed.data), []byte{4, 8, 16}, []byte{0, 1, 2, 3}, uint8(0))
		f.Add(bytes.Clone(seed.data), []byte{0, 1, 2}, []byte{3, 4, 5}, flagDataErr)
		f.Add(bytes.Clone(seed.data), []byte{64}, []byte{0}, flagUseNumber|flagDisallowUnknown)
		f.Add(bytes.Clone(seed.data), []byte{1, 0, 1}, []byte{0}, flagSourceErr)
	}

	f.Fuzz(func(t *testing.T, data, chunkPlan, actions []byte, flags uint8) {
		if len(data) > maxDecoderFuzzData || len(chunkPlan) > maxChunkPlanBytes || len(actions) > maxActionPlanBytes {
			t.Skip()
		}

		// First compare streamNormalizer ReadAll with Sanitize under EOF
		cleanReader := newFuzzReader(data, chunkPlan, flags&^flagSourceErr)
		streamed, streamErr := io.ReadAll(newStreamNormalizer(cleanReader))
		sanitized, sanitizeErr := Sanitize(data)

		if sanitizeErr != nil {
			if streamErr == nil {
				t.Fatalf("stream normalization succeeded on invalid input %q", data)
			}
			requireEquivalentFuzzError(t, streamErr, sanitizeErr)
		} else {
			if streamErr != nil {
				t.Fatalf("stream normalization error = %v for valid input", streamErr)
			}
			if !bytes.Equal(streamed, sanitized) {
				t.Fatalf("streamed bytes differ from Sanitize\n got: %q\nwant: %q", streamed, sanitized)
			}
		}

		// If whole-input sanitization fails, drive public decoder only far enough to detect errors
		if sanitizeErr != nil {
			dec := NewDecoder(newFuzzReader(data, chunkPlan, flags))
			if flags&flagUseNumber != 0 {
				dec.UseNumber()
			}
			if flags&flagDisallowUnknown != 0 {
				dec.DisallowUnknownFields()
			}

			ops := len(actions)
			if ops == 0 {
				ops = 1
			}
			if ops > maxDecoderOperations {
				ops = maxDecoderOperations
			}

			for i := 0; i < ops; i++ {
				var act byte
				if i < len(actions) {
					act = actions[i]
				}
				var err error
				switch act % 4 {
				case 0:
					var v any
					err = dec.Decode(&v)
				case 1:
					var s fuzzFixedStruct
					err = dec.Decode(&s)
				case 2:
					var raw RawMessage
					err = dec.Decode(&raw)
				case 3:
					_, err = dec.Token()
				}
				if err != nil {
					var syntaxErr *JSONCSyntaxError
					if errors.As(err, &syntaxErr) || errors.Is(err, ErrInvalidUTF8) || errors.Is(err, ErrUnterminatedBlockComment) {
						var repeatedVal any
						repeatedErr := dec.Decode(&repeatedVal)
						requireEquivalentFuzzError(t, repeatedErr, err)
						if _, err := io.ReadAll(dec.Buffered()); err != nil {
							t.Fatalf("Buffered() error after lexical error: %v", err)
						}
					}
					_ = dec.InputOffset()
					break
				}
			}
			return
		}

		// When sanitization succeeds, compare NewDecoder with stdjson.NewDecoder over normalized bytes
		jsoncReader := newFuzzReader(data, chunkPlan, flags)
		stdReader := newFuzzReader(sanitized, chunkPlan, flags)

		gotDec := NewDecoder(jsoncReader)
		wantDec := stdjson.NewDecoder(stdReader)

		if flags&flagUseNumber != 0 {
			gotDec.UseNumber()
			wantDec.UseNumber()
		}
		if flags&flagDisallowUnknown != 0 {
			gotDec.DisallowUnknownFields()
			wantDec.DisallowUnknownFields()
		}

		depth := 0
		ops := len(actions)
		if ops == 0 {
			ops = 1
		}
		if ops > maxDecoderOperations {
			ops = maxDecoderOperations
		}

		for i := 0; i < ops; i++ {
			var act byte
			if i < len(actions) {
				act = actions[i]
			}
			op := act % 7

			switch op {
			case 0:
				var gotVal, wantVal any
				gotErr := gotDec.Decode(&gotVal)
				wantErr := wantDec.Decode(&wantVal)
				requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
				if !reflect.DeepEqual(gotVal, wantVal) {
					t.Fatalf("Decode(any) mismatch:\n got: %#v\nwant: %#v", gotVal, wantVal)
				}
				if gotDec.InputOffset() != wantDec.InputOffset() {
					t.Fatalf("InputOffset after Decode(any) = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
				}
				if gotErr != nil {
					return
				}

			case 1:
				var gotStruct, wantStruct fuzzFixedStruct
				gotErr := gotDec.Decode(&gotStruct)
				wantErr := wantDec.Decode(&wantStruct)
				requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
				if !reflect.DeepEqual(gotStruct, wantStruct) {
					t.Fatalf("Decode(struct) mismatch:\n got: %#v\nwant: %#v", gotStruct, wantStruct)
				}
				if gotDec.InputOffset() != wantDec.InputOffset() {
					t.Fatalf("InputOffset after Decode(struct) = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
				}
				if gotErr != nil {
					return
				}

			case 2:
				var gotRaw, wantRaw RawMessage
				gotErr := gotDec.Decode(&gotRaw)
				wantErr := wantDec.Decode(&wantRaw)
				requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
				if !bytes.Equal(gotRaw, wantRaw) {
					t.Fatalf("Decode(RawMessage) mismatch:\n got: %q\nwant: %q", gotRaw, wantRaw)
				}
				if gotDec.InputOffset() != wantDec.InputOffset() {
					t.Fatalf("InputOffset after Decode(RawMessage) = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
				}
				if gotErr != nil {
					return
				}

			case 3:
				gotToken, gotErr := gotDec.Token()
				wantToken, wantErr := wantDec.Token()
				requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
				if !reflect.DeepEqual(gotToken, wantToken) {
					t.Fatalf("Token mismatch:\n got: %T(%v)\nwant: %T(%v)", gotToken, gotToken, wantToken, wantToken)
				}
				if gotDec.InputOffset() != wantDec.InputOffset() {
					t.Fatalf("InputOffset after Token = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
				}
				if d, ok := gotToken.(Delim); ok {
					if d == '[' || d == '{' {
						depth++
					} else if d == ']' || d == '}' {
						if depth > 0 {
							depth--
						}
					}
				}
				if gotErr != nil {
					return
				}

			case 4:
				if depth > 0 {
					gotMore := gotDec.More()
					wantMore := wantDec.More()
					if gotMore != wantMore {
						t.Fatalf("More() = %v, want %v", gotMore, wantMore)
					}
					if gotDec.InputOffset() != wantDec.InputOffset() {
						t.Fatalf("InputOffset after More() = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
					}
				} else {
					var gotVal, wantVal any
					gotErr := gotDec.Decode(&gotVal)
					wantErr := wantDec.Decode(&wantVal)
					requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
					if !reflect.DeepEqual(gotVal, wantVal) {
						t.Fatalf("Decode(any) mismatch:\n got: %#v\nwant: %#v", gotVal, wantVal)
					}
					if gotDec.InputOffset() != wantDec.InputOffset() {
						t.Fatalf("InputOffset after Decode(any) = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
					}
					if gotErr != nil {
						return
					}
				}

			case 5:
				offsetBefore := gotDec.InputOffset()
				bufReader := gotDec.Buffered()
				limitReader := io.LimitReader(bufReader, 512)
				bufBytes, bufErr := io.ReadAll(limitReader)
				if bufErr != nil {
					t.Fatalf("Buffered() ReadAll error = %v", bufErr)
				}
				offsetAfter := gotDec.InputOffset()
				if offsetBefore != offsetAfter {
					t.Fatalf("Buffered() advanced InputOffset from %d to %d", offsetBefore, offsetAfter)
				}
				if int(offsetBefore) <= len(sanitized) {
					remaining := sanitized[offsetBefore:]
					if len(bufBytes) > len(remaining) || !bytes.Equal(bufBytes, remaining[:len(bufBytes)]) {
						t.Fatalf("Buffered bytes %q are not prefix of remaining %q", bufBytes, remaining)
					}
				}

			case 6:
				var gotRec, wantRec fuzzRecordingUnmarshaler
				gotErr := gotDec.Decode(&gotRec)
				wantErr := wantDec.Decode(&wantRec)
				requireEquivalentDecoderFuzzError(t, gotErr, wantErr)
				if !bytes.Equal(gotRec.data, wantRec.data) {
					t.Fatalf("Decode(recording) mismatch:\n got: %q\nwant: %q", gotRec.data, wantRec.data)
				}
				if gotDec.InputOffset() != wantDec.InputOffset() {
					t.Fatalf("InputOffset after Decode(recording) = %d, want %d", gotDec.InputOffset(), wantDec.InputOffset())
				}
				if gotErr != nil {
					return
				}
			}
		}
	})
}

func TestFuzzTargetInventory(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(filepath.Join("testdata", "fuzz-targets.txt"))
	if err != nil {
		t.Fatalf("ReadFile(testdata/fuzz-targets.txt) error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		t.Fatal("fuzz target inventory is empty")
	}

	seen := make(map[string]bool)
	var targets []string
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if seen[name] {
			t.Fatalf("duplicate fuzz target %q in inventory", name)
		}
		seen[name] = true
		targets = append(targets, name)
	}

	wantTargets := []string{
		"FuzzSanitize",
		"FuzzFacadeDifferential",
		"FuzzDecoder",
	}

	if len(targets) != len(wantTargets) {
		t.Fatalf("fuzz target count = %d, want %d", len(targets), len(wantTargets))
	}
	for i, want := range wantTargets {
		if targets[i] != want {
			t.Fatalf("fuzz target at index %d = %q, want %q", i, targets[i], want)
		}
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool {
		return strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	var foundFuzz []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					if strings.HasPrefix(fn.Name.Name, "Fuzz") {
						foundFuzz = append(foundFuzz, fn.Name.Name)
					}
				}
			}
		}
	}

	sort.Strings(foundFuzz)
	expectedSorted := append([]string(nil), wantTargets...)
	sort.Strings(expectedSorted)

	if !reflect.DeepEqual(foundFuzz, expectedSorted) {
		t.Fatalf("found fuzz functions %v, want %v", foundFuzz, expectedSorted)
	}
}

func requireJSONCLexicalFuzzError(t *testing.T, err error, inputLength int) {
	t.Helper()

	if !errors.Is(err, ErrInvalidUTF8) && !errors.Is(err, ErrUnterminatedBlockComment) {
		t.Fatalf("Sanitize error = %v, want a JSONC lexical category", err)
	}
	var syntaxErr *JSONCSyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("Sanitize error type = %T, want *JSONCSyntaxError", err)
	}
	if syntaxErr.Offset < 1 || syntaxErr.Offset > int64(inputLength) {
		t.Fatalf("JSONCSyntaxError.Offset = %d, input length = %d", syntaxErr.Offset, inputLength)
	}
}

func requireEquivalentFuzzError(t *testing.T, got, want error) {
	t.Helper()

	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Fatalf("errors differ: got %T(%v), want %T(%v)", got, got, want, want)
		}
		return
	}
	if reflect.TypeOf(got) != reflect.TypeOf(want) || got.Error() != want.Error() {
		t.Fatalf("errors differ: got %T(%v), want %T(%v)", got, got, want, want)
	}

	var gotJSONC, wantJSONC *JSONCSyntaxError
	gotIsJSONC := errors.As(got, &gotJSONC)
	wantIsJSONC := errors.As(want, &wantJSONC)
	if gotIsJSONC || wantIsJSONC {
		if gotJSONC == nil || wantJSONC == nil || gotJSONC.Offset != wantJSONC.Offset ||
			errors.Is(gotJSONC, ErrInvalidUTF8) != errors.Is(wantJSONC, ErrInvalidUTF8) ||
			errors.Is(gotJSONC, ErrUnterminatedBlockComment) != errors.Is(wantJSONC, ErrUnterminatedBlockComment) {
			t.Fatalf("JSONC errors differ: got %#v, want %#v", gotJSONC, wantJSONC)
		}
	}

	var gotSyntax, wantSyntax *SyntaxError
	gotIsSyntax := errors.As(got, &gotSyntax)
	wantIsSyntax := errors.As(want, &wantSyntax)
	if gotIsSyntax || wantIsSyntax {
		if gotSyntax == nil || wantSyntax == nil || gotSyntax.Offset != wantSyntax.Offset {
			t.Fatalf("syntax errors differ: got %#v, want %#v", gotSyntax, wantSyntax)
		}
	}

	var gotType, wantType *UnmarshalTypeError
	gotIsType := errors.As(got, &gotType)
	wantIsType := errors.As(want, &wantType)
	if gotIsType || wantIsType {
		if gotType == nil || wantType == nil ||
			gotType.Value != wantType.Value ||
			gotType.Type != wantType.Type ||
			gotType.Offset != wantType.Offset ||
			gotType.Struct != wantType.Struct ||
			gotType.Field != wantType.Field {
			t.Fatalf("unmarshal type errors differ: got %#v, want %#v", gotType, wantType)
		}
	}
}

func requireEquivalentDecoderFuzzError(t *testing.T, got, want error) {
	t.Helper()

	if got == nil || want == nil {
		if got != nil || want != nil {
			t.Fatalf("decoder errors differ: got %T(%v), want %T(%v)", got, got, want, want)
		}
		return
	}
	if errors.Is(got, errFuzzSource) && errors.Is(want, errFuzzSource) {
		return
	}
	if reflect.TypeOf(got) != reflect.TypeOf(want) || got.Error() != want.Error() {
		t.Fatalf("decoder errors differ: got %T(%v), want %T(%v)", got, got, want, want)
	}

	var gotSyntax, wantSyntax *SyntaxError
	if errors.As(got, &gotSyntax) && errors.As(want, &wantSyntax) {
		if gotSyntax.Offset != wantSyntax.Offset {
			t.Fatalf("decoder syntax offset = %d, want %d", gotSyntax.Offset, wantSyntax.Offset)
		}
	}

	var gotType, wantType *UnmarshalTypeError
	if errors.As(got, &gotType) && errors.As(want, &wantType) {
		if gotType.Value != wantType.Value ||
			gotType.Type != wantType.Type ||
			gotType.Offset != wantType.Offset ||
			gotType.Struct != wantType.Struct ||
			gotType.Field != wantType.Field {
			t.Fatalf("decoder unmarshal type errors differ: got %#v, want %#v", gotType, wantType)
		}
	}
}
