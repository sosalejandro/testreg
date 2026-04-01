# go/types Integration Plan — Optional Full Type Resolution

**Date:** 2026-03-31
**Status:** Designed, not implemented
**Trigger:** `graph.type_checking: true` in `.testreg.yaml` (default: false)

---

## What It Solves

The current `go/ast` scanner resolves ~95% of call edges via struct field maps and naming heuristics. The remaining 5% are:
- Cross-package struct field delegation (`r.Training.LogSet()` where Training is a qualified type from another package)
- Interface implementations not covered by Wire/Fx resolvers
- Generic types
- Embedded struct method sets
- Type assertions in switch statements

With `go/types`, these resolve exactly — no heuristics, no ambiguity flags.

---

## Architecture

Both scanners implement the same `ports.GraphBuilder` interface. Config selects which one runs:

```
                    ports.GraphBuilder
                    /              \
        GoASTScanner            TypedScanner
        (go/ast only)           (go/ast + go/types)
        default, fast           opt-in, exact
        any source dir          needs buildable project
        0.7s / 153 MB           3-8s / 300-500 MB
```

```yaml
# .testreg.yaml
graph:
  type_checking: true   # opt-in
```

If `type_checking: true` and the project can't be loaded (missing deps, broken build), fall back to `GoASTScanner` with a warning. Never fail silently, never block the user.

---

## Implementation

### New dependency

```
go get golang.org/x/tools/go/packages
```

Adds ~6 MB to binary. ~20 transitive deps (all stdlib-adjacent from x/tools).

### New file: `internal/adapters/typed_scanner.go` (~200 lines)

```go
type TypedScanner struct {
    fallback *GoASTScanner // used when type checking fails
}

func (s *TypedScanner) Build(projectRoot string, config ports.GraphConfig) (*domain.Graph, error) {
    cfg := &packages.Config{
        Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
              packages.NeedName | packages.NeedFiles | packages.NeedImports,
        Dir:  projectRoot,
    }
    
    pkgs, err := packages.Load(cfg, "./"+config.BackendRoot+"/...")
    if err != nil {
        fmt.Fprintf(os.Stderr, "warning: type checking failed, falling back to AST: %v\n", err)
        return s.fallback.Build(projectRoot, config)
    }
    
    graph := domain.NewGraph()
    
    for _, pkg := range pkgs {
        // Phase 1: Create nodes from function declarations
        for _, file := range pkg.Syntax {
            for _, decl := range file.Decls {
                fn, ok := decl.(*ast.FuncDecl)
                if !ok { continue }
                // Node creation (same as GoASTScanner but with exact type info)
                node := buildNodeFromFunc(pkg, fn)
                graph.AddNode(node)
            }
        }
        
        // Phase 2: Resolve calls using TypesInfo
        for _, file := range pkg.Syntax {
            ast.Inspect(file, func(n ast.Node) bool {
                call, ok := n.(*ast.CallExpr)
                if !ok { return true }
                
                // TypesInfo.Uses resolves the callee exactly
                callee := resolveCalleeViaTypes(pkg.TypesInfo, call)
                if callee != "" {
                    graph.AddEdge(currentFunc, callee)
                }
                return true
            })
        }
    }
    
    return graph, nil
}

func resolveCalleeViaTypes(info *types.Info, call *ast.CallExpr) string {
    // For selector expressions like r.Training.LogSet():
    // info.Selections[sel] gives the exact method being called,
    // including the receiver type resolved through all indirections
    
    // For package-level calls like fmt.Println():
    // info.Uses[ident] gives the exact object being referenced
}
```

### New file: `internal/adapters/typed_scanner_test.go` (~100 lines)

Test that typed scanner produces the same graph as AST scanner for simple cases, plus verify it resolves cross-package calls that AST scanner marks as ambiguous.

### Modified: `internal/adapters/graph_config.go`

```go
type GraphSection struct {
    // ... existing fields
    TypeChecking bool `yaml:"type_checking"` // opt-in go/types resolution
}
```

### Modified: `internal/ports/graph.go`

```go
type GraphConfig struct {
    // ... existing fields
    TypeChecking bool
}
```

### Modified: `cmd/audit.go` (and other commands that create the scanner)

```go
var builder ports.GraphBuilder
if config.TypeChecking {
    builder = adapters.NewTypedScanner()
} else {
    builder = adapters.NewGoASTScanner()
}
```

Or better — put the selection in a factory:

```go
// internal/adapters/scanner_factory.go
func NewGraphBuilder(config ports.GraphConfig) ports.GraphBuilder {
    if config.TypeChecking {
        return NewTypedScanner()
    }
    return NewGoASTScanner()
}
```

---

## Performance Impact

| Metric | go/ast (default) | go/types (opt-in) |
|--------|------------------|-------------------|
| Binary size | 6.5 MB | ~12-15 MB |
| Scan time (184 features) | 10s | 15-25s |
| Single trace | 0.7s | 3-8s |
| Peak memory | 153 MB | 300-500 MB |
| Dependencies | 2 direct | 3 direct + ~20 transitive |
| Requires buildable project | No | Yes |
| Edge accuracy | ~95% | ~100% |

Memory and CPU are only consumed when `type_checking: true`. Default behavior is unchanged.

---

## What NOT to change

- `GoASTScanner` stays exactly as-is — it's the default, battle-tested path
- Wire/Fx/Dig resolvers remain useful even with go/types (they provide explicit DI mappings that complement type resolution)
- The `@calls` annotation escape hatch remains for edges neither scanner can follow
- All existing tests continue to use `GoASTScanner`

---

## Verification

```bash
# Without type checking (default, unchanged behavior)
testreg trace auth.login --project-root /path/to/project

# With type checking (opt-in, deeper resolution)
# Requires: go mod download in the target project
testreg trace training.record-exercise --project-root /path/to/project
# Should show: full chain through cross-package delegation

# Verify fallback works
# Break the target project's go.mod, verify warning + AST fallback
```
