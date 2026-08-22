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

package json

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

var (
	// ErrInvalidUTF8 reports malformed UTF-8 in a JSONC comment body.
	// Malformed UTF-8 outside comments retains encoding/json behavior.
	ErrInvalidUTF8 = errors.New("jsonc: invalid UTF-8 in comment")

	// ErrUnterminatedBlockComment reports a block comment without a closing */.
	ErrUnterminatedBlockComment = errors.New("jsonc: unterminated block comment")
)

// JSONCSyntaxError describes syntax that belongs to the JSONC extension rather
// than ordinary JSON. Offset is a one-based byte offset into the original
// input. Err can be inspected with [errors.Is].
type JSONCSyntaxError struct {
	Offset int64
	Err    error
}

func (e *JSONCSyntaxError) Error() string {
	if e == nil {
		return "jsonc: <nil>"
	}
	if e.Err == nil {
		return fmt.Sprintf("jsonc: syntax error at byte %d", e.Offset)
	}
	return fmt.Sprintf("%v at byte %d", e.Err, e.Offset)
}

// Unwrap returns the JSONC syntax category.
func (e *JSONCSyntaxError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// Sanitize replaces JSONC comments and accepted trailing commas with
// byte-length-preserving JSON whitespace. It never mutates data.
//
// Sanitize validates JSONC lexical extensions but does not otherwise validate
// JSON. Call [Valid] or [Unmarshal] when full validation is required.
func Sanitize(data []byte) ([]byte, error) {
	data, _, err := normalizeBytes(data)
	return data, err
}

// HasCommentRunes reports whether data contains a // or /* opener outside a
// JSON string. It does not validate the surrounding document or detect
// trailing commas.
func HasCommentRunes(data []byte) bool {
	return hasCommentOpener(data)
}

func normalizeBytes(data []byte) ([]byte, bool, error) {
	if !mayHaveExtensions(data) {
		return data, false, nil
	}
	return normalizeExtensions(data)
}

// mayHaveExtensions is the allocation-free strict JSON fast path. It is
// deliberately conservative for malformed JSON: normalizeExtensions performs
// the structural decision before changing a possible trailing comma.
func mayHaveExtensions(data []byte) bool {
	var previous byte
	havePrevious := false
	pendingComma := false
	inString := false
	escaped := false

	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
				previous = c
				havePrevious = true
			}
			continue
		}

		if isJSONWhitespace(c) {
			continue
		}
		if c == '/' && i+1 < len(data) && (data[i+1] == '/' || data[i+1] == '*') {
			return true
		}
		if pendingComma {
			if c == ']' || c == '}' {
				return true
			}
			pendingComma = false
		}

		switch c {
		case '"':
			inString = true
		case ',':
			pendingComma = canPrecedeTrailingComma(previous, havePrevious)
		}
		previous = c
		havePrevious = true
	}

	return false
}

func hasCommentOpener(data []byte) bool {
	inString := false
	escaped := false
	for i := 0; i < len(data); i++ {
		c := data[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		if c == '"' {
			inString = true
			continue
		}
		if c == '/' && i+1 < len(data) && (data[i+1] == '/' || data[i+1] == '*') {
			return true
		}
	}
	return false
}

const maxTrackedNestingDepth = 10_000

type containerFrame struct {
	opener              byte
	keySeen             bool
	colonSeen           bool
	valueSeen           bool
	countsAsParentValue bool
}

type stringRole uint8

const (
	stringRoleNone stringRole = iota
	stringRoleObjectKey
	stringRoleValue
)

func normalizeExtensions(data []byte) ([]byte, bool, error) {
	var normalized []byte
	var frames []containerFrame
	pendingComma := -1
	pendingDepth := 0
	var pendingClose byte
	structureDisabled := false
	inString := false
	escaped := false
	role := stringRoleNone
	roleDepth := 0
	changed := false

	ensureCopy := func() {
		if normalized == nil {
			normalized = append([]byte(nil), data...)
		}
	}

	for i := 0; i < len(data); {
		c := data[i]

		if inString {
			if escaped {
				escaped = false
				i++
				continue
			}
			switch c {
			case '\\':
				escaped = true
			case '"':
				inString = false
				if !structureDisabled && len(frames) == roleDepth && roleDepth > 0 {
					frame := &frames[roleDepth-1]
					switch role {
					case stringRoleObjectKey:
						frame.keySeen = true
					case stringRoleValue:
						frame.valueSeen = true
					}
				}
				role = stringRoleNone
			}
			i++
			continue
		}

		if isJSONWhitespace(c) {
			i++
			continue
		}

		if c == '/' {
			end, isComment, err := scanComment(data, i)
			if err != nil {
				return nil, false, err
			}
			if isComment {
				ensureCopy()
				blankComment(normalized, i, end)
				changed = true
				i = end
				continue
			}
		}

		if pendingComma >= 0 {
			if !structureDisabled && len(frames) == pendingDepth && c == pendingClose {
				ensureCopy()
				normalized[pendingComma] = ' '
				changed = true
			}
			pendingComma = -1
		}

		switch c {
		case '"':
			inString = true
			role, roleDepth = nextStringRole(frames, structureDisabled)
		case '[', '{':
			if structureDisabled {
				break
			}
			if len(frames) >= maxTrackedNestingDepth {
				// encoding/json rejects documents beyond this depth. Stop
				// recognizing trailing commas rather than growing attacker-
				// controlled structural memory without bound.
				structureDisabled = true
				frames = nil
				break
			}
			frames = append(frames, containerFrame{
				opener:              c,
				countsAsParentValue: canStartValue(frames),
			})
		case ']', '}':
			if structureDisabled || len(frames) == 0 {
				break
			}
			last := len(frames) - 1
			if !matchingDelimiter(frames[last].opener, c) {
				break
			}
			countsAsValue := frames[last].countsAsParentValue
			frames = frames[:last]
			if countsAsValue {
				markValueSeen(frames)
			}
		case ':':
			if structureDisabled || len(frames) == 0 {
				break
			}
			frame := &frames[len(frames)-1]
			if frame.opener == '{' && frame.keySeen && !frame.colonSeen {
				frame.colonSeen = true
			}
		case ',':
			if structureDisabled || len(frames) == 0 {
				break
			}
			frame := &frames[len(frames)-1]
			if frameHasValue(*frame) {
				pendingComma = i
				pendingDepth = len(frames)
				pendingClose = matchingClose(frame.opener)
			}
			frame.keySeen = false
			frame.colonSeen = false
			frame.valueSeen = false
		default:
			if isScalarStart(c) && !structureDisabled {
				markValueSeen(frames)
			}
		}
		i++
	}

	if !changed {
		return data, false, nil
	}
	return normalized, true, nil
}

func scanComment(data []byte, start int) (end int, isComment bool, err error) {
	if start+1 >= len(data) || data[start] != '/' {
		return start + 1, false, nil
	}

	switch data[start+1] {
	case '/':
		for i := start + 2; i < len(data); {
			if data[i] == '\r' || data[i] == '\n' {
				return i, true, nil
			}
			size, err := commentRuneSize(data, i)
			if err != nil {
				return 0, true, err
			}
			i += size
		}
		return len(data), true, nil
	case '*':
		for i := start + 2; i < len(data); {
			if i+1 < len(data) && data[i] == '*' && data[i+1] == '/' {
				return i + 2, true, nil
			}
			size, err := commentRuneSize(data, i)
			if err != nil {
				return 0, true, err
			}
			i += size
		}
		return 0, true, &JSONCSyntaxError{
			Offset: int64(start + 1),
			Err:    ErrUnterminatedBlockComment,
		}
	default:
		return start + 1, false, nil
	}
}

func commentRuneSize(data []byte, offset int) (int, error) {
	r, size := utf8.DecodeRune(data[offset:])
	if r == utf8.RuneError && size == 1 {
		return 0, &JSONCSyntaxError{
			Offset: int64(offset + 1),
			Err:    ErrInvalidUTF8,
		}
	}
	return size, nil
}

func blankComment(data []byte, start, end int) {
	for i := start; i < end; i++ {
		if data[i] != '\r' && data[i] != '\n' {
			data[i] = ' '
		}
	}
}

func canPrecedeTrailingComma(previous byte, ok bool) bool {
	if !ok {
		return false
	}
	switch previous {
	case '[', '{', ',', ':':
		return false
	default:
		return true
	}
}

func nextStringRole(frames []containerFrame, disabled bool) (stringRole, int) {
	if disabled || len(frames) == 0 {
		return stringRoleNone, len(frames)
	}
	frame := frames[len(frames)-1]
	if frame.opener == '[' && !frame.valueSeen {
		return stringRoleValue, len(frames)
	}
	if frame.opener == '{' {
		if !frame.keySeen && !frame.colonSeen && !frame.valueSeen {
			return stringRoleObjectKey, len(frames)
		}
		if frame.keySeen && frame.colonSeen && !frame.valueSeen {
			return stringRoleValue, len(frames)
		}
	}
	return stringRoleNone, len(frames)
}

func canStartValue(frames []containerFrame) bool {
	if len(frames) == 0 {
		return false
	}
	frame := frames[len(frames)-1]
	if frame.opener == '[' {
		return !frame.valueSeen
	}
	return frame.keySeen && frame.colonSeen && !frame.valueSeen
}

func markValueSeen(frames []containerFrame) {
	if len(frames) == 0 {
		return
	}
	frame := &frames[len(frames)-1]
	if frame.opener == '[' || (frame.keySeen && frame.colonSeen) {
		frame.valueSeen = true
	}
}

func frameHasValue(frame containerFrame) bool {
	if frame.opener == '[' {
		return frame.valueSeen
	}
	return frame.keySeen && frame.colonSeen && frame.valueSeen
}

func matchingDelimiter(open, close byte) bool {
	return open == '[' && close == ']' || open == '{' && close == '}'
}

func matchingClose(open byte) byte {
	if open == '[' {
		return ']'
	}
	return '}'
}

func isScalarStart(c byte) bool {
	return c == '-' || c >= '0' && c <= '9' || c == 't' || c == 'f' || c == 'n'
}

func isJSONWhitespace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}
