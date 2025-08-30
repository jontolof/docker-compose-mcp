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