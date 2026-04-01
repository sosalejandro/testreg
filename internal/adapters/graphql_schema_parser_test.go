// @testreg trace.graphql-schema-parser
package adapters

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGraphQLSchemaParser_ParseSimpleType(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "types.graphqls", `
type User {
  id: UUID!
  name: String!
  email: String
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	userType, ok := schema.Types["User"]
	if !ok {
		t.Fatal("expected User type in schema")
	}

	if len(userType.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(userType.Fields))
	}

	// id: UUID! — required
	assertField(t, userType.Fields[0], "id", "UUID!", true)
	// name: String! — required
	assertField(t, userType.Fields[1], "name", "String!", true)
	// email: String — optional
	assertField(t, userType.Fields[2], "email", "String", false)
}

func TestGraphQLSchemaParser_ParseInputType(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "inputs.graphqls", `
input TrainingLogSetInput {
  sessionId: UUID!
  exerciseId: UUID!
  reps: Int
  weight: Float
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	inputType, ok := schema.Types["TrainingLogSetInput"]
	if !ok {
		t.Fatal("expected TrainingLogSetInput type in schema")
	}

	if len(inputType.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(inputType.Fields))
	}

	assertField(t, inputType.Fields[0], "sessionId", "UUID!", true)
	assertField(t, inputType.Fields[1], "exerciseId", "UUID!", true)
	assertField(t, inputType.Fields[2], "reps", "Int", false)
	assertField(t, inputType.Fields[3], "weight", "Float", false)
}

func TestGraphQLSchemaParser_ParseExtendTypeMutation(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "mutations.graphqls", `
extend type Mutation {
  trainingLogSet(input: TrainingLogSetInput!): TrainingExerciseSet!
  deleteSession(id: UUID!): Boolean!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if len(schema.Mutations) != 2 {
		t.Fatalf("expected 2 mutations, got %d", len(schema.Mutations))
	}

	logSet, ok := schema.Mutations["trainingLogSet"]
	if !ok {
		t.Fatal("expected trainingLogSet mutation")
	}

	if logSet.Type != "TrainingExerciseSet!" {
		t.Errorf("expected return type TrainingExerciseSet!, got %s", logSet.Type)
	}

	if len(logSet.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(logSet.Args))
	}
	if logSet.Args[0].Name != "input" {
		t.Errorf("expected arg name 'input', got %s", logSet.Args[0].Name)
	}
	if logSet.Args[0].Type != "TrainingLogSetInput!" {
		t.Errorf("expected arg type 'TrainingLogSetInput!', got %s", logSet.Args[0].Type)
	}
}

func TestGraphQLSchemaParser_ParseFieldWithArguments(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "queries.graphqls", `
extend type Query {
  trainingSessions(limit: Int, offset: Int): [TrainingSession!]!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	sessions, ok := schema.Queries["trainingSessions"]
	if !ok {
		t.Fatal("expected trainingSessions query")
	}

	if len(sessions.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(sessions.Args))
	}

	if sessions.Args[0].Name != "limit" {
		t.Errorf("expected first arg 'limit', got %s", sessions.Args[0].Name)
	}
	if sessions.Args[1].Name != "offset" {
		t.Errorf("expected second arg 'offset', got %s", sessions.Args[1].Name)
	}

	if sessions.Type != "[TrainingSession!]!" {
		t.Errorf("expected return type '[TrainingSession!]!', got %s", sessions.Type)
	}
}

func TestGraphQLSchemaParser_RequiredVsOptionalFields(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "types.graphqls", `
type Product {
  id: UUID!
  name: String!
  description: String
  price: Float!
  tags: [String]
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	product, ok := schema.Types["Product"]
	if !ok {
		t.Fatal("expected Product type")
	}

	assertField(t, product.Fields[0], "id", "UUID!", true)
	assertField(t, product.Fields[1], "name", "String!", true)
	assertField(t, product.Fields[2], "description", "String", false)
	assertField(t, product.Fields[3], "price", "Float!", true)
	assertField(t, product.Fields[4], "tags", "[String]", false)
}

func TestGraphQLSchemaParser_ParseListTypes(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "types.graphqls", `
type Container {
  items: [String!]!
  optionalItems: [String]
  nestedList: [Int!]
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	container, ok := schema.Types["Container"]
	if !ok {
		t.Fatal("expected Container type")
	}

	if len(container.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(container.Fields))
	}

	// [String!]! — list is required
	assertField(t, container.Fields[0], "items", "[String!]!", true)
	// [String] — list is optional
	assertField(t, container.Fields[1], "optionalItems", "[String]", false)
	// [Int!] — list is optional, but elements are required
	assertField(t, container.Fields[2], "nestedList", "[Int!]", false)
}

func TestGraphQLSchemaParser_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "empty.graphqls", "")

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if len(schema.Types) != 0 {
		t.Errorf("expected 0 types, got %d", len(schema.Types))
	}
	if len(schema.Mutations) != 0 {
		t.Errorf("expected 0 mutations, got %d", len(schema.Mutations))
	}
	if len(schema.Queries) != 0 {
		t.Errorf("expected 0 queries, got %d", len(schema.Queries))
	}
}

func TestGraphQLSchemaParser_MultipleTypesInOneFile(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "schema.graphqls", `
type User {
  id: UUID!
  name: String!
}

input CreateUserInput {
  name: String!
  email: String!
}

type UserResponse {
  user: User!
  token: String!
}

extend type Mutation {
  createUser(input: CreateUserInput!): UserResponse!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if len(schema.Types) != 3 {
		t.Errorf("expected 3 types, got %d", len(schema.Types))
	}

	if _, ok := schema.Types["User"]; !ok {
		t.Error("expected User type")
	}
	if _, ok := schema.Types["CreateUserInput"]; !ok {
		t.Error("expected CreateUserInput type")
	}
	if _, ok := schema.Types["UserResponse"]; !ok {
		t.Error("expected UserResponse type")
	}

	if len(schema.Mutations) != 1 {
		t.Errorf("expected 1 mutation, got %d", len(schema.Mutations))
	}
}

func TestGraphQLSchemaParser_SubdirectoryScanning(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "subdir")

	writeGraphQLFile(t, dir, "root.graphqls", `
type RootType {
  id: UUID!
}
`)
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeGraphQLFile(t, subDir, "nested.graphqls", `
type NestedType {
  name: String!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if _, ok := schema.Types["RootType"]; !ok {
		t.Error("expected RootType from root directory")
	}
	if _, ok := schema.Types["NestedType"]; !ok {
		t.Error("expected NestedType from subdirectory")
	}
}

func TestGraphQLSchemaParser_RootMutationType(t *testing.T) {
	// Tests "type Mutation { ... }" (not "extend type Mutation")
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "schema.graphqls", `
type Mutation {
  login(email: String!, password: String!): AuthResponse!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	if len(schema.Mutations) != 1 {
		t.Fatalf("expected 1 mutation, got %d", len(schema.Mutations))
	}

	login, ok := schema.Mutations["login"]
	if !ok {
		t.Fatal("expected login mutation")
	}
	if len(login.Args) != 2 {
		t.Errorf("expected 2 args, got %d", len(login.Args))
	}
}

func TestGraphQLSchemaParser_SkipsCommentsAndScalars(t *testing.T) {
	dir := t.TempDir()
	writeGraphQLFile(t, dir, "schema.graphqls", `
# This is a comment
scalar UUID
scalar DateTime

"""
Doc string block
"""
type Item {
  id: UUID!
  createdAt: DateTime!
}
`)

	parser := NewGraphQLSchemaParser()
	schema, err := parser.ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir error: %v", err)
	}

	item, ok := schema.Types["Item"]
	if !ok {
		t.Fatal("expected Item type")
	}

	if len(item.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(item.Fields))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func writeGraphQLFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
}

func assertField(t *testing.T, field GraphQLField, name, typeName string, required bool) {
	t.Helper()
	if field.Name != name {
		t.Errorf("expected field name %q, got %q", name, field.Name)
	}
	if field.Type != typeName {
		t.Errorf("field %q: expected type %q, got %q", name, typeName, field.Type)
	}
	if field.Required != required {
		t.Errorf("field %q: expected required=%v, got %v", name, required, field.Required)
	}
}
