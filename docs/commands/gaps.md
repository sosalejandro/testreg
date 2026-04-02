# testreg gaps

Extract actionable test gap information in structured formats. Designed for feeding into automated test-writing workflows and AI subagents.

## Usage

```
testreg gaps [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--feature` | (all) | Show gaps for a specific feature |
| `--format` | `terminal` | Output format: `terminal`, `json`, `actionable`, `prompt` |
| `-n, --limit` | (all) | Limit number of features shown |
| `--min-health` | `0.0` | Only features below this health score |
| `--priority` | (all) | Filter features by priority (comma-separated) |

## Examples

### Terminal output

```
$ testreg gaps --feature auth.login

Feature: auth.login (critical, health: 74%)
  ✘ [CRITICAL] authRepository.StoreRefreshToken — no unit test
  ✘ [HIGH]     repositories.HashToken — no unit test
  ✘ [MEDIUM]   sql:GetUserByEmail — no query-level test
```

### Actionable format (for humans)

```
$ testreg gaps --feature auth.login --format actionable

Feature: auth.login (critical, health: 74%)
  [CRITICAL] authService.Login -- no unit test
    File: src/application/services/auth_service.go:172
    Action: Write unit test for authService.Login
    Pattern: table-driven test with mock repository

  [HIGH] authRepository.StoreRefreshToken -- no integration test
    File: src/domain/repositories/auth_repository.go:329
    Action: Write integration test for authRepository.StoreRefreshToken
    Pattern: TestMain setup with test database
```

### Prompt format (for AI agents)

```
$ testreg gaps --feature auth.login --format prompt

## Feature: auth.login
Priority: critical | Health: 74% | Target: 100%

### Gaps (3):
1. CRITICAL: authService.Login has no unit test for service method
   - Source: src/application/services/auth_service.go:172
   - Write: Write unit test for authService.Login in src/application/services/auth_service_test.go
   - Annotation: // @testreg auth.login #real
```

### AI workflow

```bash
# Extract gaps for top 5 critical features
testreg gaps --priority critical -n 5 --format prompt > /tmp/test-gaps.md

# Feed to AI agent, then verify
testreg scan
testreg audit --priority critical
testreg diff
```

## Format comparison

| Format | Use case | Output |
|--------|----------|--------|
| `terminal` | Human reading in terminal | Color-coded severity + file locations |
| `json` | CI pipelines, scripting | Structured JSON array |
| `actionable` | Human following step-by-step | Per-gap fix instructions with test patterns |
| `prompt` | AI agent input | Markdown with source, write target, and annotation |

## Tips

- `--format prompt` is specifically designed for AI coding agents -- each gap includes file paths, annotations, and test patterns.
- Combine with `testreg sprint` to prioritize which gaps to fix first.
