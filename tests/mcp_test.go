package tests

import (
	"encoding/json"
	"testing"

	"github.com/jontolof/docker-compose-mcp/internal/mcp"
)

func TestServer_RegisterTool(t *testing.T) {
	server := mcp.NewServer()
	
	tool := mcp.Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: mcp.Schema{Type: "object"},
		Handler: func(params interface{}) (interface{}, error) {
			return "test result", nil
		},
	}
	
	server.RegisterTool(tool)
	
	if len(server.GetTools()) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(server.GetTools()))
	}
}

func TestRequest_JSON(t *testing.T) {
	req := mcp.Request{
		Jsonrpc: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	var decoded mcp.Request
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}
	
	if decoded.Method != "tools/list" {
		t.Errorf("Expected method 'tools/list', got %q", decoded.Method)
	}
}

func TestDatabaseTools_Registration(t *testing.T) {
	server := mcp.NewServer()
	
	migrateHandler := func(params interface{}) (interface{}, error) {
		return "Migration completed", nil
	}
	resetHandler := func(params interface{}) (interface{}, error) {
		return "Database reset completed", nil
	}
	backupHandler := func(params interface{}) (interface{}, error) {
		return "Backup completed", nil
	}
	
	server.RegisterTool(mcp.Tool{
		Name:        "compose_migrate",
		Description: "Run database migrations in a service container",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {Type: "string"},
			},
			Required: []string{"service"},
		},
		Handler: migrateHandler,
	})
	
	server.RegisterTool(mcp.Tool{
		Name:        "compose_db_reset",
		Description: "Reset database to clean state, optionally with seeds",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {Type: "string"},
				"confirm": {Type: "boolean"},
			},
			Required: []string{"service", "confirm"},
		},
		Handler: resetHandler,
	})
	
	server.RegisterTool(mcp.Tool{
		Name:        "compose_db_backup",
		Description: "Create, restore, or list database backups",
		InputSchema: mcp.Schema{
			Type: "object",
			Properties: map[string]mcp.Schema{
				"service": {Type: "string"},
				"action":  {Type: "string"},
			},
			Required: []string{"service", "action"},
		},
		Handler: backupHandler,
	})
	
	tools := server.GetTools()
	
	if len(tools) != 3 {
		t.Errorf("Expected 3 database tools, got %d", len(tools))
	}
	
	expectedTools := []string{"compose_migrate", "compose_db_reset", "compose_db_backup"}
	for _, expected := range expectedTools {
		if _, exists := tools[expected]; !exists {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
	
	migrateTool := tools["compose_migrate"]
	if len(migrateTool.InputSchema.Required) != 1 {
		t.Errorf("Expected 1 required field for migrate tool, got %d", len(migrateTool.InputSchema.Required))
	}
	
	resetTool := tools["compose_db_reset"]
	if len(resetTool.InputSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields for reset tool, got %d", len(resetTool.InputSchema.Required))
	}
	
	backupTool := tools["compose_db_backup"]
	if len(backupTool.InputSchema.Required) != 2 {
		t.Errorf("Expected 2 required fields for backup tool, got %d", len(backupTool.InputSchema.Required))
	}
}