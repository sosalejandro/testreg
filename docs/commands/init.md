# testreg init

Bootstrap the registry directory with template YAML domain files. Idempotent: running it again merges new features without overwriting existing entries.

## Usage

```
testreg init [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--discover` | false | Auto-discover features from router file and project structure |

## Examples

### Basic init (templates)

```bash
$ testreg init

Created docs/testing/registry/
  auth.yaml (template)
  users.yaml (template)
```

### Auto-discover from routes

```
$ testreg init --discover

Discovered 43 routes -> 43 features across 15 domains
  auth: 5 features
  careers: 5 features
  enroll: 5 features
  subjects: 4 features
  student: 3 features
  professor: 4 features
  admin: 4 features
  grades: 3 features
  schedule: 4 features
  attendance: 3 features
  notifications: 2 features
  reports: 2 features
  health: 1 feature
  settings: 2 features
  feedback: 1 feature
```

*Output from Metro-Grama -- an Echo/SurrealDB application. testreg parsed the Echo router file to discover all HTTP endpoints, grouped them by URL path prefix into domains, and generated registry YAML files with real API surfaces.*

### What --discover generates

For each discovered route, testreg creates a feature entry with:
- Feature ID derived from the URL path (e.g., `POST /api/v1/auth/login` -> `auth.login`)
- HTTP method and path as API surface
- Priority set to `medium` (you adjust manually)
- Empty coverage entries ready for `testreg scan` to populate

## Tips

- Use `--discover` on existing projects for instant onboarding -- no annotations needed for the initial setup
- Run `testreg scan` immediately after `init --discover` to map existing tests
- `init` is idempotent -- safe to run again after adding new routes
- Supports Chi, Echo, and stdlib routers. For Gin/Fiber, use basic `init` and add features manually
