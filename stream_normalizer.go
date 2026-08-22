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
	"fmt"
	"io"
	"unicode/utf8"
)

const maxStreamRead = 32 << 10

type streamMode uint8

const (
	streamNormal streamMode = iota
	streamString
	streamLineComment
	streamBlockComment
)

// streamNormalizer incrementally replaces JSONC extensions with whitespace.
// Every successful output byte corresponds to exactly one input byte.
type streamNormalizer struct {
	source io.Reader

	scratch []byte
	ready   []byte
	readyAt int

	terminalErr  error
	sourceOffset int64

	mode          streamMode
	stringEscaped bool
	pendingSlash  bool
	blockStar     bool
	blockStart    int64

	utf8Buf    [utf8.UTFMax]byte
	utf8Len    int
	utf8Offset int64

	structureDisabled bool
	frames            []containerFrame
	stringRole        stringRole
	stringRoleDepth   int

	commaPending bool
	pendingDepth int
	pendingClose byte
	pending      []byte
}

func newStreamNormalizer(source io.Reader) *streamNormalizer {
	return &streamNormalizer{source: source}
}

func (n *streamNormalizer) Read(dst []byte) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	if count := n.copyReady(dst); count != 0 {
		return count, nil
	}
	if n.terminalErr != nil {
		return 0, n.terminalErr
	}

	for {
		readSize := len(dst)
		if readSize > maxStreamRead {
			readSize = maxStreamRead
		}
		if cap(n.scratch) < readSize {
			n.scratch = make([]byte, readSize)
		}
		n.scratch = n.scratch[:readSize]

		count, readErr := n.source.Read(n.scratch)
		if count < 0 || count > len(n.scratch) {
			n.fail(fmt.Errorf("jsonc: reader returned invalid count %d", count))
			return 0, n.terminalErr
		}
		if count > 0 {
			n.process(n.scratch[:count])
		}

		if n.terminalErr == nil && readErr != nil {
			n.finish(readErr)
		}

		if count := n.copyReady(dst); count != 0 {
			return count, nil
		}
		if n.terminalErr != nil {
			return 0, n.terminalErr
		}

		// A source Reader returning no bytes and no error has made no
		// progress. Preserve that result rather than spinning internally.
		if count == 0 {
			return 0, nil
		}
		// All produced bytes are waiting on a slash or comma lookahead.
		// Read another chunk to resolve it.
	}
}

func (n *streamNormalizer) process(data []byte) {
	for _, current := range data {
		if n.terminalErr != nil {
			return
		}

		n.sourceOffset++
		n.processByte(current, n.sourceOffset)
	}
}

func (n *streamNormalizer) processByte(current byte, offset int64) {
	switch n.mode {
	case streamString:
		n.appendOutput(current)
		if n.stringEscaped {
			n.stringEscaped = false
			return
		}
		switch current {
		case '\\':
			n.stringEscaped = true
		case '"':
			n.mode = streamNormal
			n.finishStringValue()
		}
		return

	case streamLineComment:
		if current == '\r' || current == '\n' {
			if err := n.finishCommentUTF8(); err != nil {
				n.fail(err)
				return
			}
			n.mode = streamNormal
			n.appendOutput(current)
			return
		}
		if err := n.acceptCommentByte(current, offset); err != nil {
			n.fail(err)
			return
		}
		n.appendOutput(' ')
		return

	case streamBlockComment:
		if err := n.acceptCommentByte(current, offset); err != nil {
			n.fail(err)
			return
		}
		if current == '\r' || current == '\n' {
			n.appendOutput(current)
		} else {
			n.appendOutput(' ')
		}
		if n.blockStar && current == '/' {
			n.blockStar = false
			n.mode = streamNormal
			return
		}
		n.blockStar = current == '*'
		return
	}

	if n.pendingSlash {
		n.pendingSlash = false
		switch current {
		case '/':
			n.appendOutput(' ')
			n.appendOutput(' ')
			n.mode = streamLineComment
			n.resetCommentUTF8()
			return
		case '*':
			n.appendOutput(' ')
			n.appendOutput(' ')
			n.mode = streamBlockComment
			n.blockStar = false
			n.blockStart = offset - 1
			n.resetCommentUTF8()
			return
		default:
			n.appendSignificant('/')
			// The current byte has not been consumed by the lexer.
		}
	}

	if isJSONWhitespace(current) {
		n.appendOutput(current)
		return
	}

	switch current {
	case '"':
		n.appendSignificant(current)
		n.stringRole, n.stringRoleDepth = nextStringRole(n.frames, n.structureDisabled)
		n.mode = streamString
		n.stringEscaped = false
	case '/':
		n.pendingSlash = true
	case ',':
		n.acceptComma()
	case ']', '}':
		n.acceptClose(current)
	case '[', '{':
		n.appendSignificant(current)
		n.pushFrame(current)
	case ':':
		n.appendSignificant(current)
		n.acceptColon()
	default:
		n.appendSignificant(current)
		if isScalarStart(current) && !n.structureDisabled {
			markValueSeen(n.frames)
		}
	}
}

func (n *streamNormalizer) acceptComma() {
	if n.commaPending {
		n.resolveComma(false)
	}

	if !n.structureDisabled && len(n.frames) > 0 {
		frame := &n.frames[len(n.frames)-1]
		if frameHasValue(*frame) {
			n.commaPending = true
			n.pendingDepth = len(n.frames)
			n.pendingClose = matchingClose(frame.opener)
			n.pending = append(n.pending[:0], ',')
		}
		frame.keySeen = false
		frame.colonSeen = false
		frame.valueSeen = false
	}

	if !n.commaPending {
		n.ready = append(n.ready, ',')
	}
}

func (n *streamNormalizer) acceptClose(close byte) {
	if n.commaPending {
		n.resolveComma(!n.structureDisabled && len(n.frames) == n.pendingDepth && close == n.pendingClose)
	}
	n.ready = append(n.ready, close)
	n.popFrame(close)
}

func (n *streamNormalizer) resolveComma(trailing bool) {
	if !n.commaPending {
		return
	}
	if trailing {
		n.pending[0] = ' '
	}
	if len(n.ready) == 0 {
		// A comma lookahead can span an arbitrarily large comment. Transfer
		// that allocation into the committed queue instead of retaining two
		// attacker-sized buffers.
		n.ready = n.pending
		n.pending = nil
	} else {
		// If ready is non-empty, the comma was resolved within the current
		// bounded source read, so pending cannot exceed maxStreamRead.
		n.ready = append(n.ready, n.pending...)
		n.pending = n.pending[:0]
	}
	n.commaPending = false
}

func (n *streamNormalizer) appendSignificant(current byte) {
	if n.commaPending {
		n.resolveComma(false)
	}
	n.ready = append(n.ready, current)
}

func (n *streamNormalizer) appendOutput(current byte) {
	if n.commaPending {
		n.pending = append(n.pending, current)
		return
	}
	n.ready = append(n.ready, current)
}

func (n *streamNormalizer) finishStringValue() {
	if n.structureDisabled || len(n.frames) != n.stringRoleDepth || n.stringRoleDepth == 0 {
		n.stringRole = stringRoleNone
		return
	}
	frame := &n.frames[n.stringRoleDepth-1]
	switch n.stringRole {
	case stringRoleObjectKey:
		frame.keySeen = true
	case stringRoleValue:
		frame.valueSeen = true
	}
	n.stringRole = stringRoleNone
}

func (n *streamNormalizer) pushFrame(open byte) {
	if n.structureDisabled {
		return
	}
	if len(n.frames) >= maxTrackedNestingDepth {
		n.structureDisabled = true
		n.frames = nil
		return
	}
	n.frames = append(n.frames, containerFrame{
		opener:              open,
		countsAsParentValue: canStartValue(n.frames),
	})
}

func (n *streamNormalizer) popFrame(close byte) {
	if n.structureDisabled || len(n.frames) == 0 {
		return
	}
	last := len(n.frames) - 1
	if !matchingDelimiter(n.frames[last].opener, close) {
		return
	}
	countsAsValue := n.frames[last].countsAsParentValue
	n.frames = n.frames[:last]
	if countsAsValue {
		markValueSeen(n.frames)
	}
}

func (n *streamNormalizer) acceptColon() {
	if n.structureDisabled || len(n.frames) == 0 {
		return
	}
	frame := &n.frames[len(n.frames)-1]
	if frame.opener == '{' && frame.keySeen && !frame.colonSeen {
		frame.colonSeen = true
	}
}

func (n *streamNormalizer) acceptCommentByte(current byte, offset int64) error {
	if n.utf8Len == 0 && current < utf8.RuneSelf {
		return nil
	}
	if n.utf8Len == 0 {
		n.utf8Offset = offset
	}
	if n.utf8Len == len(n.utf8Buf) {
		return n.invalidUTF8Error()
	}
	n.utf8Buf[n.utf8Len] = current
	n.utf8Len++

	buffered := n.utf8Buf[:n.utf8Len]
	if !utf8.FullRune(buffered) {
		return nil
	}
	r, size := utf8.DecodeRune(buffered)
	if r == utf8.RuneError && size == 1 {
		return n.invalidUTF8Error()
	}
	n.utf8Len = 0
	return nil
}

func (n *streamNormalizer) finishCommentUTF8() error {
	if n.utf8Len == 0 {
		return nil
	}
	return n.invalidUTF8Error()
}

func (n *streamNormalizer) resetCommentUTF8() {
	n.utf8Len = 0
	n.utf8Offset = 0
}

func (n *streamNormalizer) invalidUTF8Error() error {
	return &JSONCSyntaxError{Offset: n.utf8Offset, Err: ErrInvalidUTF8}
}

func (n *streamNormalizer) finish(readErr error) {
	if readErr == io.EOF {
		if n.mode == streamLineComment || n.mode == streamBlockComment {
			if err := n.finishCommentUTF8(); err != nil {
				n.fail(err)
				return
			}
		}
		if n.mode == streamBlockComment {
			n.fail(&JSONCSyntaxError{
				Offset: n.blockStart,
				Err:    ErrUnterminatedBlockComment,
			})
			return
		}
		if n.mode == streamLineComment {
			n.mode = streamNormal
		}
	}

	if n.pendingSlash {
		n.pendingSlash = false
		n.appendSignificant('/')
	}
	if n.commaPending {
		n.resolveComma(false)
	}
	n.terminalErr = readErr
}

func (n *streamNormalizer) fail(err error) {
	n.terminalErr = err
	n.commaPending = false
	n.pendingDepth = 0
	n.pendingClose = 0
	n.pending = nil
}

func (n *streamNormalizer) copyReady(dst []byte) int {
	if n.readyAt == len(n.ready) {
		if cap(n.ready) > maxStreamRead {
			// Do not retain an attacker-sized lookahead allocation after the
			// standard decoder has consumed it.
			n.ready = nil
		} else {
			n.ready = n.ready[:0]
		}
		n.readyAt = 0
		return 0
	}
	count := copy(dst, n.ready[n.readyAt:])
	n.readyAt += count
	return count
}

// committed returns normalized output which has been produced but not yet read.
// It intentionally excludes a trailing-comma candidate awaiting lookahead.
func (n *streamNormalizer) committed() []byte {
	return n.ready[n.readyAt:]
}
