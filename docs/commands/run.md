# testreg run

Execute tests associated with a feature. Collects run commands from the registry and executes them sequentially.

## Usage

```
testreg run [feature-id] [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--platform` | (all) | Filter by platform: `backend`, `web`, `mobile` |
| `--type` | (all) | Filter by test type: `unit`, `integration`, `e2e` |
| `--dry-run` | false | Print commands without executing |
| `--failing` | false | Run tests for features with failures only |
| `--priority` | (all) | Run tests by priority level |

## Examples

### Run all tests for a feature

```bash
testreg run auth.login
```

### Backend unit tests only

```bash
testreg run auth.login --platform backend --type unit
```

### Preview without executing

```
$ testreg run auth.login --dry-run

Would execute:
  1. go test -run TestLogin ./src/services/...
  2. go test -run TestAuthHandler ./src/handlers/...
  3. npx playwright test e2e/auth.spec.ts
```

### Run all failing features

```bash
testreg run --failing
```

### Run all critical-priority tests

```bash
testreg run --priority critical
```

## Tips

- Use `--dry-run` to see what would run before executing
- `--failing` is useful in CI to re-run only broken tests
- `--priority critical` runs tests for all critical features — useful as a smoke test
- Flags compose: `--platform backend --type unit` runs only backend unit tests
