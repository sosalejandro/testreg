# testreg sprint

Rank features by priority-weighted gap score for sprint planning.

## Score formula

```
score = weight * max(0, target - health)
```

| Priority | Weight | Target |
|----------|--------|--------|
| critical | 4 | 100% |
| high | 3 | 80% |
| medium | 2 | 60% |
| low | 1 | 40% |

Features at or above target are excluded.

## Usage

```
testreg sprint [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `terminal` | Output format: `terminal`, `json` |
| `-n, --limit` | `20` | Top N results |
| `--priority` | (all) | Filter by priority (comma-separated) |
| `--group-by` | (none) | Group output by: `type`, `domain` |

## Examples

### Default sprint output

```
$ testreg sprint

Sprint Priorities (20 features, sorted by priority score):

  Score  Priority   Health  Target  Feature
  ──────────────────────────────────────────────────────────────────────
   4.00  critical      0%    100%  auth.login
   4.00  critical      0%    100%  auth.register
   3.00  critical     25%    100%  auth.token-refresh
   2.40  high         20%     80%  recipes.create

  By Fix Type:
    unit tests:          34 features
    integration tests:   18 features
    e2e tests:            6 features
    benchmarks:          12 features
    race tests:           8 features
```

### Group by domain

```
$ testreg sprint --group-by domain -n 10

  auth (3 features, score: 11.00):
    4.00  critical   0%  auth.login
    4.00  critical   0%  auth.register
    3.00  critical  25%  auth.token-refresh

  recipes (2 features, score: 4.80):
    2.40  high      20%  recipes.create
    2.40  high      20%  recipes.search
```

### Critical and high only

```bash
testreg sprint --priority critical,high -n 10
```

## Tips

- Run at the start of each sprint to decide what to fix.
- Use `--group-by domain` to assign test-writing work by team or domain owner.
- Combine with `testreg gaps --format prompt` to hand off specific fixes to AI agents.
