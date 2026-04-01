package adapters

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GraphQLSchema holds parsed type and operation definitions from .graphqls files.
type GraphQLSchema struct {
	Types     map[string]*GraphQLType  // "TrainingLogSetInput" -> fields
	Mutations map[string]*GraphQLField // "trainingLogSet" -> field def
	Queries   map[string]*GraphQLField // "trainingSessions" -> field def
}

// GraphQLType represents a named type or input definition.
type GraphQLType struct {
	Name   string
	Fields []GraphQLField
}

// GraphQLField represents a single field in a type or operation.
type GraphQLField struct {
	Name       string
	Type       string // "UUID!", "[String]", "Int"
	Required   bool
	Args       []GraphQLField // for mutations/queries with arguments
	ReturnType string         // resolved return type for operations
}

// GraphQLSchemaParser parses .graphqls files using regex.
type GraphQLSchemaParser struct{}

// NewGraphQLSchemaParser creates a new parser instance.
func NewGraphQLSchemaParser() *GraphQLSchemaParser {
	return &GraphQLSchemaParser{}
}

// Regex patterns for parsing GraphQL schema files.
var (
	// Matches type/input block start: "type Foo {" or "input Foo {"
	reTypeStart = regexp.MustCompile(`^\s*(type|input)\s+(\w+)\s*\{`)

	// Matches extend type Mutation/Query: "extend type Mutation {"
	reExtendType = regexp.MustCompile(`^\s*extend\s+type\s+(Mutation|Query)\s*\{`)

	// Matches a field definition: "fieldName: TypeName!" or "fieldName(args...): ReturnType!"
	reField = regexp.MustCompile(`^\s+(\w+)\s*(?:\(([^)]*)\))?\s*:\s*(.+?)\s*$`)

	// Matches argument in an argument list: "argName: ArgType"
	reArg = regexp.MustCompile(`(\w+)\s*:\s*([^\s,]+)`)

	// Matches block close.
	reBlockClose = regexp.MustCompile(`^\s*\}`)

	// Matches comment or directive lines.
	reComment = regexp.MustCompile(`^\s*(#|"""|\.\.\.)`)

	// Matches triple-quote doc strings.
	reTripleQuote = regexp.MustCompile(`^\s*"""`)

	// Matches "scalar Foo" or "enum Foo" — lines to skip.
	reScalarOrEnum = regexp.MustCompile(`^\s*(scalar|enum|union|interface)\s+\w+`)
)

// ParseDir reads all .graphqls files in a directory (recursively) and returns
// a merged schema containing all types, mutations, and queries.
func (p *GraphQLSchemaParser) ParseDir(dir string) (*GraphQLSchema, error) {
	schema := &GraphQLSchema{
		Types:     make(map[string]*GraphQLType),
		Mutations: make(map[string]*GraphQLField),
		Queries:   make(map[string]*GraphQLField),
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".graphqls") {
			return nil
		}
		return p.parseFile(path, schema)
	})
	if err != nil {
		return schema, err
	}

	return schema, nil
}

// parseFile parses a single .graphqls file and merges results into the schema.
func (p *GraphQLSchemaParser) parseFile(path string, schema *GraphQLSchema) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	p.parseContent(string(data), schema)
	return nil
}

// parseContent parses GraphQL schema content and merges into the schema.
func (p *GraphQLSchemaParser) parseContent(content string, schema *GraphQLSchema) {
	lines := strings.Split(content, "\n")

	inBlock := false
	inDocString := false
	var blockTarget string // "type:FooBar", "mutation", "query"
	var currentType *GraphQLType

	for _, line := range lines {
		// Handle doc strings (triple-quoted).
		if reTripleQuote.MatchString(line) {
			inDocString = !inDocString
			continue
		}
		if inDocString {
			continue
		}

		// Skip comments and directives.
		if reComment.MatchString(line) {
			continue
		}

		// Skip scalar/enum/union/interface declarations.
		if reScalarOrEnum.MatchString(line) {
			continue
		}

		if !inBlock {
			// Check for type/input definition.
			if m := reTypeStart.FindStringSubmatch(line); m != nil {
				typeName := m[2]
				// Skip Mutation and Query root types (those are handled via extend).
				if typeName == "Mutation" || typeName == "Query" {
					blockTarget = strings.ToLower(typeName)
					inBlock = true
					currentType = nil
					continue
				}
				currentType = &GraphQLType{Name: typeName}
				blockTarget = "type:" + typeName
				inBlock = true
				continue
			}

			// Check for extend type Mutation/Query.
			if m := reExtendType.FindStringSubmatch(line); m != nil {
				blockTarget = strings.ToLower(m[1])
				inBlock = true
				currentType = nil
				continue
			}

			continue
		}

		// Inside a block.
		if reBlockClose.MatchString(line) {
			if currentType != nil {
				schema.Types[currentType.Name] = currentType
			}
			inBlock = false
			currentType = nil
			blockTarget = ""
			continue
		}

		// Try to parse a field.
		if m := reField.FindStringSubmatch(line); m != nil {
			fieldName := m[1]
			argsStr := m[2]
			fieldType := strings.TrimSpace(m[3])

			field := GraphQLField{
				Name:     fieldName,
				Type:     fieldType,
				Required: isRequired(fieldType),
			}

			// Parse arguments if present.
			if argsStr != "" {
				field.Args = parseArgs(argsStr)
				field.ReturnType = fieldType
			}

			switch {
			case blockTarget == "mutation":
				schema.Mutations[fieldName] = &field
			case blockTarget == "query":
				schema.Queries[fieldName] = &field
			case currentType != nil:
				currentType.Fields = append(currentType.Fields, field)
			}
		}
	}
}

// isRequired returns true if a GraphQL type ends with "!" (non-nullable).
func isRequired(typeName string) bool {
	return strings.HasSuffix(strings.TrimSpace(typeName), "!")
}

// parseArgs parses a GraphQL argument list string like "input: TrainingLogSetInput!, limit: Int".
func parseArgs(argsStr string) []GraphQLField {
	matches := reArg.FindAllStringSubmatch(argsStr, -1)
	args := make([]GraphQLField, 0, len(matches))
	for _, m := range matches {
		args = append(args, GraphQLField{
			Name:     m[1],
			Type:     m[2],
			Required: isRequired(m[2]),
		})
	}
	return args
}
