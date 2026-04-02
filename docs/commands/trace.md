# testreg trace

Trace a feature's full-stack dependency graph from API entry points through handlers, services, repositories, and SQL queries. Includes frontend routes and hooks when TypeScript scanning is configured.

## Usage

```
testreg trace <feature-id> [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `tree` | Output format: `tree`, `json` |
| `--depth` | from config (10) | Maximum traversal depth |
| `--verbose` | false | Include utility functions in trace |
| `--list-nodes` | false | Output flat list of all node IDs |
| `--kind` | (all) | Filter nodes by kind: `handler`, `service`, `repository`, `query`, `component`, `hook` |
| `--validate` | false | Check for duplicates, cycles, missing refs |

## Examples

### Basic trace (tree output)

```
$ testreg trace auth.login

  Feature: User Login (auth.login)
  Priority: critical
  API Surfaces:
    POST /api/v1/auth/login

route:/login                                              apps/web/src/router.tsx:142
└─ LoginPage                                              apps/web/src/pages/auth/LoginPage.tsx:13
   └─ useAuth                                             apps/web/src/hooks/useAuth.ts:19
      └─ authApi.login                                    apps/web/src/services/api/auth.ts:46
         └─ POST /api/v1/auth/login                       src/infrastructure/http/handlers/auth_handler.go:576
            └─ AuthHandler.Login                          src/infrastructure/http/handlers/auth_handler.go:249
               └─ authService.Login                       src/application/services/auth_service.go:172
                  ├─ JWTGenerator.GenerateTokenPair        src/infrastructure/auth/jwt_generator.go:70
                  │  ├─ JWTGenerator.GenerateAccessToken   src/infrastructure/auth/jwt_generator.go:97
                  │  └─ JWTGenerator.GenerateRefreshToken  src/infrastructure/auth/jwt_generator.go:123
                  ├─ authRepository.StoreRefreshToken      src/domain/repositories/auth_repository.go:329
                  ├─ repositories.HashToken                src/domain/repositories/auth_repository.go:90
                  └─ sql:GetUserByEmail                    src/domain/repositories/queries/user.sql:21

  Confidence: 100%  |  Nodes: 11  |  Depth: 7
```

*Output from nutrition-project-v2 -- a full-stack Go + React monorepo with 184 features.*

### List nodes (for scripting)

```
$ testreg trace auth.login --list-nodes

route:/login
LoginPage
useAuth
authApi.login
POST /api/v1/auth/login
AuthHandler.Login
authService.Login
JWTGenerator.GenerateTokenPair
JWTGenerator.GenerateAccessToken
JWTGenerator.GenerateRefreshToken
authRepository.StoreRefreshToken
repositories.HashToken
sql:GetUserByEmail
```

### Filter by node kind

```
$ testreg trace auth.login --list-nodes --kind service

authService.Login
JWTGenerator.GenerateTokenPair
JWTGenerator.GenerateAccessToken
JWTGenerator.GenerateRefreshToken
```

### JSON output

```
$ testreg trace auth.login --format json
```

Returns the full trace tree as structured JSON with node IDs, kinds, file paths, line numbers, and children.

### Validate trace integrity

```
$ testreg trace auth.login --validate

  Validation Results:
    Duplicate nodes: 0
    Cycles detected: 0
    Missing references: 0
    Status: PASS
```

### Limit depth

```
$ testreg trace auth.login --depth 3

route:/login                                              apps/web/src/router.tsx:142
└─ LoginPage                                              apps/web/src/pages/auth/LoginPage.tsx:13
   └─ useAuth                                             apps/web/src/hooks/useAuth.ts:19
      └─ authApi.login                                    apps/web/src/services/api/auth.ts:46

  Confidence: 100%  |  Nodes: 4  |  Depth: 3
```

## How It Works

1. Looks up the feature in the registry YAML to find its API surfaces (e.g., `POST /api/v1/auth/login`)
2. Finds the handler node matching that endpoint (via router parsing or `@api` annotations)
3. Walks the call graph following struct field method calls, Wire/Fx DI bindings, and SQLC mappings
4. If TypeScript scanning is configured, traces from the frontend route through components and hooks to the API call
5. Renders the tree with file:line references and color-coded node kinds

## Tips

- Use `--list-nodes --kind service` to get a list of all service functions involved in a feature -- useful for targeted test writing
- Pipe `--list-nodes` output to scripts for automation (e.g., checking which nodes have test files)
- Low confidence means missing edges -- check the warnings for which files failed to parse
- The trace is a DFS on a cached adjacency list and takes milliseconds; the graph building (~0.7s) is the slow part
