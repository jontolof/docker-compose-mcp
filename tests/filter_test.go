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

func TestOutputFilter_FilterMigrationOutput(t *testing.T) {
	f := filter.NewOutputFilter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "keeps migration success messages",
			input: `Connecting to database...
Connection established
Migration 001_create_users.up.sql applied successfully
Migration 002_add_indexes.up.sql applied successfully
Migration completed successfully
Verbose debug info...`,
			expected: `Migration 001_create_users.up.sql applied successfully
Migration 002_add_indexes.up.sql applied successfully
Migration completed successfully`,
		},
		{
			name: "keeps migration errors",
			input: `Connecting to database...
Starting migration...
Migration 001_create_users.up.sql applied successfully
ERROR: Migration 002_bad_syntax.up.sql failed: syntax error
Migration failed`,
			expected: `Starting migration...
Migration 001_create_users.up.sql applied successfully
ERROR: Migration 002_bad_syntax.up.sql failed: syntax error
Migration failed`,
		},
		{
			name: "keeps database operations",
			input: `Creating table users...
Creating table posts...
Adding foreign key constraints...
Creating indexes...
Migration completed`,
			expected: `Creating table users...
Creating table posts...
Adding foreign key constraints...
Creating indexes...
Migration completed`,
		},
		{
			name: "empty migration returns completion message",
			input: "",
			expected: "Migration command completed",
		},
		{
			name: "keeps version information",
			input: `Current schema version: 001
Applying migration version 002
Migration version 002 applied successfully
Schema version updated to 002`,
			expected: `Current schema version: 001
Applying migration version 002
Migration version 002 applied successfully
Schema version updated to 002`,
		},
		{
			name: "filters verbose connection info",
			input: `Connecting to database postgresql://user@host:5432/db
Connection established successfully
Debug: Query execution time: 0.5ms
Migration 001_users.up.sql applied successfully
Debug: Another verbose message
Migration completed`,
			expected: `Migration 001_users.up.sql applied successfully
Migration completed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.FilterMigrationOutput(tt.input)
			if result != tt.expected {
				t.Errorf("FilterMigrationOutput() = %q, expected %q", result, tt.expected)
			}
		})
	}
}