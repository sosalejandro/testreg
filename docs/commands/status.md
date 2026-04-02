# testreg status

Display a terminal table with coverage metrics per domain and platform.

## Usage

```
testreg status [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--domain` | (all) | Filter by domain name |
| `--priority` | (all) | Filter by priority: `critical`, `high`, `medium`, `low` |
| `--format` | `table` | Output format: `table`, `json` |

## Examples

### Full dashboard

```
$ testreg status

┌──────────────────────┬───────┬──────────┬──────────┬──────────┐
│ Domain               │ Total │ Unit     │ Integ.   │ E2E      │
├──────────────────────┼───────┼──────────┼──────────┼──────────┤
│ auth                 │ 5     │ 5/5 OK   │ 0/5 !!   │ 0/5 !!   │
│ careers              │ 5     │ 0/5 !!   │ 0/5 !!   │ 0/5 !!   │
│ enroll               │ 5     │ 3/5 ✓    │ 0/5 !!   │ 0/5 !!   │
│ student              │ 3     │ 3/3 OK   │ 0/3 !!   │ 0/3 !!   │
├──────────────────────┼───────┼──────────┼──────────┼──────────┤
│ TOTAL                │ 43    │ 26% !!   │ 0% !!    │ 0% !!    │
└──────────────────────┴───────┴──────────┴──────────┴──────────┘
```

*Output from Metro-Grama — an Echo/SurrealDB application.*

### Filter by domain

```bash
testreg status --domain auth
```

### Filter by priority

```bash
testreg status --priority critical
```

## Tips

- `status` shows registry-level coverage (annotation-based), not graph-level health scores
- For graph-aware health scores, use `testreg audit` instead
- Use `--format json` for CI integration
