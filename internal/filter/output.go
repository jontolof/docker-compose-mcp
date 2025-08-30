package filter

import (
	"regexp"
	"strings"
)

type OutputFilter struct {
	keepPatterns []string
	skipPatterns []string
}

func NewOutputFilter() *OutputFilter {
	return &OutputFilter{
		keepPatterns: []string{
			"Error",
			"ERRO",
			"FAIL",
			"WARNING",
			"WARN",
			"Creating",
			"Starting",
			"Stopping",
			"Removing",
			"Building",
			"Pulling.*\\(",
			"Pushed",
			"Successfully",
			"exited with code",
			"Status:",
		},
		skipPatterns: []string{
			"Pulling fs layer",
			"Downloading",
			"Extracting",
			"Pull complete",
			"Waiting",
			"Verifying Checksum",
			"Download complete",
			"Already exists",
			"^\\d+: Pulling from",
		},
	}
}

func (f *OutputFilter) Filter(output string) string {
	lines := strings.Split(output, "\n")
	var filtered []string
	
	for _, line := range lines {
		if f.shouldKeepLine(line) {
			filtered = append(filtered, line)
		}
	}
	
	if len(filtered) == 0 && len(lines) > 0 {
		return "Command completed successfully"
	}
	
	return strings.Join(filtered, "\n")
}

func (f *OutputFilter) shouldKeepLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	
	for _, pattern := range f.skipPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, line); matched {
			return false
		}
	}
	
	for _, pattern := range f.keepPatterns {
		if matched, _ := regexp.MatchString("(?i)"+pattern, line); matched {
			return true
		}
	}
	
	if strings.Contains(line, ":") && (strings.Contains(line, "http") || strings.Contains(line, "tcp")) {
		return true
	}
	
	return len(line) < 100
}

func (f *OutputFilter) FilterTestOutput(output, framework string) string {
	lines := strings.Split(output, "\n")
	var filtered []string
	var summary []string
	
	switch framework {
	case "go":
		filtered, summary = f.filterGoTestOutput(lines)
	case "jest", "node":
		filtered, summary = f.filterJestTestOutput(lines)
	case "pytest", "python":
		filtered, summary = f.filterPytestOutput(lines)
	default:
		filtered, summary = f.filterGenericTestOutput(lines)
	}
	
	result := strings.Join(filtered, "\n")
	if len(summary) > 0 {
		result += "\n\n" + strings.Join(summary, "\n")
	}
	
	return result
}

func (f *OutputFilter) filterGoTestOutput(lines []string) ([]string, []string) {
	var filtered []string
	var summary []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if regexp.MustCompile(`^=== RUN`).MatchString(line) {
			continue
		}
		
		if regexp.MustCompile(`^--- (PASS|FAIL|SKIP):`).MatchString(line) ||
		   regexp.MustCompile(`^(PASS|FAIL|ok)\s+`).MatchString(line) ||
		   strings.Contains(line, "coverage:") ||
		   strings.Contains(line, "FAIL") ||
		   strings.Contains(line, "panic") ||
		   strings.Contains(line, "Error") ||
		   regexp.MustCompile(`^\d+\s+(passed|failed)`).MatchString(line) {
			filtered = append(filtered, line)
		}
	}
	
	for _, line := range lines {
		if regexp.MustCompile(`^(PASS|FAIL|ok)\s+.*\s+\d+\.\d+s`).MatchString(line) ||
		   strings.Contains(line, "coverage:") {
			summary = append(summary, line)
		}
	}
	
	return filtered, summary
}

func (f *OutputFilter) filterJestTestOutput(lines []string) ([]string, []string) {
	var filtered []string
	var summary []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.Contains(line, "PASS") ||
		   strings.Contains(line, "FAIL") ||
		   strings.Contains(line, "Error") ||
		   strings.Contains(line, "Test Suites:") ||
		   strings.Contains(line, "Tests:") ||
		   strings.Contains(line, "Snapshots:") ||
		   strings.Contains(line, "Time:") ||
		   strings.Contains(line, "Coverage") {
			filtered = append(filtered, line)
		}
	}
	
	for _, line := range lines {
		if strings.Contains(line, "Test Suites:") ||
		   strings.Contains(line, "Tests:") ||
		   strings.Contains(line, "Time:") {
			summary = append(summary, line)
		}
	}
	
	return filtered, summary
}

func (f *OutputFilter) filterPytestOutput(lines []string) ([]string, []string) {
	var filtered []string
	var summary []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if regexp.MustCompile(`^=+.*=+$`).MatchString(line) ||
		   strings.Contains(line, "FAILED") ||
		   strings.Contains(line, "ERROR") ||
		   strings.Contains(line, "passed") ||
		   strings.Contains(line, "failed") ||
		   strings.Contains(line, "error") ||
		   strings.Contains(line, "coverage") {
			filtered = append(filtered, line)
		}
	}
	
	for _, line := range lines {
		if regexp.MustCompile(`=+\s*\d+.*in\s+\d+`).MatchString(line) {
			summary = append(summary, line)
		}
	}
	
	return filtered, summary
}

func (f *OutputFilter) filterGenericTestOutput(lines []string) ([]string, []string) {
	var filtered []string
	var summary []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.Contains(strings.ToLower(line), "pass") ||
		   strings.Contains(strings.ToLower(line), "fail") ||
		   strings.Contains(strings.ToLower(line), "error") ||
		   strings.Contains(strings.ToLower(line), "test") {
			filtered = append(filtered, line)
		}
	}
	
	return filtered, summary
}