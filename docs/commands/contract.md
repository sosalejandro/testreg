# testreg contract

Show the full API contract and implementation chain for a feature. Traces from the API entry point (REST or GraphQL) through each architectural layer, showing function signatures, data types, and test coverage at each level.

## Usage

```
testreg contract <feature-id> [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` | `terminal` | Output format: `terminal`, `json`, `markdown` |
| `--layer` | `0` (all) | Show only up to this layer depth |

## Examples

### Terminal output

```
$ testreg contract auth.login

  Layer 5: Handler
    File: src/infrastructure/http/handlers/auth_handler.go:249
    func (*AuthHandler) Login(w http.ResponseWriter, r *http.Request)
    Delegates to: authService.Login

  Layer 6: Service
    File: src/application/services/auth_service.go:172
    func (*authService) Login(ctx context.Context, email string,
                              password string) (*AuthResponse, error)
    Also calls: JWTGenerator.GenerateTokenPair,
                authRepository.StoreRefreshToken, sql:GetUserByEmail

  Layer 7: Repository
    File: src/domain/repositories/auth_repository.go:329
    func (*authRepository) StoreRefreshToken(ctx context.Context,
                                              token *RefreshToken) error

  Layer 8: Query
    File: src/domain/repositories/queries/user.sql:21
    SQL: SELECT id, email, password_hash, role FROM users WHERE email = $1
```

*Output from nutrition-project-v2.*

### Limit to first 2 layers

```
$ testreg contract auth.login --layer 2
```

Shows only the handler and service layers.

### Markdown output (for PRs/docs)

```bash
testreg contract auth.login --format markdown > docs/api/auth-login-contract.md
```

### With type_checking enabled (experimental)

> **Warning:** `type_checking: true` is experimental and buggy. It does not yet integrate the route parser, Wire/Fx resolver, or SQLC mapper — producing fewer traced nodes than the default scanner. It also uses significantly more memory (~4 GB vs ~150 MB for large workspaces). **We do not recommend enabling this feature yet.** The default `go/ast` scanner is the recommended path for all commands.

When `type_checking: true` is set in `.testreg.yaml`, the contract includes struct field tables at each layer showing exact input/output types with required/optional markers. However, due to the limitations above, the output may be incomplete compared to the default scanner.

### GraphQL feature

```
$ testreg contract training.record-exercise

  Layer 1: GraphQL Mutation
    Schema: src/training/pkg/schema/training.graphqls
    mutation { trainingLogSet(input: TrainingLogSetInput!): TrainingLogSetPayload! }

  Layer 2: Resolver
    File: src/cmd/graphql/resolvers/training.resolvers.go:60
    func (*mutationResolver) TrainingLogSet(ctx context.Context,
                                             input model.TrainingLogSetInput) (*model.TrainingLogSetPayload, error)
    Delegates to: sessionService.LogSet

  Layer 3: Service
    ...
```

## Tips

- Use `--format markdown` to auto-generate API documentation from source code
- Combine with `testreg trace` to get both the visual tree and the detailed contracts
- The `--layer` flag is useful when you only care about the API surface (layer 1-2) and not the internals
- GraphQL support requires `graphql.schema_dirs` in `.testreg.yaml`
