//go:build ignore

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

// Command generate_medium creates the project-original medium test fixtures.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const serviceCount = 256

type fixture struct {
	Notice    string            `json:"_notice"`
	CSS       css               `json:"css"`
	Workloads []workload        `json:"workloads"`
	Metadata  map[string]string `json:"metadata"`
	Notes     []string          `json:"notes"`
}

type css struct {
	EditorSuggestInsertMode string `json:"editor.suggest.insertMode"`
}

type workload struct {
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	Replicas     int               `json:"replicas"`
	Endpoint     string            `json:"endpoint"`
	RetryWindows []int             `json:"retryWindows"`
	Limits       limits            `json:"limits"`
	Labels       map[string]string `json:"labels"`
}

type limits struct {
	RequestsPerSecond int `json:"requestsPerSecond"`
	Burst             int `json:"burst"`
	TimeoutMillis     int `json:"timeoutMillis"`
}

func main() {
	directory := "testdata"
	if len(os.Args) == 2 {
		directory = os.Args[1]
	} else if len(os.Args) > 2 {
		panic("usage: go run ./testdata/generate_medium.go [output-directory]")
	}

	withMarkers := newFixture([]string{
		"line marker // remains string data",
		"block marker /* remains string data */",
	})
	withoutMarkers := newFixture([]string{
		"line marker remains string data",
		"block marker remains string data",
	})

	strict := mustJSON(withMarkers)
	mustWrite(filepath.Join(directory, "medium_uncommented.json"), strict)
	mustWrite(filepath.Join(directory, "medium_no_comment_runes.json"), mustJSON(withoutMarkers))
	mustWrite(filepath.Join(directory, "medium.json"), withComments(strict))
}

func newFixture(notes []string) fixture {
	workloads := make([]workload, serviceCount)
	for index := range workloads {
		workloads[index] = workload{
			Name:         fmt.Sprintf("service-%03d", index),
			Enabled:      index%5 != 0,
			Replicas:     index%7 + 1,
			Endpoint:     fmt.Sprintf("service-%03d.internal.example", index),
			RetryWindows: []int{25 + index%10, 100 + index%25, 500 + index%100},
			Limits: limits{
				RequestsPerSecond: 1000 + index,
				Burst:             50 + index%50,
				TimeoutMillis:     1500 + index%500,
			},
			Labels: map[string]string{
				"owner":  fmt.Sprintf("team-%02d", index%16),
				"tier":   fmt.Sprintf("tier-%d", index%4),
				"region": fmt.Sprintf("region-%d", index%6),
			},
		}
	}

	return fixture{
		Notice:    "This file was modified by FloraSync in 2026.",
		CSS:       css{EditorSuggestInsertMode: "replace"},
		Workloads: workloads,
		Metadata: map[string]string{
			"generator": "testdata/generate_medium.go",
			"purpose":   "synthetic JSONC parser workload",
		},
		Notes: notes,
	}
}

func mustJSON(value fixture) []byte {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func withComments(strict []byte) []byte {
	lines := bytes.Split(bytes.TrimSuffix(strict, []byte{'\n'}), []byte{'\n'})
	var output bytes.Buffer
	for index, line := range lines {
		output.Write(line)
		output.WriteByte('\n')
		if index < len(lines)-1 {
			fmt.Fprintf(&output, "  // project-original synthetic fixture line %d\n", index+1)
		}
	}
	return output.Bytes()
}

func mustWrite(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
}
