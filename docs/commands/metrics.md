# testreg metrics

Analyze historical test run data and surface quality signals including slowest tests, flaky tests, memory-intensive tests, race conditions, and declining health trends.

Requires `testreg update --with-metrics` to have been run previously to build history.

## Usage

```
testreg metrics [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--feature` | (none) | Show health trend for a specific feature ID |
| `--slow` | (none) | Show tests slower than this duration (e.g., `5s`, `500ms`) |
| `--flaky` | false | Show only flaky tests |
| `--races` | false | Show only race conditions detected |
| `--format` | `text` | Output format: `text`, `json` |

## Examples

### All quality signals

```
$ testreg metrics

Quality Signals:

  Slowest Tests:
    12.3s  TestFullAuthFlow              src/handlers/auth_e2e_test.go
     8.7s  TestDatabaseMigration         src/repo/migration_test.go
     5.1s  TestRecipeSearchIntegration   src/services/recipe_test.go

  Flaky Tests (passed then failed within 7 days):
    auth.login:  TestLoginRateLimit       (3 flips in 7 days)
    meals.log:   TestMealLogConcurrent    (2 flips in 7 days)

  Race Conditions:
    (none detected)
```

### Feature health trend

```
$ testreg metrics --feature auth.login

Health Trend: auth.login
  2026-03-01   0%
  2026-03-08  45%
  2026-03-15  74%
  2026-03-22  74%
  2026-03-29 100%
```

### Slow tests

```bash
testreg metrics --slow 5s
```

### Flaky tests only

```bash
testreg metrics --flaky
```

## Tips

- Run `testreg update --with-metrics` in CI after each test run to build history
- Use `--slow` to find tests that are candidates for optimization
- Flaky tests are detected by tracking pass/fail state changes over time
- Health trends show whether your testing effort is actually improving things
