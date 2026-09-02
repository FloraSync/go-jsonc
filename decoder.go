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

// Decoder reads JSON or JSONC values from an input stream.
//
// Decoder preserves encoding/json's streaming behavior while presenting a
// byte-length-preserving normalized view to the standard decoder. As with
// encoding/json.Decoder, a Decoder is not safe for concurrent use.
type Decoder struct {
	decoder    *stdjson.Decoder
	normalizer *streamNormalizer
}

// NewDecoder returns a decoder that reads JSON or JSONC values from r.
//
// The decoder introduces its own buffering and may read data from r beyond the
// values requested.
func NewDecoder(r io.Reader) *Decoder {
	normalizer := newStreamNormalizer(r)
	return &Decoder{
		decoder:    stdjson.NewDecoder(normalizer),
		normalizer: normalizer,
	}
}

// Decode reads the next JSON-encoded or JSONC-encoded value from its input and
// stores it in the value pointed to by v.
func (d *Decoder) Decode(v any) error {
	return d.decoder.Decode(v)
}

// Buffered returns a reader of committed normalized data remaining in the
// Decoder's buffers. The reader is valid until the next call to Decode.
//
// A trailing-comma candidate whose disposition depends on unread input is not
// committed until the normalizer encounters the following significant byte.
func (d *Decoder) Buffered() io.Reader {
	return io.MultiReader(
		d.decoder.Buffered(),
		bytes.NewReader(d.normalizer.committed()),
	)
}

// Token returns the next JSON token in the input stream.
func (d *Decoder) Token() (Token, error) {
	return d.decoder.Token()
}

// More reports whether there is another element in the current array or
// object being parsed.
func (d *Decoder) More() bool {
	return d.decoder.More()
}

// InputOffset returns the original input stream byte offset of the current
// decoder position.
func (d *Decoder) InputOffset() int64 {
	return d.decoder.InputOffset()
}

// UseNumber causes the Decoder to unmarshal a number into an interface value
// as a Number instead of as a float64.
func (d *Decoder) UseNumber() {
	d.decoder.UseNumber()
}

// DisallowUnknownFields causes the Decoder to return an error when the
// destination is a struct and an object contains an unknown key.
func (d *Decoder) DisallowUnknownFields() {
	d.decoder.DisallowUnknownFields()
}
