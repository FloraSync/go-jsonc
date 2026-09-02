# Test fixture provenance

The `medium*.json` fixture family is project-original synthetic data. It does
not contain Visual Studio Code settings or other third-party fixture content.

Regenerate the family deterministically from the repository root with:

```sh
go run ./testdata/generate_medium.go
```

The generator constructs the strict `medium_uncommented.json` workload, derives
`medium.json` by inserting JSONC line comments only at whitespace boundaries,
and produces `medium_no_comment_runes.json` from the same model with comment
marker characters removed from string values.
