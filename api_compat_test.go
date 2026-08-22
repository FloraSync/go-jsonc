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
	"io"
	"reflect"

	"github.com/FloraSync/go-jsonc"
)

// These assignments deliberately use the package's implicit identifier json.
// They fail to compile if replacing only the encoding/json import string does
// not preserve the stable Go 1.26 package surface.
var (
	_ func(*bytes.Buffer, []byte) error                 = json.Compact
	_ func(*bytes.Buffer, []byte)                       = json.HTMLEscape
	_ func(*bytes.Buffer, []byte, string, string) error = json.Indent
	_ func(any) ([]byte, error)                         = json.Marshal
	_ func(any, string, string) ([]byte, error)         = json.MarshalIndent
	_ func(io.Reader) *json.Decoder                     = json.NewDecoder
	_ func(io.Writer) *json.Encoder                     = json.NewEncoder
	_ func([]byte, any) error                           = json.Unmarshal
	_ func([]byte) bool                                 = json.Valid
	_ func(*json.Decoder) io.Reader                     = (*json.Decoder).Buffered
	_ func(*json.Decoder, any) error                    = (*json.Decoder).Decode
	_ func(*json.Decoder)                               = (*json.Decoder).DisallowUnknownFields
	_ func(*json.Decoder) int64                         = (*json.Decoder).InputOffset
	_ func(*json.Decoder) bool                          = (*json.Decoder).More
	_ func(*json.Decoder) (json.Token, error)           = (*json.Decoder).Token
	_ func(*json.Decoder)                               = (*json.Decoder).UseNumber
	_ func(*json.Encoder, any) error                    = (*json.Encoder).Encode
	_ func(*json.Encoder, bool)                         = (*json.Encoder).SetEscapeHTML
	_ func(*json.Encoder, string, string)               = (*json.Encoder).SetIndent
	_ func(json.Delim) string                           = json.Delim.String
	_ func(json.Number) (float64, error)                = json.Number.Float64
	_ func(json.Number) (int64, error)                  = json.Number.Int64
	_ func(json.Number) string                          = json.Number.String
	_ func(json.RawMessage) ([]byte, error)             = json.RawMessage.MarshalJSON
	_ func(*json.RawMessage, []byte) error              = (*json.RawMessage).UnmarshalJSON
	_ interface{ Unwrap() error }                       = (*json.MarshalerError)(nil)
	_ error                                             = (*json.InvalidUTF8Error)(nil)
	_ error                                             = (*json.InvalidUnmarshalError)(nil)
	_ error                                             = (*json.MarshalerError)(nil)
	_ error                                             = (*json.SyntaxError)(nil)
	_ error                                             = (*json.UnmarshalFieldError)(nil)
	_ error                                             = (*json.UnmarshalTypeError)(nil)
	_ error                                             = (*json.UnsupportedTypeError)(nil)
	_ error                                             = (*json.UnsupportedValueError)(nil)
	_ json.Marshaler                                    = json.RawMessage(nil)
	_ json.Unmarshaler                                  = (*json.RawMessage)(nil)
	_ func([]byte) ([]byte, error)                      = json.Sanitize
	_ func([]byte) bool                                 = json.HasCommentRunes
	_ error                                             = json.ErrInvalidUTF8
	_ error                                             = json.ErrUnterminatedBlockComment
	_ error                                             = (*json.JSONCSyntaxError)(nil)
	_ interface{ Unwrap() error }                       = (*json.JSONCSyntaxError)(nil)
	_                                                   = json.InvalidUTF8Error{S: ""}
	_                                                   = json.InvalidUnmarshalError{Type: reflect.TypeFor[int]()}
	_                                                   = json.MarshalerError{Type: reflect.TypeFor[int](), Err: io.EOF}
	_                                                   = json.SyntaxError{Offset: 1}
	_                                                   = json.UnmarshalFieldError{Key: "key", Type: reflect.TypeFor[int](), Field: reflect.StructField{}}
	_                                                   = json.UnmarshalTypeError{Value: "number", Type: reflect.TypeFor[int](), Offset: 1, Struct: "T", Field: "Field"}
	_                                                   = json.UnsupportedTypeError{Type: reflect.TypeFor[chan int]()}
	_                                                   = json.UnsupportedValueError{Value: reflect.ValueOf(1), Str: "value"}
	_                                                   = json.JSONCSyntaxError{Offset: 1, Err: json.ErrInvalidUTF8}
)

// Concrete bidirectional assignments ensure the replacement aliases the
// corresponding standard types instead of defining look-alikes.
var (
	replacementDelim                 json.Delim
	replacementEncoder               *json.Encoder
	replacementInvalidUTF8Error      *json.InvalidUTF8Error
	replacementInvalidUnmarshalError *json.InvalidUnmarshalError
	replacementMarshalerError        *json.MarshalerError
	replacementNumber                json.Number
	replacementRawMessage            json.RawMessage
	replacementSyntaxError           *json.SyntaxError
	replacementUnmarshalFieldError   *json.UnmarshalFieldError
	replacementUnmarshalTypeError    *json.UnmarshalTypeError
	replacementUnsupportedTypeError  *json.UnsupportedTypeError
	replacementUnsupportedValueError *json.UnsupportedValueError

	_ stdjson.Delim                  = replacementDelim
	_ *stdjson.Encoder               = replacementEncoder
	_ *stdjson.InvalidUTF8Error      = replacementInvalidUTF8Error
	_ *stdjson.InvalidUnmarshalError = replacementInvalidUnmarshalError
	_ *stdjson.MarshalerError        = replacementMarshalerError
	_ stdjson.Number                 = replacementNumber
	_ stdjson.RawMessage             = replacementRawMessage
	_ *stdjson.SyntaxError           = replacementSyntaxError
	_ *stdjson.UnmarshalFieldError   = replacementUnmarshalFieldError
	_ *stdjson.UnmarshalTypeError    = replacementUnmarshalTypeError
	_ *stdjson.UnsupportedTypeError  = replacementUnsupportedTypeError
	_ *stdjson.UnsupportedValueError = replacementUnsupportedValueError
)
