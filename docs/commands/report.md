# testreg report

Generate a comprehensive coverage report from the current registry state.

## Usage

```
testreg report [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `md` | Output format: `md`, `json`, `terminal` |
| `--output` | `docs/testing/COVERAGE.md` | Output file path |

## Examples

### Generate markdown report

```bash
testreg report
# Writes to docs/testing/COVERAGE.md
```

### Terminal output

```bash
testreg report --format terminal
```

### JSON for CI

```bash
testreg report --format json
```

### Custom output path

```bash
testreg report --output ./COVERAGE.md
```

## Tips

- The markdown report is useful for PR descriptions or documentation
- Run `testreg scan` before `testreg report` for fresh data
- JSON format is useful for CI dashboards and automated reporting
