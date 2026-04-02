# testreg update

Ingest test results from CI output and update the registry with pass/fail status, pass rates, and last-run timestamps.

## Usage

```
testreg update [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--playwright` | (none) | Path to Playwright JSON results directory or file |
| `--gotest` | (none) | Path to `go test -json` output file |
| `--maestro` | (none) | Path to Maestro output directory |
| `--with-metrics` | false | Also capture test metrics into history |

## Examples

### Ingest Playwright results

```bash
testreg update --playwright ./test-results/
```

### Ingest Go test results

```bash
go test -json ./... > go-test-output.json
testreg update --gotest ./go-test-output.json
```

### Ingest Maestro results

```bash
testreg update --maestro ./maestro-output/
```

### With metrics history

```bash
testreg update --gotest ./results.json --with-metrics
```

Captures timing, flakiness, and memory data for `testreg metrics` analysis.

## How It Works

1. Parses the test result format (JSON for Playwright/Go, XML for Maestro)
2. Matches test names/paths to features via `@testreg` annotations in the registry
3. Updates coverage entries with: status (covered/failing), pass_rate, last_run date
4. Writes updated registry YAML files

## Tips

- Run in CI after your test suite to keep the registry current
- Use `--with-metrics` to build history for `testreg metrics` quality signals
- Combine with `testreg diff --save-snapshot` for automated progress tracking
