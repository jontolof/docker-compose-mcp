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