// Copyright 2023 Marco Zaccaro. All Rights Reserved.
// This file was modified by FloraSync in 2026.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	_ "embed"
	stdjson "encoding/json"
	"reflect"
	"testing"
)

var (
	//go:embed testdata/small.json
	_small []byte

	//go:embed testdata/small_uncommented.json
	_smallUncommented []byte

	//go:embed testdata/small_no_comment_runes.json
	_smallNoCommentRunes []byte

	//go:embed testdata/medium.json
	_medium []byte

	//go:embed testdata/medium_uncommented.json
	_mediumUncommented []byte

	//go:embed testdata/medium_no_comment_runes.json
	_mediumNoCommentRunes []byte

	_invalidChar = []byte("\xa5")
)

func FieldsValue[T DataType](t testing.TB, j T) {
	t.Helper()
	switch j := any(j).(type) {
	case Small:
		var w Small
		mustStandardUnmarshal(t, _smallUncommented, &w)
		if !reflect.DeepEqual(w, j) {
			t.Fatalf("unmarshaled JSON = %#v, want %#v", j, w)
		}
		w.X = "x" // ensure fields are checked
		if reflect.DeepEqual(w, j) {
			t.Fatal("test mutation did not change Small value")
		}
	case Medium:
		var w Medium
		mustStandardUnmarshal(t, _mediumUncommented, &w)
		if !reflect.DeepEqual(w, j) {
			t.Fatal("unmarshaled medium JSON differs from strict fixture")
		}
		w.CSS.EditorSuggestInsertMode = "insert_replace" // ensure fields are checked
		if reflect.DeepEqual(w, j) {
			t.Fatal("test mutation did not change Medium value")
		}
	case SmallNoCommentRunes:
		var w SmallNoCommentRunes
		mustStandardUnmarshal(t, _smallNoCommentRunes, &w)
		if !reflect.DeepEqual(w, j) {
			t.Fatalf("unmarshaled JSON = %#v, want %#v", j, w)
		}
		w.X = "x" // ensure fields are checked
		if reflect.DeepEqual(w, j) {
			t.Fatal("test mutation did not change SmallNoCommentRunes value")
		}
	case MediumNoCommentRunes:
		var w MediumNoCommentRunes
		mustStandardUnmarshal(t, _mediumNoCommentRunes, &w)
		if !reflect.DeepEqual(w, j) {
			t.Fatal("unmarshaled medium JSON differs from strict fixture")
		}
		w.CSS.EditorSuggestInsertMode = "insert_replace" // ensure fields are checked
		if reflect.DeepEqual(w, j) {
			t.Fatal("test mutation did not change MediumNoCommentRunes value")
		}
	default:
		t.Fatalf("unexpected data type %T", j)
	}
}

func mustStandardUnmarshal(t testing.TB, data []byte, destination any) {
	t.Helper()
	if err := stdjson.Unmarshal(data, destination); err != nil {
		t.Fatalf("encoding/json.Unmarshal() error = %v", err)
	}
}

func TestHasCommentRunes(t *testing.T) {
	t.Parallel()
	for _, tt := range hasCommentRunesTests {
		tt := tt
		t.Run(tt.Name, func(t *testing.T) {
			t.Parallel()
			if got := HasCommentRunes(tt.Data); got != tt.Want {
				t.Fatalf("HasCommentRunes(%q) = %t, want %t", tt.Data, got, tt.Want)
			}
		})
	}
}

var hasCommentRunesTests = [...]struct {
	Name string
	Data []byte
	Want bool
}{
	{"Small/Commented", _small, true},
	{"Small/Uncommented", _smallUncommented, false},
	{"Small/NoCommentRunes", _smallNoCommentRunes, false},
	{"Medium/Commented", _medium, true},
	{"Medium/Uncommented", _mediumUncommented, false},
	{"Medium/NoCommentRunes", _mediumNoCommentRunes, false},
	{"Line/Incomplete", []byte("//"), true},
	{"Block/Incomplete", []byte("/*"), true},
	{"LoneSlash", []byte("/"), false},
	{"StrayCloser", []byte("*/"), false},
	{"String/LineMarker", []byte(`{"s":"//"}`), false},
	{"String/BlockMarker", []byte(`{"s":"/*"}`), false},
	{"String/OddBackslashParity", []byte(`{"s":"\\\" // inside"}`), false},
	{"String/EvenBackslashParity", []byte(`{"s":"\\\\" /* outside */}`), true},
	{"Binary/OutsideThenComment", []byte{0xff, '/', '/', 'x'}, true},
}

func BenchmarkHasCommentRunes(b *testing.B) {
	for _, tt := range hasCommentRunesTests {
		tt := tt
		b.Run(tt.Name, func(b *testing.B) {
			if got := HasCommentRunes(tt.Data); got != tt.Want {
				b.Fatalf("HasCommentRunes(%q) = %t, want %t", tt.Data, got, tt.Want)
			}
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					if got := HasCommentRunes(tt.Data); got != tt.Want {
						b.Fatalf("HasCommentRunes(%q) = %t, want %t", tt.Data, got, tt.Want)
					}
				}
			})
		})
	}
}

type DataType interface {
	Small | SmallNoCommentRunes | Medium | MediumNoCommentRunes
}

type SmallNoCommentRunes Small

type Small struct {
	Foo   string `json:"foo"`
	Baz   string `json:"baz"`
	Hello string `json:"hello"`
	X     string `json:"x,omitempty"`
}

type MediumNoCommentRunes Medium
type Medium struct {
	Notice    string            `json:"_notice"`
	CSS       mediumCSS         `json:"css"`
	Workloads []mediumWorkload  `json:"workloads"`
	Metadata  map[string]string `json:"metadata"`
	Notes     []string          `json:"notes"`
}

type mediumCSS struct {
	EditorSuggestInsertMode string `json:"editor.suggest.insertMode"`
}

type mediumWorkload struct {
	Name         string            `json:"name"`
	Enabled      bool              `json:"enabled"`
	Replicas     int               `json:"replicas"`
	Endpoint     string            `json:"endpoint"`
	RetryWindows []int             `json:"retryWindows"`
	Limits       mediumLimits      `json:"limits"`
	Labels       map[string]string `json:"labels"`
}

type mediumLimits struct {
	RequestsPerSecond int `json:"requestsPerSecond"`
	Burst             int `json:"burst"`
	TimeoutMillis     int `json:"timeoutMillis"`
}
