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
	"bytes"
	stdjson "encoding/json"
	"io"
)

type (
	Delim                 = stdjson.Delim
	Encoder               = stdjson.Encoder
	InvalidUTF8Error      = stdjson.InvalidUTF8Error
	InvalidUnmarshalError = stdjson.InvalidUnmarshalError
	Marshaler             = stdjson.Marshaler
	MarshalerError        = stdjson.MarshalerError
	Number                = stdjson.Number
	RawMessage            = stdjson.RawMessage
	SyntaxError           = stdjson.SyntaxError
	Token                 = stdjson.Token
	UnmarshalFieldError   = stdjson.UnmarshalFieldError
	UnmarshalTypeError    = stdjson.UnmarshalTypeError
	Unmarshaler           = stdjson.Unmarshaler
	UnsupportedTypeError  = stdjson.UnsupportedTypeError
	UnsupportedValueError = stdjson.UnsupportedValueError
)

// Compact appends to dst the compacted form of the JSONC-encoded src.
func Compact(dst *bytes.Buffer, src []byte) error {
	src, _, err := normalizeBytes(src)
	if err != nil {
		return err
	}
	return stdjson.Compact(dst, src)
}

// HTMLEscape preserves the behavior of [encoding/json.HTMLEscape]. Its input
// contract remains ordinary JSON because the standard signature cannot report
// malformed JSONC extensions.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	stdjson.HTMLEscape(dst, src)
}

// Indent appends to dst an indented form of the JSONC-encoded src.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	src, _, err := normalizeBytes(src)
	if err != nil {
		return err
	}
	return stdjson.Indent(dst, src, prefix, indent)
}

// Marshal returns the JSON encoding of v.
func Marshal(v any) ([]byte, error) {
	return stdjson.Marshal(v)
}

// MarshalIndent returns the indented JSON encoding of v.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return stdjson.MarshalIndent(v, prefix, indent)
}

// NewEncoder returns a standard JSON encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return stdjson.NewEncoder(w)
}

// Unmarshal parses JSON or JSONC data and stores the result in v.
func Unmarshal(data []byte, v any) error {
	data, _, err := normalizeBytes(data)
	if err != nil {
		return err
	}
	return stdjson.Unmarshal(data, v)
}

// Valid reports whether data is valid under the FloraSync JSONC Profile v1.
func Valid(data []byte) bool {
	data, _, err := normalizeBytes(data)
	return err == nil && stdjson.Valid(data)
}
