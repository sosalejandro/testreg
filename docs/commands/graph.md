# testreg graph

Export a feature's dependency graph in visualization formats. Produces the same graph as `trace` but outputs it in Graphviz DOT, Mermaid, or JSON format for embedding in documentation, PRs, or design tools.

## Usage

```
testreg graph <feature-id> [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `dot` | Output format: `dot`, `mermaid`, `json` |
| `--output` | stdout | Write to file instead of stdout |

## Examples

### Graphviz DOT

```
$ testreg graph auth.login --format dot

digraph trace {
  rankdir=TB;
  node [fontname="Helvetica" fontsize=10];
  edge [fontname="Helvetica" fontsize=9];

  "AuthHandler.Login" [label="AuthHandler.Login\nsrc/infrastructure/http/handlers/auth_handler.go:249" shape=box];
  "authService.Login" [label="authService.Login\nsrc/application/services/auth_service.go:172" shape=ellipse];
  "authRepository.StoreRefreshToken" [label="authRepository.StoreRefreshToken\nsrc/domain/repositories/auth_repository.go:329" shape=cylinder];
  "sql:GetUserByEmail" [label="sql:GetUserByEmail\nsrc/domain/repositories/queries/user.sql:21" shape=note];

  "AuthHandler.Login" -> "authService.Login";
  "authService.Login" -> "authRepository.StoreRefreshToken";
  "authService.Login" -> "sql:GetUserByEmail";
}
```

### Render to SVG

```bash
testreg graph auth.login --format dot | dot -Tsvg -o auth-login.svg
```

### Mermaid (for GitHub PRs and docs)

```
$ testreg graph auth.login --format mermaid

flowchart TD
  AuthHandler_Login["AuthHandler.Login"]
  authService_Login("authService.Login")
  authRepository_StoreRefreshToken[("authRepository.StoreRefreshToken")]
  sql_GetUserByEmail>"sql:GetUserByEmail"]

  AuthHandler_Login --> authService_Login
  authService_Login --> authRepository_StoreRefreshToken
  authService_Login --> sql_GetUserByEmail

  classDef handler fill:#e0f7fa,stroke:#00acc1
  classDef service fill:#e8f5e9,stroke:#43a047
  classDef repository fill:#fff8e1,stroke:#f9a825
  classDef query fill:#f3e5f5,stroke:#8e24aa
```

### Write to file

```bash
testreg graph auth.login --format mermaid --output docs/auth-login-graph.md
```

## Tips

- Use Mermaid output in GitHub PR descriptions -- GitHub renders it natively
- DOT output can be piped to `dot`, `neato`, or `sfdp` for different layout algorithms
- JSON output is useful for building custom visualizations or feeding into other tools
