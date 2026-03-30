package domain

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// regexSpecialChars matches characters that are special in regex.
var regexSpecialChars = regexp.MustCompile(`[.*+?^${}()|[\]\\]`)

// GenerateRunCommand builds the shell command to run a specific test function.
// It takes the test metadata and a function name, and returns the exact command.
func GenerateRunCommand(framework, testType, filePath, funcName string) string {
	switch framework {
	case "go":
		return generateGoRunCommand(testType, filePath, funcName)
	case "playwright":
		return generatePlaywrightRunCommand(funcName)
	case "vitest":
		return generateVitestRunCommand(funcName)
	case "jest":
		return generateJestRunCommand(funcName)
	case "maestro":
		return generateMaestroRunCommand(filePath)
	default:
		return fmt.Sprintf("# unsupported framework: %s", framework)
	}
}

// generateGoRunCommand produces `go test -run FuncName ./package/...`
func generateGoRunCommand(testType, filePath, funcName string) string {
	pkgPath := goPackagePath(filePath)

	if testType == "e2e" {
		return fmt.Sprintf("go test -tags=e2e -run %s %s", funcName, pkgPath)
	}
	return fmt.Sprintf("go test -run %s %s", funcName, pkgPath)
}

// goPackagePath derives the Go package path from a file path.
// e.g., "src/application/services/auth_test.go" -> "./src/application/services/..."
func goPackagePath(filePath string) string {
	dir := filepath.ToSlash(filepath.Dir(filePath))
	if dir == "." || dir == "" {
		return "./..."
	}
	return "./" + dir + "/..."
}

// generatePlaywrightRunCommand produces `npx playwright test -g 'TestName'`
func generatePlaywrightRunCommand(funcName string) string {
	escaped := escapeRegex(funcName)
	return fmt.Sprintf("npx playwright test -g '%s'", escaped)
}

// generateVitestRunCommand produces `npx vitest -t 'TestName'`
func generateVitestRunCommand(funcName string) string {
	return fmt.Sprintf("npx vitest -t '%s'", funcName)
}

// generateJestRunCommand produces `npx jest -t 'TestName'`
func generateJestRunCommand(funcName string) string {
	return fmt.Sprintf("npx jest -t '%s'", funcName)
}

// generateMaestroRunCommand produces `maestro test FilePath`
func generateMaestroRunCommand(filePath string) string {
	return fmt.Sprintf("maestro test %s", filePath)
}

// escapeRegex escapes special regex characters in a string.
func escapeRegex(s string) string {
	return regexSpecialChars.ReplaceAllString(s, `\$0`)
}

// CollectRunCommands gathers all run commands from a feature's coverage entries.
// It returns command strings filtered by optional platform and test type constraints.
func CollectRunCommands(f *Feature, platform, testType string) []string {
	var commands []string

	collect := func(entry *CoverageEntry, entryType, entryPlatform string) {
		if entry == nil {
			return
		}
		if platform != "" && entryPlatform != platform {
			return
		}
		if testType != "" && entryType != testType {
			return
		}
		for _, te := range entry.Tests {
			for _, fn := range te.Functions {
				if fn.Run != "" {
					commands = append(commands, fn.Run)
				}
			}
		}
	}

	collectE2E := func(entry *E2ECoverageEntry, entryType, entryPlatform string) {
		if entry == nil {
			return
		}
		if platform != "" && entryPlatform != platform {
			return
		}
		if testType != "" && entryType != testType {
			return
		}
		for _, te := range entry.Tests {
			for _, fn := range te.Functions {
				if fn.Run != "" {
					commands = append(commands, fn.Run)
				}
			}
		}
	}

	collect(f.Coverage.Unit.Backend, "unit", "backend")
	collect(f.Coverage.Unit.Web, "unit", "web")
	collect(f.Coverage.Unit.Mobile, "unit", "mobile")
	collect(f.Coverage.Integration.Backend, "integration", "backend")
	collect(f.Coverage.Integration.Mobile, "integration", "mobile")
	collectE2E(f.Coverage.E2E.Web, "e2e", "web")
	collectE2E(f.Coverage.E2E.Mobile, "e2e", "mobile")

	return commands
}

// CollectRunCommandsByPriority gathers all run commands from features matching a priority.
func CollectRunCommandsByPriority(features []Feature, priority Priority, platform, testType string) []string {
	var commands []string
	for i := range features {
		if features[i].Priority != priority {
			continue
		}
		commands = append(commands, CollectRunCommands(&features[i], platform, testType)...)
	}
	return commands
}

// CollectFailingRunCommands gathers run commands from features that have failing status.
func CollectFailingRunCommands(features []Feature, platform, testType string) []string {
	var commands []string
	for i := range features {
		f := &features[i]
		entries := f.AllCoverageEntries()
		hasFailing := false
		for _, s := range entries {
			if s.IsFailing() {
				hasFailing = true
				break
			}
		}
		if !hasFailing {
			continue
		}

		// Collect all commands from failing entries specifically
		collectFailing := func(entry *CoverageEntry, entryType, entryPlatform string) {
			if entry == nil || !entry.Status.IsFailing() {
				return
			}
			if platform != "" && entryPlatform != platform {
				return
			}
			if testType != "" && entryType != testType {
				return
			}
			for _, te := range entry.Tests {
				for _, fn := range te.Functions {
					if fn.Run != "" {
						commands = append(commands, fn.Run)
					}
				}
			}
		}
		collectFailingE2E := func(entry *E2ECoverageEntry, entryType, entryPlatform string) {
			if entry == nil || !entry.Status.IsFailing() {
				return
			}
			if platform != "" && entryPlatform != platform {
				return
			}
			if testType != "" && entryType != testType {
				return
			}
			for _, te := range entry.Tests {
				for _, fn := range te.Functions {
					if fn.Run != "" {
						commands = append(commands, fn.Run)
					}
				}
			}
		}

		collectFailing(f.Coverage.Unit.Backend, "unit", "backend")
		collectFailing(f.Coverage.Unit.Web, "unit", "web")
		collectFailing(f.Coverage.Unit.Mobile, "unit", "mobile")
		collectFailing(f.Coverage.Integration.Backend, "integration", "backend")
		collectFailing(f.Coverage.Integration.Mobile, "integration", "mobile")
		collectFailingE2E(f.Coverage.E2E.Web, "e2e", "web")
		collectFailingE2E(f.Coverage.E2E.Mobile, "e2e", "mobile")
	}
	return commands
}

// HasFlag checks if a flag is present in a slice of flags.
func HasFlag(flags []string, flag string) bool {
	lower := strings.ToLower(flag)
	for _, f := range flags {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}
