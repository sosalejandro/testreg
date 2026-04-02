# testreg diagnose

Match an error symptom against built-in failure patterns, then trace the feature's dependency graph to identify which files to check first. Useful for rapid triage when a test or production error occurs.

## Usage

```
testreg diagnose <feature-id> [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--symptom` | (required) | Error symptom text to diagnose |
| `--json` | false | Output as JSON |

## Examples

### Basic diagnosis

```
$ testreg diagnose auth.login --symptom "401 unauthorized"

  Diagnosis Report
  Feature:  auth.login
  Symptom:  401 unauthorized

  Best Match
  Layer:        backend-auth
  Confidence:   70%
  Description:  Authentication failure: request lacks valid credentials or session has expired
  Check order:  handler -> service -> external

  Dependency Trace
  (trace tree output here)

  Files to check (ordered by likelihood):
    1. src/infrastructure/http/handlers/auth_handler.go
    2. src/application/services/auth_service.go
    3. src/infrastructure/auth/jwt_generator.go
```

### Multi-match example

When an error matches multiple patterns, diagnose shows the best match and also lists secondary matches:

```
$ testreg diagnose auth.login --symptom "500 internal server error: context deadline exceeded"

  Diagnosis Report
  Feature:  auth.login
  Symptom:  500 internal server error: context deadline exceeded

  Best Match
  Layer:        infra
  Confidence:   60%
  Description:  Operation exceeded time limit: network, database, or service latency
  Check order:  external -> repository -> service

  Also Matched
    50% backend-bug — Server-side crash or unhandled error in business logic
```

### Database constraint errors (high confidence)

```
$ testreg diagnose users.create --symptom "unique constraint violation on users.email"

  Best Match
  Layer:        data
  Confidence:   95%
  Description:  Duplicate record: insert or update violates a uniqueness constraint
  Check order:  repository -> query -> service
```

### JSON output

```bash
testreg diagnose auth.login --symptom "timeout exceeded" --json
```

Returns structured JSON with `best_match`, `all_matches` (each with confidence), and `check_files`.

## Built-in Symptom Patterns

### High Confidence (85-95%)
| Pattern | Layer | Check Order |
|---------|-------|-------------|
| `unique constraint / duplicate key` | data | repository -> query -> service |
| `foreign key violation` | data | repository -> query -> service |
| `deadlock detected` | data | repository -> service |
| `sql: no rows` | data | repository -> query -> service |
| `json: cannot unmarshal` | backend-bug | handler -> service -> external |
| `login failed / invalid credentials` | backend-auth | service -> handler -> external |
| `connection refused / ECONNREFUSED` | infra | external -> repository |
| `CORS / origin not allowed` | infra | handler -> external |
| `selector not found / getBy failed` | frontend | component -> hook |
| `TypeError / Cannot read property` | frontend | component -> hook -> service |
| `hydration mismatch` | frontend | component -> hook |
| `TLS / certificate error` | infra | external |

### Medium Confidence (70-85%)
| Pattern | Layer | Check Order |
|---------|-------|-------------|
| `no route / route not found` | backend-routing | endpoint -> handler |
| `409 / conflict` | data | repository -> service -> handler |
| `422 / validation failed` | backend-bug | handler -> service |
| `429 / rate limit` | infra | handler -> external |
| `502 / bad gateway` | infra | external -> handler |
| `503 / service unavailable` | infra | external -> service |
| `context canceled` | infra | service -> external |
| `EOF / broken pipe` | infra | external -> repository -> service |
| `401 / unauthorized` | backend-auth | handler -> service -> external |
| `403 / forbidden` | backend-auth | handler -> service |

### Lower Confidence (50-60%)
| Pattern | Layer | Check Order |
|---------|-------|-------------|
| `404 / not found` | backend-routing | endpoint -> handler -> repository |
| `timeout / deadline exceeded` | infra | external -> repository -> service |
| `500 / internal server error` | backend-bug | service -> repository -> handler |
| `empty response / no data` | data | repository -> query -> service |

## How It Works

1. Regex-matches the symptom text against all built-in rules
2. Returns all matches ranked by confidence (highest first)
3. Traces the feature's dependency graph
4. Walks the trace tree and groups files by node kind (handler, service, repository, etc.)
5. Orders files according to the best match's check order -- files matching the first kind appear first

## Tips

- Higher confidence rules (DB constraints, serialization errors) are almost always right about the layer
- Lower confidence rules (generic 500, 404) are directional hints -- the error could originate in multiple layers
- When multiple rules match, look at all of them -- a "500: context deadline exceeded" is both a server error AND a timeout
- Use `--json` for programmatic consumption in CI or automated triage workflows
