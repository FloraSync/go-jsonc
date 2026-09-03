//go:build go1.27 && goexperiment.jsonv2

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
	jsonv2 "encoding/json/v2"
	"testing"
)

func TestSanitizeInteroperatesWithEncodingJSONV2(t *testing.T) {
	t.Parallel()

	input := []byte(`{
		// Native v2 semantic decoding follows JSONC normalization.
		"service": "flora",
		"ports": [8080, 8443,],
	}`)

	normalized, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}

	var got struct {
		Service string `json:"service"`
		Ports   []int  `json:"ports"`
	}
	if err := jsonv2.Unmarshal(normalized, &got); err != nil {
		t.Fatalf("jsonv2.Unmarshal() error = %v", err)
	}
	if got.Service != "flora" {
		t.Errorf("Service = %q, want flora", got.Service)
	}
	if len(got.Ports) != 2 || got.Ports[0] != 8080 || got.Ports[1] != 8443 {
		t.Errorf("Ports = %v, want [8080 8443]", got.Ports)
	}
}

func TestSanitizePreservesEncodingJSONV2StrictDefaults(t *testing.T) {
	t.Parallel()

	normalized, err := Sanitize([]byte(`{
		"role": "reader",
		/* Duplicate names remain visible to the semantic decoder. */
		"role": "writer",
	}`))
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}

	var got map[string]string
	if err := jsonv2.Unmarshal(normalized, &got); err == nil {
		t.Fatal("jsonv2.Unmarshal() error = nil, want duplicate-name rejection")
	}
}

func TestSanitizePreservesEncodingJSONV2InvalidUTF8Rejection(t *testing.T) {
	t.Parallel()

	input := []byte{'"', 0xff, '"'}
	normalized, err := Sanitize(input)
	if err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}

	var got string
	if err := jsonv2.Unmarshal(normalized, &got); err == nil {
		t.Fatal("jsonv2.Unmarshal() error = nil, want invalid UTF-8 rejection")
	}
}
