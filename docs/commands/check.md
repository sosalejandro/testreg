# testreg check

Show detailed coverage for a single feature, including all test entries with status, gap analysis, and actionable suggestions.

## Usage

```
testreg check <feature-id> [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `table` | Output format: `table`, `json` |

## Examples

### Detailed feature check

```
$ testreg check auth.login

Feature: auth.login
Name: User Login
Priority: critical
Description: Email/password authentication with JWT tokens

Coverage:
  Unit:
    Backend:  covered  [src/services/auth_service_test.go] (mocked)
    Web:      missing
  Integration:
    Backend:  covered  [src/handlers/auth_e2e_test.go]
  E2E:
    Web:      covered  [e2e/auth.spec.ts]  (pass_rate: 100%, last_run: 2026-03-30)

Gaps:
  - Missing unit web tests
  - Unit backend tests are mocked — consider adding #real tests

Suggestions:
  1. Add @testreg auth.login to a web unit test file
  2. Write non-mocked unit test for backend auth service
```

## Tips

- `check` shows registry-level detail (what's declared in YAML), not graph-level dependency analysis
- For graph-aware analysis with health scores, use `testreg audit <feature-id>`
- Use `--format json` for programmatic access
