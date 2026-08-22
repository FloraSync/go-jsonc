#!/usr/bin/env bash

# Run all benchmarks against the standard-library facade.
set -euo pipefail

export GOWORK=off

# Set the path to the temporary directory for the benchmark results.
# The path is relative to the root of the repository.
benchmarkDir="$(dirname "${BASH_SOURCE[0]}")/../tmp/benchmarks"

# Create the temporary directory for the benchmark results
# if it does not exist and add a .gitignore file to it.
mkdir -p "$benchmarkDir"
echo "*" >"$benchmarkDir/.gitignore"

# Set the number of times to run each benchmark.
# The default value is 10.
# It may be overridden by passing a value as the first argument.
count=${1:-10}

# Run the JSONC normalization and facade benchmarks.
echo "
Running JSONC facade benchmarks"
go test -run='^$' -bench=. -benchmem -count "$count" |
  tee "$benchmarkDir/jsonc_profile.txt"

echo "
Running encoding/json baseline benchmarks on uncommented JSON"
go test -run='^$' -bench=BenchmarkUnmarshal -benchmem -count "$count" -tags=uncommented_test |
  tee "$benchmarkDir/encoding_json.txt"
