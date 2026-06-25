package mcp

import (
	"context"
	"encoding/json"
)

// Schema 表示 JSON Schema (MCP v2024-11-05 格式)
type Schema struct {
	Type       string             `json:"type"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Required   []string           `json:"required,omitempty"`
}

// Tool 表示 MCP 工具定义
type Tool struct {
	Name        string                                            `json:"name"`
	Description string                                            `json:"description"`
	InputSchema Schema                                            `json:"inputSchema"`
	Handler     func(context.Context, map[string]any) (any, error) `json:"-"`
}

// Server 表示 MCP 服务器
type Server struct {
	Name    string
	Version string
	tools   map[string]Tool
}

// NewServer 创建新的 MCP 服务器
func NewServer(name, version string) *Server {
	return &Server{
		Name:    name,
		Version: version,
		tools:   make(map[string]Tool),
	}
}

// RegisterTool 注册 MCP 工具
func (s *Server) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
}

// GetTools 获取所有已注册的工具
func (s *Server) GetTools() []Tool {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

// CallTool 调用指定工具
func (s *Server) CallTool(name string, params map[string]any) (any, error) {
	tool, exists := s.tools[name]
	if !exists {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: "method not found",
		}
	}

	if tool.Handler == nil {
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: "tool handler not implemented",
		}
	}

	return tool.Handler(context.Background(), params)
}

// JSONRPCError JSON-RPC 错误
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *JSONRPCError) Error() string {
	return e.Message
}

// NewJSONRPCResponse 创建 JSON-RPC 响应
func NewJSONRPCResponse(result any, id interface{}) map[string]any {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"result":  result,
		"id":      id,
	}
	return resp
}

// NewJSONRPCErrorResponse 创建 JSON-RPC 错误响应
func NewJSONRPCErrorResponse(code int, message, data string, id interface{}) map[string]any {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"id": id,
	}
	if data != "" {
		resp["error"].(map[string]any)["data"] = data
	}
	return resp
}

// ParseRequest 解析 JSON-RPC 请求
func ParseRequest(data []byte) (method string, params map[string]any, id interface{}, err error) {
	var req struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      interface{}     `json:"id"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return "", nil, nil, &JSONRPCError{Code: -32700, Message: "parse error"}
	}

	if req.JSONRPC != "2.0" {
		return "", nil, nil, &JSONRPCError{Code: -32600, Message: "invalid request"}
	}

	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return "", nil, nil, &JSONRPCError{Code: -32602, Message: "invalid params"}
		}
	}

	return req.Method, params, req.ID, nil
}
