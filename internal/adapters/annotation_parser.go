package adapters

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// AnnotatedTest represents a test file parsed for @testreg annotations.
type AnnotatedTest struct {
	FilePath   string
	FeatureIDs []string
	Flags      []string
	TestType   string // unit, integration, e2e
	Platform   string // backend, web, mobile
	Framework  string // go, vitest, playwright, jest, maestro
	Functions  []ExtractedFunction
}

// ExtractedFunction represents a single test function extracted from a file.
type ExtractedFunction struct {
	Name       string
	Line       int
	FeatureIDs []string
	Flags      []string
}

// annotation holds a parsed @testreg line with its position.
type annotation struct {
	line       int
	featureIDs []string
	flags      []string
}

// Regex patterns for extracting test functions across languages.
var (
	goTestFuncRe       = regexp.MustCompile(`^func\s+(Test\w+|Benchmark\w+)\s*\(`)
	jsTestRe           = regexp.MustCompile(`(?:^|\s)test\(\s*['"](.+?)['"]`)
	jsTestDescribeRe   = regexp.MustCompile(`(?:^|\s)test\.describe\(\s*['"](.+?)['"]`)
	jsItRe             = regexp.MustCompile(`(?:^|\s)it\(\s*['"](.+?)['"]`)
	annotationRe       = regexp.MustCompile(`@testreg\s+(.+)`)
	goBuildTagE2ERe    = regexp.MustCompile(`//go:build\s+e2e`)
	goBuildTagOldE2ERe = regexp.MustCompile(`\+build\s+e2e`)
)

// ParseAnnotatedFile reads a file and extracts @testreg annotations and test functions.
// Returns nil if the file has no annotations (caller can treat it as unmapped).
func ParseAnnotatedFile(filePath string, relPath string) (*AnnotatedTest, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	lang := detectLanguage(ext)
	if lang == "" {
		return nil, nil // unsupported file type
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var (
		annotations     []annotation
		functions       []ExtractedFunction
		hasE2EBuildTag  bool
		lineNum         int
	)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check for Go e2e build tags
		if lang == "go" && lineNum <= 5 {
			if goBuildTagE2ERe.MatchString(line) || goBuildTagOldE2ERe.MatchString(line) {
				hasE2EBuildTag = true
			}
		}

		// Check for @testreg annotations in comments
		if ann := parseAnnotationLine(line, lineNum); ann != nil {
			annotations = append(annotations, *ann)
		}

		// Extract test functions
		if fns := extractFunctions(line, lineNum, lang); len(fns) > 0 {
			functions = append(functions, fns...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// If no annotations found, return nil to indicate unmapped file.
	// Callers should use ParseAnnotatedFileForUnmapped to get function info for unmapped files.
	if len(annotations) == 0 {
		return nil, nil
	}

	// Associate annotations with functions
	result := buildAnnotatedResult(relPath, lang, annotations, functions, hasE2EBuildTag)
	return result, nil
}

// ParseAnnotatedFileForUnmapped parses a file to extract function names only (for unmapped report).
// Returns nil if the file has no test functions.
func ParseAnnotatedFileForUnmapped(filePath string, relPath string) (*AnnotatedTest, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(filePath))
	lang := detectLanguage(ext)
	if lang == "" {
		return nil, nil
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 256*1024)

	var (
		functions      []ExtractedFunction
		hasAnnotation  bool
		hasE2EBuildTag bool
		lineNum        int
	)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if lang == "go" && lineNum <= 5 {
			if goBuildTagE2ERe.MatchString(line) || goBuildTagOldE2ERe.MatchString(line) {
				hasE2EBuildTag = true
			}
		}

		if annotationRe.MatchString(line) {
			hasAnnotation = true
		}

		if fns := extractFunctions(line, lineNum, lang); len(fns) > 0 {
			functions = append(functions, fns...)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if hasAnnotation {
		return nil, nil // has annotations, not unmapped
	}

	if len(functions) == 0 && lang != "maestro" {
		return nil, nil
	}

	return buildUnmappedResult(relPath, lang, functions, hasE2EBuildTag), nil
}

// detectLanguage returns the language identifier based on file extension.
func detectLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx", ".js", ".jsx":
		return "js" // covers TS/JS/JSX/TSX
	case ".yaml", ".yml":
		return "maestro"
	default:
		return ""
	}
}

// parseAnnotationLine checks if a line contains a @testreg annotation and parses it.
func parseAnnotationLine(line string, lineNum int) *annotation {
	match := annotationRe.FindStringSubmatch(line)
	if match == nil {
		return nil
	}

	// Verify this is in a comment
	trimmed := strings.TrimSpace(line)
	isComment := strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*")

	if !isComment {
		return nil
	}

	payload := strings.TrimSpace(match[1])
	return parseAnnotationPayload(payload, lineNum)
}

// parseAnnotationPayload parses the content after @testreg into feature IDs and flags.
func parseAnnotationPayload(payload string, lineNum int) *annotation {
	parts := strings.Fields(payload)
	ann := &annotation{line: lineNum}

	for _, p := range parts {
		if strings.HasPrefix(p, "#") {
			ann.flags = append(ann.flags, p)
		} else {
			// Feature IDs can be comma-separated
			ids := strings.Split(p, ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				if id != "" {
					ann.featureIDs = append(ann.featureIDs, id)
				}
			}
		}
	}

	return ann
}

// extractFunctions extracts test function declarations from a line.
func extractFunctions(line string, lineNum int, lang string) []ExtractedFunction {
	switch lang {
	case "go":
		return extractGoFunctions(line, lineNum)
	case "js":
		return extractJSFunctions(line, lineNum)
	case "maestro":
		// Maestro YAML: filename is the test name, no function extraction
		return nil
	default:
		return nil
	}
}

// extractGoFunctions extracts Go test function names.
func extractGoFunctions(line string, lineNum int) []ExtractedFunction {
	match := goTestFuncRe.FindStringSubmatch(line)
	if match == nil {
		return nil
	}
	return []ExtractedFunction{{Name: match[1], Line: lineNum}}
}

// extractJSFunctions extracts JavaScript/TypeScript test function names.
func extractJSFunctions(line string, lineNum int) []ExtractedFunction {
	var fns []ExtractedFunction

	// Try test() pattern
	if matches := jsTestRe.FindAllStringSubmatch(line, -1); matches != nil {
		for _, m := range matches {
			fns = append(fns, ExtractedFunction{Name: m[1], Line: lineNum})
		}
	}

	// Try test.describe() pattern
	if matches := jsTestDescribeRe.FindAllStringSubmatch(line, -1); matches != nil {
		for _, m := range matches {
			fns = append(fns, ExtractedFunction{Name: m[1], Line: lineNum})
		}
	}

	// Try it() pattern
	if matches := jsItRe.FindAllStringSubmatch(line, -1); matches != nil {
		for _, m := range matches {
			fns = append(fns, ExtractedFunction{Name: m[1], Line: lineNum})
		}
	}

	return fns
}

// buildAnnotatedResult creates an AnnotatedTest from parsed annotations and functions.
func buildAnnotatedResult(relPath, lang string, annotations []annotation, functions []ExtractedFunction, hasE2EBuildTag bool) *AnnotatedTest {
	testType, platform := classifyFromPath(relPath, lang, hasE2EBuildTag)
	framework := frameworkFromLang(lang, relPath)

	result := &AnnotatedTest{
		FilePath:  relPath,
		TestType:  testType,
		Platform:  platform,
		Framework: framework,
	}

	// Classify annotations as test-level or file-level.
	// An annotation is test-level if:
	//   1. A function declaration appears on the very next line (annotation line + 1)
	//   2. There's no blank line between them
	// Everything else is file-level.

	// For each annotation, check if a function declaration appears on the very next line.
	// If so, that annotation is test-level for that function.
	annToFunc := make(map[int]int) // annotation index -> function index
	usedFuncs := make(map[int]bool)

	for ai, ann := range annotations {
		for fi, fn := range functions {
			if usedFuncs[fi] {
				continue
			}
			// Test-level: function is on the immediately next line
			if fn.Line == ann.line+1 {
				annToFunc[ai] = fi
				usedFuncs[fi] = true
				break
			}
		}
	}

	testLevelAnns := make(map[int]bool) // annotation index
	funcToAnn := make(map[int]int)      // function index -> annotation index
	for ai, fi := range annToFunc {
		testLevelAnns[ai] = true
		funcToAnn[fi] = ai
	}

	var fileLevelAnnotations []annotation
	for i, ann := range annotations {
		if !testLevelAnns[i] {
			fileLevelAnnotations = append(fileLevelAnnotations, ann)
		}
	}

	// Collect file-level feature IDs and flags
	for _, ann := range fileLevelAnnotations {
		result.FeatureIDs = append(result.FeatureIDs, ann.featureIDs...)
		result.Flags = append(result.Flags, ann.flags...)
	}
	result.FeatureIDs = dedup(result.FeatureIDs)
	result.Flags = dedup(result.Flags)

	// For maestro YAML files, the filename is the test
	if lang == "maestro" {
		baseName := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
		fn := ExtractedFunction{
			Name:       baseName,
			Line:       1,
			FeatureIDs: result.FeatureIDs,
			Flags:      result.Flags,
		}
		result.Functions = []ExtractedFunction{fn}
		return result
	}

	// Associate each function with its annotation
	for i := range functions {
		fn := &functions[i]
		if ai, ok := funcToAnn[i]; ok {
			// This function has a direct (test-level) annotation
			ann := annotations[ai]
			fn.FeatureIDs = ann.featureIDs
			fn.Flags = ann.flags
		} else {
			// Fall back to file-level annotations
			var featureIDs, flags []string
			for _, ann := range fileLevelAnnotations {
				featureIDs = append(featureIDs, ann.featureIDs...)
				flags = append(flags, ann.flags...)
			}
			fn.FeatureIDs = dedup(featureIDs)
			fn.Flags = dedup(flags)
		}
		result.Functions = append(result.Functions, *fn)
	}

	return result
}

// buildUnmappedResult creates an AnnotatedTest for unmapped files (no annotations).
func buildUnmappedResult(relPath, lang string, functions []ExtractedFunction, hasE2EBuildTag bool) *AnnotatedTest {
	testType, platform := classifyFromPath(relPath, lang, hasE2EBuildTag)
	framework := frameworkFromLang(lang, relPath)

	result := &AnnotatedTest{
		FilePath:  relPath,
		TestType:  testType,
		Platform:  platform,
		Framework: framework,
		Functions: functions,
	}

	// For maestro YAML files, the filename is the test
	if lang == "maestro" && len(functions) == 0 {
		baseName := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
		result.Functions = []ExtractedFunction{{Name: baseName, Line: 1}}
	}

	return result
}

// classifyFromPath determines test type and platform from the relative file path.
func classifyFromPath(relPath, lang string, hasE2EBuildTag bool) (string, string) {
	lower := strings.ToLower(filepath.ToSlash(relPath))

	switch lang {
	case "go":
		if hasE2EBuildTag || strings.Contains(lower, "_e2e_test.go") || strings.Contains(lower, "/e2e/") {
			return "e2e", "backend"
		}
		if strings.Contains(lower, "_integration_test.go") || strings.Contains(lower, "/integration/") {
			return "integration", "backend"
		}
		return "unit", "backend"

	case "js":
		// Web paths
		if strings.HasPrefix(lower, "apps/web/e2e/") || strings.HasPrefix(lower, "web/e2e/") {
			return "e2e", "web"
		}
		if strings.HasPrefix(lower, "apps/web/tests/") || strings.HasPrefix(lower, "web/tests/") ||
			strings.HasPrefix(lower, "apps/web/src/") {
			return "unit", "web"
		}

		// Mobile paths
		if strings.Contains(lower, "apps/mobile/src/__tests__/integration/") {
			return "integration", "mobile"
		}
		if strings.Contains(lower, "apps/mobile/src/__tests__/") {
			return "unit", "mobile"
		}
		if strings.HasPrefix(lower, "apps/mobile/e2e/") {
			return "e2e", "mobile"
		}

		// Spec files are typically e2e
		if strings.HasSuffix(lower, ".spec.ts") || strings.HasSuffix(lower, ".spec.js") {
			return "e2e", "web"
		}

		// Default for JS
		return "unit", "web"

	case "maestro":
		return "e2e", "mobile"
	}

	return "unit", "backend"
}

// frameworkFromLang determines the test framework from language and path.
func frameworkFromLang(lang, relPath string) string {
	lower := strings.ToLower(filepath.ToSlash(relPath))

	switch lang {
	case "go":
		return "go"
	case "js":
		// Spec files = Playwright
		if strings.HasSuffix(lower, ".spec.ts") || strings.HasSuffix(lower, ".spec.js") {
			return "playwright"
		}
		// Mobile test files = Jest
		if strings.Contains(lower, "apps/mobile/") || strings.Contains(lower, "mobile/") {
			return "jest"
		}
		// Web test files = Vitest
		return "vitest"
	case "maestro":
		return "maestro"
	}
	return ""
}

// dedup removes duplicate strings from a slice while preserving order.
func dedup(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
