// Copyright 2023 Marco Zaccaro. All Rights Reserved.
// This file was modified by FloraSync in 2026.
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

// Package json implements the stable encoding/json API with support for the
// FloraSync JSONC Profile v1.
//
// The profile adds JavaScript-style line and block comments and permits one
// trailing comma after the final member of a non-empty object or element of a
// non-empty array. All encoding operations emit ordinary JSON.
package json
