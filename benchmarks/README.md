# Benchmark results

Committed benchmark snapshots were removed when the project dropped its alternative JSON backends and established Go 1.26 as its sole baseline.

Run `./scripts/benchmark.sh` to compare the JSONC facade's strict fast path and normalization path with Go 1.26 `encoding/json` on the current machine. Results are written beneath `tmp/benchmarks/`, which is intentionally untracked because benchmark numbers depend on the exact toolchain, CPU, operating system, and workload.
