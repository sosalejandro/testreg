# Coverage Enrichment Plan — Statement-Level Health Scoring

**Date:** 2026-03-31
**Status:** Designed, not implemented
**Problem:** testreg says 100% health when test files exist, but `go test -cover` shows 64-73% statement coverage. Shallow tests that only cover the happy path get full credit.

---

## What Changes

Today: "Does a test file exist near this source file?" → tested/partial/untested
After: "What percentage of statements in this file are actually covered by tests?" → real coverage %

The health score becomes a **blend** of file-existence (40%) and statement coverage (60%) when a coverprofile is available. Without one, behavior is unchanged.

---

## How It Works

```bash
# Step 1: Generate coverage data (standard Go toolchain)
go test -coverprofile=cover.out ./...

# Step 2: Run audit with coverage enrichment
testreg audit --coverprofile cover.out
testreg audit auth.login --coverprofile cover.out
```

The `--coverprofile` flag is optional. testreg also checks:
1. `coverprofile:` field in `.testreg.yaml`
2. `.testreg-cache/coverage.out` (well-known location)

---

## Example: Before vs After

**Before** (file-existence only):
```
POST /api/v1/auth/login  ✓ tested (auth_handler_test.go)
└─ AuthService.Login  ✓ tested (auth_service_test.go)
   └─ UserRepo.FindByEmail  ◐ partial (user_repo_test.go)

Health: 74%
```

**After** (with --coverprofile):
```
POST /api/v1/auth/login  ✓ tested 73%  (auth_handler_test.go)
└─ AuthService.Login  ◐ partial 12%  (auth_service_test.go)
   └─ UserRepo.FindByEmail  ◐ partial 0%  (user_repo_test.go)

Coverage by Layer:
  Handler:     1/1  (100%)  ████████████████████  stmt: 73%
  Service:     0/1  (  0%)  ░░░░░░░░░░░░░░░░░░░░  stmt: 12%

Health: 46%
```

AuthService.Login had a test file (showed "tested") but only 12% of its statements were covered → demoted to "partial." Health drops from 74% to 46%, reflecting reality.

---

## Architecture

Follows existing hexagonal patterns — no new dependencies.

### New Files (5)

| File | Purpose |
|------|---------|
| `internal/domain/coverage_profile.go` | `FileCoverage`, `CoverageProfile` types |
| `internal/ports/coverage_profile.go` | `CoverageProfileReader` interface |
| `internal/adapters/go_cover_profile.go` | Parses `go test -coverprofile` output |
| `internal/adapters/go_cover_profile_test.go` | Parser tests |
| `internal/domain/coverage_profile_test.go` | Domain type tests |

### Modified Files (7)

| File | Change |
|------|--------|
| `internal/domain/audit.go` | Add `StatementCoverage *float64` to `AnnotatedNode`, `StatementPct *float64` to `LayerCoverage`, `CoverageProfileUsed bool` to `AuditOutput` |
| `internal/app/audit_feature.go` | Add `SetCoverageProfile()`, `enrichNodesWithCoverage()`, blend statement coverage into `calculateHealthScore()` |
| `internal/app/audit_feature_test.go` | Tests for enrichment and blended scoring |
| `internal/adapters/graph_config.go` | Add `Coverprofile string` to `GraphSection` |
| `internal/ports/graph.go` | Add `Coverprofile string` to `GraphConfig` |
| `cmd/audit.go` | Add `--coverprofile` flag, resolution logic, `detectModulePrefix()` |
| `internal/adapters/audit_renderer.go` | Show `stmt: X%` in layer bars, percentage in trace annotations, optional "Stmt" column in summary table |

### Config Addition

```yaml
# .testreg.yaml
graph:
  coverprofile: cover.out  # path to go test -coverprofile output
```

---

## Scoring Formula

When coverprofile is available, health = weighted average of blended layer scores:

```
layer_score = 0.4 × file_existence_pct + 0.6 × statement_coverage_pct
health = Σ(layer_weight × layer_score) / Σ(layer_weight)
```

Demotion rule: a node marked "tested" with <20% statement coverage is demoted to "partial."

Without coverprofile: `health = file_existence_pct` (unchanged from today).

---

## Stale Data Handling

Coverprofile mtime is checked. If older than 24 hours:
```
WARNING: coverprofile is 3 day(s) old. Run: go test -coverprofile=cover.out ./...
```

---

## Multi-Language (Phase 2, deferred)

The `CoverageProfileReader` port abstracts the format. Go is Phase 1. Future adapters:

| Language | Coverage Tool | Output Format |
|----------|--------------|---------------|
| Go | `go test -coverprofile` | Custom text format |
| TypeScript | `vitest --coverage`, Istanbul | JSON (lcov/clover) |
| Python | `coverage.py` | JSON |

Each adapter implements the same port, producing `CoverageProfile` with file-level percentages. The enrichment logic in `audit_feature.go` is language-agnostic.

---

## Implementation Sequence

1. **Domain types** — new file, no behavior change, all tests pass
2. **Port + adapter** — coverprofile parser with tests
3. **Config plumbing** — add field to GraphSection, GraphConfig, ToPortsConfig
4. **App layer** — enrichment logic, blended scoring, demotion rule
5. **CLI flag** — `--coverprofile`, resolution chain, staleness warning
6. **Renderers** — statement percentage display in all output formats
7. **Self-test** — run `go test -coverprofile` on testreg itself, verify enriched audit

---

## Verification

```bash
# Generate profile
go test -coverprofile=cover.out ./...

# Run enriched audit
testreg audit --coverprofile cover.out
testreg audit --coverprofile cover.out --summary
testreg audit auth.login --coverprofile cover.out

# Verify: features with shallow tests should show lower health
# Verify: features with deep tests should show higher health
# Verify: without --coverprofile, output is identical to today
```
