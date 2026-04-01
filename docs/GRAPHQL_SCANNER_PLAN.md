# GraphQL Dependency Chain Scanner — Implementation Plan

**Date:** 2026-03-31
**Status:** Design, ready to implement
**Problem:** `testreg trace training.record-exercise` returns empty because testreg can't resolve `GRAPHQL Mutation.trainingLogSet` to a Go function entry point.

---

## How It Works Today (REST)

```
Registry: method: POST, path: /api/v1/auth/login
    ↓ route parser finds "POST /api/v1/auth/login" in router file
    ↓ maps to handler function ID: "AuthHandler.Login"
    ↓ Go AST scanner traces from that entry point
    ↓ produces dependency chain
```

## How It Should Work (GraphQL)

```
Registry: method: GRAPHQL, path: Mutation.trainingLogSet
    ↓ GraphQL resolver scanner finds method TrainingLogSet on mutationResolver
    ↓ maps to resolver function ID: "mutationResolver.TrainingLogSet"
    ↓ Go AST scanner traces from that entry point
    ↓ produces dependency chain through delegation layers
```

The key insight: **GraphQL resolvers are just Go functions.** Once we find the entry point function, the existing AST call tracer handles everything downstream. We only need a new **entry point discovery** mechanism.

---

## The Call Chain to Trace

```
Mutation.trainingLogSet                    (schema definition)
  → mutationResolver.TrainingLogSet        (gateway resolver - generated+edited)
    → r.Training.LogSet                    (delegates to bounded context)
      → TrainingResolver.LogSet            (internal resolver)
        → sessionService.LogSet            (service layer)
          → setRepo.Create                 (repository)
            → sql:CreateExerciseSet        (SQL query)
```

The Go AST scanner already traces steps 2-6 via struct field resolution. Step 1 (schema → Go method) is the only new logic needed.

---

## Implementation

### File: `internal/adapters/graphql_resolver_scanner.go` (new)

```go
// GraphQLResolverScanner discovers GraphQL resolver methods from
// *.resolvers.go files and maps GraphQL field names to Go function IDs.
type GraphQLResolverScanner struct{}

// GraphQLMapping maps a GraphQL operation to its resolver function.
type GraphQLMapping struct {
    Operation    string // "Mutation" or "Query" or "Subscription"
    FieldName    string // "trainingLogSet"
    GoMethod     string // "TrainingLogSet"
    ReceiverType string // "mutationResolver"
    NodeID       string // "mutationResolver.TrainingLogSet"
    File         string
    Line         int
}
```

### Resolution Logic

**Step 1: Parse the registry entry**
```
"Mutation.trainingLogSet" → operation="Mutation", field="trainingLogSet"
```

**Step 2: Derive expected Go method name**
```go
func graphqlFieldToGoMethod(field string) string {
    // "trainingLogSet" → "TrainingLogSet"
    // First letter uppercased (Go export convention)
    return strings.ToUpper(field[:1]) + field[1:]
}
```

**Step 3: Derive expected receiver type**
```go
func graphqlOperationToReceiver(op string) string {
    // "Mutation" → "mutationResolver"
    // "Query" → "queryResolver"
    // "Subscription" → "subscriptionResolver"
    return strings.ToLower(op[:1]) + op[1:] + "Resolver"
}
```

**Step 4: Find the method in *.resolvers.go files**

Scan all `*.resolvers.go` files in the backend root using `go/ast`. For each file:
1. Find method declarations matching: `func (r *mutationResolver) TrainingLogSet(...)`
2. Extract file path and line number
3. Build the node ID: `"mutationResolver.TrainingLogSet"`

This uses the same AST parsing as the existing Go scanner — just targeting a specific method signature.

**Step 5: Return the node ID as an entry point**

The entry point `"mutationResolver.TrainingLogSet"` is passed to `Graph.TraceFrom()`, which follows the call chain through struct field resolution (already working).

### Integration Point: `internal/adapters/go_ast_scanner.go`

In `deriveEntryPoints()` (currently in `internal/app/trace_feature.go`), add handling for GRAPHQL method:

```go
func deriveEntryPoints(f *domain.Feature) []string {
    for _, api := range f.Surfaces.API {
        if api.Method == "GRAPHQL" {
            // "Mutation.trainingLogSet" → "mutationResolver.TrainingLogSet"
            parts := strings.SplitN(api.Path, ".", 2)
            if len(parts) == 2 {
                receiver := strings.ToLower(parts[0][:1]) + parts[0][1:] + "Resolver"
                method := strings.ToUpper(parts[1][:1]) + parts[1][1:]
                points = append(points, receiver+"."+method)
            }
        } else if httpMethods[api.Method] {
            // existing REST handling
            points = append(points, api.Method+" "+api.Path)
        }
    }
}
```

That's it. **3 lines of logic** in `deriveEntryPoints` + the naming convention. The existing AST scanner discovers the function during Phase 2 (function discovery), and `TraceFrom` follows the calls.

### Why This Works Without a Separate Scanner

The Go AST scanner's Phase 2 already walks ALL `.go` files under `backend_root`, including `*.resolvers.go` files. It already creates nodes for every exported method:
- `mutationResolver.TrainingLogSet` → discovered in Phase 2
- `TrainingResolver.LogSet` → discovered in Phase 2
- `SessionLifecycleService.LogSet` → discovered in Phase 2

The only missing piece is **entry point derivation** — converting `Mutation.trainingLogSet` to a node ID the graph already contains. Once that mapping exists, `TraceFrom` follows the entire chain automatically.

---

## Changes Required

### File 1: `internal/app/trace_feature.go` — modify `deriveEntryPoints()`

Add GRAPHQL handling alongside the existing HTTP method handling:

```go
func deriveEntryPoints(f *domain.Feature) []string {
    httpMethods := map[string]bool{
        "GET": true, "POST": true, "PUT": true, "PATCH": true,
        "DELETE": true, "HEAD": true, "OPTIONS": true,
    }

    var points []string
    for _, api := range f.Surfaces.API {
        if httpMethods[api.Method] {
            id := fmt.Sprintf("%s %s", api.Method, api.Path)
            points = append(points, id)
        } else if api.Method == "GRAPHQL" {
            // Mutation.trainingLogSet → mutationResolver.TrainingLogSet
            id := graphqlEntryPoint(api.Path)
            if id != "" {
                points = append(points, id)
            }
        } else {
            // gRPC/Consumer/Event: path is the function node ID directly
            points = append(points, api.Path)
        }
    }
    return points
}

// graphqlEntryPoint converts a GraphQL operation path to a Go resolver node ID.
// "Mutation.trainingLogSet" → "mutationResolver.TrainingLogSet"
// "Query.trainingSessions" → "queryResolver.TrainingSessions"
func graphqlEntryPoint(path string) string {
    parts := strings.SplitN(path, ".", 2)
    if len(parts) != 2 || parts[1] == "" {
        return ""
    }
    
    operation := parts[0]  // "Mutation"
    field := parts[1]      // "trainingLogSet"
    
    // Go receiver: mutationResolver, queryResolver, subscriptionResolver
    receiver := strings.ToLower(operation[:1]) + operation[1:] + "Resolver"
    
    // Go method: PascalCase of the field name
    method := strings.ToUpper(field[:1]) + field[1:]
    
    return receiver + "." + method
}
```

### File 2: `internal/adapters/go_ast_scanner.go` — ensure resolver files are scanned

Check that `*.resolvers.go` files are NOT filtered out during function discovery. Currently, the scanner skips `_test.go` and `generated/` directories. Resolver files should pass through:
- `training.resolvers.go` → not a test file ✓
- `resolver.go` → not a test file ✓
- Files under `generated/` are skipped — but resolver files live in `resolvers/`, not `generated/` ✓

**No changes needed** — the scanner already discovers resolver methods.

### File 3: `internal/adapters/go_ast_scanner.go` — node kind classification

Add a `graphql-resolver` or `handler` classification for resolver files:

```go
func classifyNodeKind(pkgDir string) domain.NodeKind {
    lower := strings.ToLower(pkgDir)
    switch {
    case strings.Contains(lower, "handler") || strings.Contains(lower, "resolver"):
        return domain.NodeHandler
    // ... existing cases
    }
}
```

The `resolver` directory name should classify functions as handlers (they serve the same architectural role as HTTP handlers).

### File 4: Tests

`internal/app/trace_feature_test.go`:
- `TestGraphqlEntryPoint` — verify naming convention mapping
- `TestDeriveEntryPoints_GRAPHQL` — verify GRAPHQL method produces correct entry point

`internal/adapters/go_ast_scanner_test.go`:
- Test that resolver files are discovered and their methods are in the graph

---

## Config Changes

### `.testreg.yaml` — optional `graphql` section

```yaml
graph:
  backend_root: src
  # Optional: specify resolver directories for better discovery
  graphql:
    schema_dirs:
      - src/training/pkg/schema
      - src/supplement/pkg/schema
    resolver_dir: src/cmd/graphql/resolvers
```

This is optional — the scanner discovers resolver methods through normal AST walking. The config could improve performance by narrowing the search, but it's not required for correctness.

---

## Validation

```bash
# Before: empty trace
testreg trace training.record-exercise --project-root /path/to/nutrition-v2

# After: full dependency chain
testreg trace training.record-exercise --project-root /path/to/nutrition-v2
#   mutationResolver.TrainingLogSet     resolvers/training.resolvers.go:60
#   └─ TrainingResolver.LogSet          resolver.go:289
#      └─ SessionLifecycleService.LogSet session_lifecycle_service.go:141
#         ├─ setRepo.Create             set_repository.go:30
#         └─ ...
```

---

## Why No Separate Scanner Is Needed

| Concern | Answer |
|---------|--------|
| Finding resolver methods | Already discovered in Phase 2 (walks all .go files) |
| Following struct field calls | Already works (r.Training.LogSet resolved via structFields) |
| Crossing package boundaries | Already works (function discovery spans all packages under backend_root) |
| Wire/DI resolution | Already works for resolver struct field types |
| Node kind classification | Just add "resolver" to the handler classification check |
| Entry point derivation | 15 lines of naming convention code in deriveEntryPoints |

The entire feature is a **naming convention mapper** — 15 lines of code to convert `Mutation.trainingLogSet` to `mutationResolver.TrainingLogSet`. Everything else is already built.

---

## Estimated Scope

| Item | Lines | Files |
|------|-------|-------|
| `graphqlEntryPoint()` function | ~15 | `trace_feature.go` |
| Add "resolver" to classifyNodeKind | ~1 | `go_ast_scanner.go` |
| Tests | ~40 | `trace_feature_test.go` |
| README update | ~10 | `README.md` |
| **Total** | **~66 lines** | **4 files** |
