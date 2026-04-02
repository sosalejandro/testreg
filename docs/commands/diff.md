# testreg diff

Compare feature health snapshots across sprints. Save baselines, track progress, identify regressions.

## Usage

```
testreg diff [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--save-snapshot` | (none) | Save current audit as a named snapshot |
| `--baseline` | (none) | Path to baseline JSON file to compare against |
| `--from` | (none) | Named snapshot to compare from |
| `--to` | (none) | Named snapshot to compare to (default: current) |
| `--format` | `terminal` | Output format: `terminal`, `json` |

Snapshots are stored in `.testreg-cache/snapshots/`.

## Examples

### Save a baseline

```bash
testreg diff --save-snapshot sprint-3
```

### Compare against latest

```
$ testreg diff

Health Changes (since sprint-1):

  Improved (12 features):
    +100%  auth.login                      0% -> 100%
    + 74%  client-management.detail        0% ->  74%
    + 50%  recipes.create                 30% ->  80%

  Regressed (2 features):
    - 10%  meals.log.create              80% ->  70%

  Unchanged: 170 features

  Summary: +15.4% average health change
```

### Compare two snapshots

```bash
testreg diff --from sprint-2 --to sprint-3
```

### Sprint tracking workflow

```bash
# Sprint start
testreg diff --save-snapshot sprint-3-start

# Work on tests...

# Sprint end
testreg diff --from sprint-3-start
```

## Tips

- `--save-snapshot` also writes `latest.json` for convenient `testreg diff` with no arguments.
- Use JSON output in CI for automated progress dashboards.
- Save snapshots on main branch merges for continuous tracking.
