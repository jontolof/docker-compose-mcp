package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Server struct {
	tools map[string]Tool
}

type Request struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema Schema `json:"inputSchema"`
	Handler     func(params interface{}) (interface{}, error)
}

type Schema struct {
	Type       string            `json:"type"`
	Properties map[string]Schema `json:"properties,omitempty"`
	Required   []string          `json:"required,omitempty"`
}

func NewServer() *Server {
	return &Server{
		tools: make(map[string]Tool),
	}
}

func (s *Server) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
}

func (s *Server) GetTools() map[string]Tool {
	return s.tools
}

func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(req.ID, -32700, "Parse error", nil)
			continue
		}

		s.handleRequest(req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req Request) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found", nil)
	}
}

func (s *Server) handleInitialize(req Request) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "docker-compose-mcp",
			"version": "1.0.0",
		},
	}
	s.sendResponse(req.ID, result)
}

func (s *Server) handleToolsList(req Request) {
	var tools []map[string]interface{}
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
	}
	s.sendResponse(req.ID, map[string]interface{}{"tools": tools})
}

func (s *Server) handleToolsCall(req Request) {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		s.sendError(req.ID, -32602, "Invalid params", nil)
		return
	}

	toolName, ok := params["name"].(string)
	if !ok {
		s.sendError(req.ID, -32602, "Missing tool name", nil)
		return
	}

	tool, exists := s.tools[toolName]
	if !exists {
		s.sendError(req.ID, -32602, "Tool not found", nil)
		return
	}

	arguments := params["arguments"]
	result, err := tool.Handler(arguments)
	if err != nil {
		s.sendError(req.ID, -32603, err.Error(), nil)
		return
	}

	s.sendResponse(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	})
}

func (s *Server) sendResponse(id interface{}, result interface{}) {
	resp := Response{
		Jsonrpc: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeJSON(resp)
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := Response{
		Jsonrpc: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeJSON(resp)
}

func (s *Server) writeJSON(v interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write JSON: %v\n", err)
	}
}