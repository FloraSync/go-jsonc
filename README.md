<!-- This file was modified by FloraSync in 2026. -->

# go-jsonc

[![Go Reference](https://pkg.go.dev/badge/github.com/FloraSync/go-jsonc.svg)](https://pkg.go.dev/github.com/FloraSync/go-jsonc)
[![CI](https://github.com/FloraSync/go-jsonc/actions/workflows/ci.yml/badge.svg)](https://github.com/FloraSync/go-jsonc/actions/workflows/ci.yml)

`go-jsonc` is a zero-dependency facade for Go's [`encoding/json`](https://pkg.go.dev/encoding/json) with native JSONC comments and trailing commas.

The module requires Go 1.26.0 or newer. Its package name is `json`, so most callers can migrate by changing only the import path:

```diff
-import "encoding/json"
+import "github.com/FloraSync/go-jsonc"
```

Callers migrating from `github.com/marcozac/go-jsonc` may preserve the local
`jsonc` identifier temporarily by aliasing the FloraSync import:

```go
import jsonc "github.com/FloraSync/go-jsonc"
```

## Supported syntax

The FloraSync JSONC Profile v1 accepts strict RFC 8259 JSON plus:

- `//` line comments, ending at CR, LF, CRLF, or EOF;
- `/* ... */` block comments; and
- one trailing comma after the final member or element of a non-empty object or array.

```jsonc
{
  // Comments are allowed wherever JSON permits whitespace.
  "service": "flora",
  "ports": [8080, 8443,],
}
```

Comment markers inside strings remain string data. Comments cannot split JSON tokens, block comments cannot nest, and unsupported JSON5 features such as single-quoted strings, unquoted keys, hexadecimal numbers, `NaN`, and `Infinity` remain invalid.

Profile v1 is pinned to the [JSONC.org draft at commit `84b0999`](https://github.com/JSONC-org/JSONC/tree/84b09994253d4da0fa8e619cfbe9aa6f6229790d). JSONC.org's default grammar does not enable trailing commas; FloraSync deliberately enables the extension permitted by its prose.

## Usage

```go
package main

import (
    "fmt"

    "github.com/FloraSync/go-jsonc"
)

func main() {
    input := []byte(`{
        // deployment target
        "region": "us-west-2",
    }`)

    var config map[string]string
    if err := json.Unmarshal(input, &config); err != nil {
        panic(err)
    }
    fmt.Println(config["region"])
}
```

`Unmarshal`, `Valid`, `Compact`, `Indent`, and `NewDecoder` accept the FloraSync JSONC profile. Encoding operations delegate directly to the standard library. The complete stable Go 1.26 `encoding/json` API is mirrored, including all documented types and decoder/encoder methods.

`Sanitize` is an additional API for obtaining the normalized JSON view:

```go
normalized, err := json.Sanitize(input)
```

Recognized comments are replaced with same-length whitespace and accepted trailing commas with one space. The input is never mutated and output offsets stay aligned with the original bytes. `ErrInvalidUTF8`, `ErrUnterminatedBlockComment`, and `JSONCSyntaxError` describe JSONC-specific lexical failures.

## Go 1.27 and encoding/json/v2

The v1 module continues to expose the stable `encoding/json` v1 contract on Go
1.26 and Go 1.27. Go 1.27 implements that standard package on the new engine
while preserving v1 behavior.

Applications that deliberately adopt Go 1.27's stricter
[`encoding/json/v2`](https://pkg.go.dev/encoding/json/v2) semantics can compose
it with `Sanitize` without waiting for a second go-jsonc API:

```go
normalized, err := json.Sanitize(input)
if err != nil {
    return err
}
if err := jsonv2.Unmarshal(normalized, &config); err != nil {
    return err
}
```

Here `json` is `github.com/FloraSync/go-jsonc` and `jsonv2` is
`encoding/json/v2`. This explicit boundary preserves v2 defaults such as
rejecting duplicate object names and invalid UTF-8. A future go-jsonc v2 API can
build on the same sanitizer boundary without changing v1 semantics.

## Compatibility and security notes

- Strict slice input is passed to `encoding/json` without allocating a
  normalized buffer; streaming always uses the incremental normalizing reader.
- Invalid UTF-8 outside comments retains the standard library's behavior. Invalid UTF-8 inside comments is rejected.
- The custom JSONC-aware `Decoder` preserves the documented standard methods and normalized byte offsets, but its pointer type is necessarily distinct from `*encoding/json.Decoder`.
- The implementation uses a deterministic linear lexer: no regular expressions, recursion, unsafe code, or third-party Go modules.

When multiple systems parse security-sensitive input, ensure they all use the same JSONC profile. Differences involving duplicate object keys, case-insensitive struct matching, number precision, or other standard `encoding/json` behaviors remain the standard library's responsibility.

## Development

The repository may live beneath a parent Go workspace, so the supplied scripts force module-local operation:

```sh
./scripts/test.sh
./scripts/benchmark.sh
```

## License

This project is an intentional, independent continuation of Marco Zaccaro's
[`github.com/marcozac/go-jsonc`](https://github.com/marcozac/go-jsonc), used and
modified under the Apache License 2.0. The upstream main branch has had no
source change since August 2023. Copyright and license notices for retained
upstream material are preserved. FloraSync's continuation is independently
maintained and is not endorsed by the upstream author.

Licensed under Apache-2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).
