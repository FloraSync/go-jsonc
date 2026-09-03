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

//go:build uncommented_test
// +build uncommented_test

package json

import (
	stdjson "encoding/json"
	"testing"
)

// This file does not contain real benchmarks, but it is used to compare the
// performances over the standard functions on uncommented JSON data.

// Check standard json.Unmarshal performance with uncommented JSON data.
func BenchmarkUnmarshal(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		b.Run("UnCommented", func(b *testing.B) {
			benchmarkStandardUnmarshal(b, _smallUncommented, Small{})
		})
		b.Run("NoCommentRunes", func(b *testing.B) {
			benchmarkStandardUnmarshal(b, _smallNoCommentRunes, SmallNoCommentRunes{})
		})
	})
	b.Run("Medium", func(b *testing.B) {
		b.Run("UnCommented", func(b *testing.B) {
			benchmarkStandardUnmarshal(b, _mediumUncommented, Medium{})
		})
		b.Run("NoCommentRunes", func(b *testing.B) {
			benchmarkStandardUnmarshal(b, _mediumNoCommentRunes, MediumNoCommentRunes{})
		})
	})
}

func benchmarkStandardUnmarshal[T DataType](b *testing.B, data []byte, initial T) {
	b.Helper()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			value := initial
			if err := stdjson.Unmarshal(data, &value); err != nil {
				b.Fatal(err)
			}
		}
	})
}
