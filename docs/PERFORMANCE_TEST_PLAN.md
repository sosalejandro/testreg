# Performance Test Plan — testreg

**Date:** 2026-03-31
**Current state:** 41/41 features at 100% unit coverage, 371 tests, 0 sprint gaps.
**Performance score:** 5/41 features have any perf coverage (40-60%), 36 have 0%.

---

## Goal

Add `Benchmark*` functions and `t.Parallel()` to test files across the codebase.
This closes the performance gap testreg itself reports (77 perf gaps across 42 test files).

---

## Priority Order

Work highest-impact files first — the ones that run on every `testreg` invocation.

### Tier 1: Hot-path parsers (critical, run on every scan/trace/audit)

| File | What to add | Why |
|------|------------|-----|
| `internal/adapters/go_ast_scanner_test.go` | `BenchmarkBuild`, `BenchmarkBuildFrom`, `t.Parallel()` | Slowest operation in testreg — parses entire Go codebase |
| `internal/adapters/annotation_parser_test.go` | `t.Parallel()` on all tests | Already has `BenchmarkLogin`; needs race coverage |
| `internal/adapters/route_parser_test.go` | `BenchmarkParse`, `BenchmarkParseDir` | Already has `t.Parallel()`; needs benchmark |
| `internal/adapters/yaml_store_test.go` | `BenchmarkLoadAll`, `BenchmarkSaveAll`, `t.Parallel()` | Registry I/O on every command |

### Tier 2: Resolvers and mappers (high, run during trace/audit)

| File | What to add | Why |
|------|------------|-----|
| `internal/adapters/wire_resolver_test.go` | `BenchmarkResolve`, `t.Parallel()` | DI resolution on trace |
| `internal/adapters/fx_resolver_test.go` | `BenchmarkResolve`, `t.Parallel()` | DI resolution on trace |
| `internal/adapters/sqlc_mapper_test.go` | `BenchmarkMap`, `t.Parallel()` | SQLC pre-resolution |
| `internal/adapters/frontend_scanner_test.go` | `t.Parallel()` | Frontend graph merge |

### Tier 3: Domain model and use cases (medium, core logic)

| File | What to add | Why |
|------|------------|-----|
| `internal/domain/graph_test.go` | `BenchmarkTraceFrom`, `BenchmarkAddEdge`, `t.Parallel()` | Graph traversal is O(nodes) |
| `internal/app/audit_feature_test.go` | `BenchmarkExecute`, `t.Parallel()` | Full audit pipeline |
| `internal/app/trace_feature_test.go` | `BenchmarkExecute`, `t.Parallel()` | Full trace pipeline |
| `internal/app/diagnose_feature_test.go` | `t.Parallel()` | Symptom matching |
| `internal/domain/feature_test.go` | `t.Parallel()` | Feature model ops |
| `internal/domain/registry_test.go` | `BenchmarkAllFeatures`, `t.Parallel()` | Registry queries |
| `internal/domain/run_command_test.go` | `t.Parallel()` | Command generation |
| `internal/domain/status_test.go` | `t.Parallel()` | Status validation |

### Tier 4: Scanners and result parsers (medium, run during scan/update)

| File | What to add | Why |
|------|------------|-----|
| `internal/adapters/go_scanner_test.go` | `t.Parallel()` | File discovery |
| `internal/adapters/vitest_scanner_test.go` | `t.Parallel()` | File discovery |
| `internal/adapters/playwright_scanner_test.go` | `t.Parallel()` | File discovery |
| `internal/adapters/jest_scanner_test.go` | `t.Parallel()` | File discovery |
| `internal/adapters/maestro_scanner_test.go` | `t.Parallel()` | File discovery |
| `internal/adapters/go_test_results_test.go` | `BenchmarkParse`, `t.Parallel()` | Result ingestion |
| `internal/adapters/playwright_results_test.go` | `t.Parallel()` | Result ingestion |
| `internal/adapters/metrics_go_parser_test.go` | `t.Parallel()` | Metrics parsing |
| `internal/adapters/metrics_playwright_parser_test.go` | `t.Parallel()` | Metrics parsing |
| `internal/adapters/metrics_vitest_parser_test.go` | `t.Parallel()` | Metrics parsing |
| `internal/adapters/metrics_store_test.go` | `BenchmarkGetQualitySignals`, `t.Parallel()` | Metrics aggregation |

### Tier 5: Renderers and cmd tests (low, output formatting)

| File | What to add | Why |
|------|------------|-----|
| `internal/adapters/audit_renderer_test.go` | `t.Parallel()` | Output rendering |
| `internal/adapters/graph_renderer_test.go` | `t.Parallel()` | Graph rendering |
| `internal/adapters/terminal_reporter_test.go` | `t.Parallel()` | Terminal output |
| `internal/adapters/markdown_reporter_test.go` | `t.Parallel()` | Markdown output |
| `internal/adapters/metrics_renderer_test.go` | `t.Parallel()` | Metrics output |
| `cmd/audit_test.go` | `t.Parallel()` | CLI pure functions |
| `cmd/sprint_test.go` | `t.Parallel()` | CLI pure functions |
| `cmd/gaps_test.go` | `t.Parallel()` | CLI pure functions |
| `cmd/diff_test.go` | `t.Parallel()` | CLI pure functions |
| `internal/app/check_feature_test.go` | `t.Parallel()` | Feature check |
| `internal/app/generate_report_test.go` | `t.Parallel()` | Report generation |
| `internal/app/get_status_test.go` | `t.Parallel()` | Status queries |
| `internal/app/init_registry_test.go` | `t.Parallel()` | Registry init |
| `internal/app/scan_tests_test.go` | `t.Parallel()` | Scan orchestration |
| `internal/app/update_coverage_test.go` | `t.Parallel()` | Coverage update |

---

## Benchmark Guidelines

Each benchmark should follow this pattern:

```go
func BenchmarkGoASTScanner_Build(b *testing.B) {
    // Setup: create a temp project with realistic Go files
    // ...
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        scanner.Build(projectRoot, config)
    }
}
```

For memory-sensitive operations (AST scanner, YAML store), use `-benchmem`:
```bash
go test -benchmem -bench BenchmarkGoASTScanner ./internal/adapters/...
```

---

## Race Test Guidelines

Add `t.Parallel()` as the first line of every top-level test function:

```go
func TestSomething(t *testing.T) {
    t.Parallel()
    // ... rest of test
}
```

For table-driven tests, also parallelize subtests:
```go
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // ... subtest
    })
}
```

Run the full race detector after adding:
```bash
go test -race ./...
```

---

## Verification

After implementation, testreg should report:

```bash
testreg audit --all  # Perf column should show 100% for all features
testreg sprint       # Still 0 gaps (perf gaps are informational, not blocking)
go test -race ./...  # 0 race conditions
go test -bench . -benchmem ./...  # All benchmarks run
```

---

## Estimated Scope

- **`t.Parallel()`**: ~42 files, mechanical — add one line per test function. Can be done with a single subagent dispatch.
- **Benchmarks**: ~15 files need meaningful benchmarks (Tier 1-3). Each requires realistic test fixtures. Estimate 1-2 hours of focused work.
- **Total perf gaps to close**: 77 → 0
