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

## Phase 2: Istanbul/C8 Coverage (vitest, jest, playwright)

One adapter covers three frameworks — vitest, jest, and playwright all produce Istanbul-compatible JSON.

### How to generate

```bash
# Vitest
npx vitest --coverage --coverage.reporter=json    # produces coverage/coverage-final.json

# Jest
npx jest --coverage --coverageReporters=json       # produces coverage/coverage-final.json

# Playwright (requires instrumented app)
# Uses Istanbul via babel-plugin-istanbul or vite-plugin-istanbul
# Produces coverage/coverage-final.json
```

### Istanbul JSON format

```json
{
  "src/services/auth.ts": {
    "path": "src/services/auth.ts",
    "statementMap": { "0": {"start":{"line":1,"column":0},"end":{"line":1,"column":30}}, ... },
    "s": { "0": 5, "1": 0, "2": 3, ... },
    "fnMap": { ... },
    "f": { "0": 5, "1": 0, ... },
    "branchMap": { ... },
    "b": { ... }
  }
}
```

Per-file: `s` = statement hit counts (keys match `statementMap`). Count `s[k] > 0` for covered statements.

### New adapter

`internal/adapters/istanbul_cover_profile.go` — implements `CoverageProfileReader`:
- Parses `coverage-final.json` (or `coverage/coverage-final.json`)
- Extracts per-file statement coverage from `s` map
- Produces the same `CoverageProfile` type as the Go adapter

### CLI usage

```bash
testreg audit --coverprofile coverage/coverage-final.json          # auto-detects format
testreg audit --coverprofile cover.out                              # Go format
testreg audit --coverprofile cover.out --coverprofile coverage/coverage-final.json  # both
```

Or in `.testreg.yaml`:
```yaml
graph:
  coverprofiles:
    - cover.out                          # Go
    - coverage/coverage-final.json       # Istanbul (vitest/jest/playwright)
```

Multiple profiles are merged into a single `CoverageProfile`. File paths are deduplicated (Go and TS files never collide).

---

## Phase 3: Python Coverage (pytest + coverage.py)

```bash
# Generate
pytest --cov=src --cov-report=json    # produces coverage.json
```

### coverage.py JSON format

```json
{
  "files": {
    "src/services/auth.py": {
      "executed_lines": [1, 2, 5, 8],
      "missing_lines": [3, 4, 6, 7],
      "summary": { "covered_lines": 4, "num_statements": 8, "percent_covered": 50.0 }
    }
  }
}
```

New adapter: `internal/adapters/python_cover_profile.go` — same port, same `CoverageProfile` output.

---

## Framework Support Boundaries

### What testreg traces (call graph) vs what it tracks (coverage)

| Capability | React Router | Next.js | Nest.js | Remix | Vue |
|------------|-------------|---------|---------|-------|-----|
| **Route auto-discovery** | Yes | No | No | No | No |
| **Component/hook tracing** | Yes | No | No | No | No |
| **@testreg annotations** | Yes | Yes | Yes | Yes | Yes |
| **Test file scanning** | Yes | Yes | Yes | Yes | Yes |
| **Coverage enrichment** | Yes (Istanbul) | Yes (Istanbul) | Yes (Istanbul) | Yes (Istanbul) | Yes (Istanbul) |

Key insight: **annotation-based tracking and coverage enrichment work with any framework**. The framework-specific limitation is only in the call graph auto-discovery (the ts-scanner.ts).

### What would be needed for new frameworks

| Framework | Call graph support | Effort | Notes |
|-----------|-------------------|--------|-------|
| **Next.js** | Scan `app/` directory for `page.tsx`, `route.ts`, `layout.tsx` | Medium | File-system routing, no AST needed for route discovery. Server components add complexity. |
| **Nest.js** | Parse `@Controller`, `@Get`, `@Post`, `@Injectable` decorators | Medium | TypeScript decorator AST parsing. Similar to Go `@api` annotations but in decorators. |
| **Remix** | Scan route directory for `loader`, `action` exports | Medium | Similar to Next.js — file convention + export detection. |
| **Vue/Nuxt** | Parse `.vue` SFCs + Vue Router config | Hard | Needs Vue SFC parser (not just TypeScript AST). |

### Recommended priority for framework support

1. **Next.js** — largest React ecosystem, file-based routing is simpler than AST parsing
2. **Nest.js** — most popular TypeScript backend framework, decorator pattern is parseable
3. **Remix** — similar to Next.js, smaller effort after Next.js patterns are built
4. **Vue/Nuxt** — different component model, lowest priority unless user has Vue projects

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
