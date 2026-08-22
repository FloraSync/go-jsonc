// Copyright 2023 Marco Zaccaro. All Rights Reserved.
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

package json_test

import (
	"fmt"

	"github.com/FloraSync/go-jsonc"
)

func ExampleUnmarshal() {
	var v interface{}

	data := []byte(`{/* comment */"foo": "bar"}`)

	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}

	fmt.Println(v)

	// Output:
	// map[foo:bar]
}

func ExampleUnmarshal_sanitizeError() {
	var v interface{}

	invalid := append([]byte("/*"), byte(0xa5))
	invalid = append(invalid, []byte("*/{}")...)

	err := json.Unmarshal(invalid, &v)
	fmt.Println(err)

	// Output:
	// jsonc: invalid UTF-8 in comment at byte 3
}
