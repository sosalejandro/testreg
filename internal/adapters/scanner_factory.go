package adapters

import "github.com/sosalejandro/testreg/internal/ports"

// NewGraphBuilder selects the appropriate GraphBuilder based on config.
// When TypeChecking is enabled, it returns a TypedScanner that uses
// golang.org/x/tools/go/packages for full type resolution.
// Otherwise it returns the default GoASTScanner.
func NewGraphBuilder(config ports.GraphConfig) ports.GraphBuilder {
	if config.TypeChecking {
		return NewTypedScanner()
	}
	return NewGoASTScanner()
}
