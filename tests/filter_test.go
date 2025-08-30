package tests

import (
	"testing"

	"github.com/jontolof/docker-compose-mcp/internal/filter"
)

func TestOutputFilter_Filter(t *testing.T) {
	f := filter.NewOutputFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "keeps important messages",
			input: `Creating network "myapp_default" with the default driver
Creating myapp_db_1 ... done
Creating myapp_web_1 ... done`,
			expected: `Creating network "myapp_default" with the default driver
Creating myapp_db_1 ... done
Creating myapp_web_1 ... done`,
		},
		{
			name: "filters verbose pull output",
			input: `Pulling db (postgres:13)...
13: Pulling from library/postgres
Pulling fs layer
Pulling fs layer
Downloading [==========================>                    ]  35.4MB/68.9MB
Downloading [================================>              ]  44.7MB/68.9MB
Download complete
Pull complete
Status: Downloaded newer image for postgres:13`,
			expected: `Pulling db (postgres:13)...
Status: Downloaded newer image for postgres:13`,
		},
		{
			name: "empty input returns completion message",
			input: "",
			expected: "Command completed successfully",
		},
		{
			name: "keeps error messages",
			input: `Error response from daemon: Conflict. The container name "/myapp_web_1" is already in use`,
			expected: `Error response from daemon: Conflict. The container name "/myapp_web_1" is already in use`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.Filter(tt.input)
			if result != tt.expected {
				t.Errorf("Filter() = %q, expected %q", result, tt.expected)
			}
		})
	}
}