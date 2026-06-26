package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vmwin11/snowmemory/internal/database"
	internalmcp "github.com/vmwin11/snowmemory/internal/mcp"
	"github.com/vmwin11/snowmemory/internal/web"
)

func main() {
	// 初始化数据库
	dbPath := "./data/snowmemory.db"
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	if err := database.Init(dbPath); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	log.Println("SnowMemory starting...")

	// 创建 MCP 服务器
	mcpServer := internalmcp.NewServer("snowmemory", "1.0.0")
	registerMCPTools(mcpServer)

	// 启动 Web 管理后台 (独立端口)
	apiServer := web.NewAPIServer()
	go func() {
		log.Println("Admin Web UI starting on :9090")
		if err := apiServer.Start("9090"); err != nil {
			log.Printf("Web server stopped: %v", err)
		}
	}()

	// 启动 MCP 服务（stdio 通信）
	go func() {
		log.Println("MCP server ready on stdio")
		runMCPStdio(mcpServer)
	}()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
}

// registerMCPTools 注册所有 MCP 工具
func registerMCPTools(s *internalmcp.Server) {
	s.RegisterTool(internalmcp.Tool{
		Name:        "query_user_profile",
		Description: "查询用户完整资料，包括全局特征和特定群的称呼",
		InputSchema: internalmcp.Schema{
			Type: "object",
			Properties: map[string]*internalmcp.Schema{
				"user_id":  {Type: "string"},
				"group_id": {Type: "string"},
			},
			Required: []string{"user_id", "group_id"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			req := mapToQueryRequest(params)
			return internalmcp.QueryUserProfileMCP(ctx, req)
		},
	})

	s.RegisterTool(internalmcp.Tool{
		Name:        "learn_user_alias",
		Description: "学习用户在特定群的称呼",
		InputSchema: internalmcp.Schema{
			Type: "object",
			Properties: map[string]*internalmcp.Schema{
				"user_id":     {Type: "string"},
				"group_id":    {Type: "string"},
				"called_name": {Type: "string"},
			},
			Required: []string{"user_id", "group_id", "called_name"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			req := mapToAliasRequest(params)
			return internalmcp.LearnUserAliasMCP(ctx, req)
		},
	})

	s.RegisterTool(internalmcp.Tool{
		Name:        "extract_and_store_fact",
		Description: "提取并存储用户的长期事实",
		InputSchema: internalmcp.Schema{
			Type: "object",
			Properties: map[string]*internalmcp.Schema{
				"user_id":   {Type: "string"},
				"category":  {Type: "string"},
				"fact_text": {Type: "string"},
			},
			Required: []string{"user_id", "category", "fact_text"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			req := mapToFactRequest(params)
			return internalmcp.ExtractAndStoreFactMCP(ctx, req)
		},
	})

	s.RegisterTool(internalmcp.Tool{
		Name:        "search_memory",
		Description: "根据关键词搜索用户记忆，包括别名和长期事实。示例：搜索'舞萌'可以找到喜欢玩舞萌的用户",
		InputSchema: internalmcp.Schema{
			Type: "object",
			Properties: map[string]*internalmcp.Schema{
				"keyword": {Type: "string"},
			},
			Required: []string{"keyword"},
		},
		Handler: func(ctx context.Context, params map[string]any) (any, error) {
			req := mapToSearchRequest(params)
			return internalmcp.SearchMemoryMCP(ctx, req)
		},
	})
}

// runMCPStdio 通过 stdio 运行 MCP JSON-RPC 服务
func runMCPStdio(s *internalmcp.Server) {
	reader := bufio.NewReader(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			log.Printf("stdin read error: %v", err)
			return
		}

		line = decodeLine(line)
		log.Printf("Received: %s", string(line))

		method, params, id, parseErr := internalmcp.ParseRequest(line)
		if parseErr != nil {
			if jsonErr, ok := parseErr.(*internalmcp.JSONRPCError); ok {
				resp := internalmcp.NewJSONRPCErrorResponse(jsonErr.Code, jsonErr.Message, jsonErr.Data, id)
				log.Printf("Response: %v", resp)
				encoder.Encode(resp)
			}
			continue
		}

		log.Printf("Method: %s, ID: %v", method, id)

		switch method {
		case "initialize":
			resp := internalmcp.NewJSONRPCResponse(map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]interface{}{
					"name":    s.Name,
					"version": s.Version,
				},
			}, id)
			log.Printf("Response: %v", resp)
			encoder.Encode(resp)

		case "initialized":
			// 初始化完成通知，无需响应
			resp := internalmcp.NewJSONRPCResponse(nil, id)
			encoder.Encode(resp)

		case "tools/list":
			tools := s.GetTools()
			toolList := make([]map[string]interface{}, 0, len(tools))
			for _, t := range tools {
				toolList = append(toolList, map[string]interface{}{
					"name":         t.Name,
					"description":  t.Description,
					"inputSchema": map[string]interface{}{
						"type":       t.InputSchema.Type,
						"properties": t.InputSchema.Properties,
						"required":   t.InputSchema.Required,
					},
				})
			}
			resp := internalmcp.NewJSONRPCResponse(map[string]interface{}{"tools": toolList}, id)
			log.Printf("Response: %v", resp)
			encoder.Encode(resp)

		case "tools/call":
			name, _ := params["name"].(string)
			args, _ := params["arguments"].(map[string]any)
			result, toolErr := s.CallTool(name, args)
			if toolErr != nil {
				resp := internalmcp.NewJSONRPCErrorResponse(-32000, toolErr.Error(), "", id)
				encoder.Encode(resp)
				continue
			}
			resp := internalmcp.NewJSONRPCResponse(map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": formatResult(result)},
				},
			}, id)
			encoder.Encode(resp)

		case "query_user_profile":
			req := mapToQueryRequest(params)
			result, err := internalmcp.QueryUserProfileMCP(context.Background(), req)
			if err != nil {
				if mcpErr, ok := err.(*internalmcp.JSONRPCError); ok {
					resp := internalmcp.NewJSONRPCErrorResponse(mcpErr.Code, mcpErr.Message, mcpErr.Data, id)
					encoder.Encode(resp)
				} else {
					resp := internalmcp.NewJSONRPCErrorResponse(-32000, err.Error(), "", id)
					encoder.Encode(resp)
				}
				continue
			}
			resp := internalmcp.NewJSONRPCResponse(result, id)
			encoder.Encode(resp)

		case "learn_user_alias":
			req := mapToAliasRequest(params)
			result, err := internalmcp.LearnUserAliasMCP(context.Background(), req)
			if err != nil {
				if mcpErr, ok := err.(*internalmcp.JSONRPCError); ok {
					resp := internalmcp.NewJSONRPCErrorResponse(mcpErr.Code, mcpErr.Message, mcpErr.Data, id)
					encoder.Encode(resp)
				} else {
					resp := internalmcp.NewJSONRPCErrorResponse(-32000, err.Error(), "", id)
					encoder.Encode(resp)
				}
				continue
			}
			resp := internalmcp.NewJSONRPCResponse(result, id)
			encoder.Encode(resp)

		case "extract_and_store_fact":
			req := mapToFactRequest(params)
			result, err := internalmcp.ExtractAndStoreFactMCP(context.Background(), req)
			if err != nil {
				if mcpErr, ok := err.(*internalmcp.JSONRPCError); ok {
					resp := internalmcp.NewJSONRPCErrorResponse(mcpErr.Code, mcpErr.Message, mcpErr.Data, id)
					encoder.Encode(resp)
				} else {
					resp := internalmcp.NewJSONRPCErrorResponse(-32000, err.Error(), "", id)
					encoder.Encode(resp)
				}
				continue
			}
			resp := internalmcp.NewJSONRPCResponse(result, id)
			encoder.Encode(resp)

		case "search_memory":
			req := mapToSearchRequest(params)
			result, err := internalmcp.SearchMemoryMCP(context.Background(), req)
			if err != nil {
				if mcpErr, ok := err.(*internalmcp.JSONRPCError); ok {
					resp := internalmcp.NewJSONRPCErrorResponse(mcpErr.Code, mcpErr.Message, mcpErr.Data, id)
					encoder.Encode(resp)
				} else {
					resp := internalmcp.NewJSONRPCErrorResponse(-32000, err.Error(), "", id)
					encoder.Encode(resp)
				}
				continue
			}
			resp := internalmcp.NewJSONRPCResponse(result, id)
			encoder.Encode(resp)

		default:
			resp := internalmcp.NewJSONRPCErrorResponse(-32601, "method not found: "+method, "", id)
			encoder.Encode(resp)
		}
	}
}

// decodeLine 处理 MCP 协议的 Content-Length 头
func decodeLine(line []byte) []byte {
	// 去除换行符
	line = line[:len(line)-1]

	// 检查是否有 Content-Length 头
	if len(line) == 0 {
		return line
	}

	// 如果是 JSON 直接返回
	if line[0] == '{' {
		return line
	}

	// 解析 Content-Length
	var contentLength int
	_, err := fmt.Sscanf(string(line), "Content-Length: %d", &contentLength)
	if err == nil {
		// 读取完整的 JSON 内容
		content := make([]byte, contentLength)
		os.Stdin.Read(content)
		return content
	}

	return line
}

func mapToQueryRequest(params map[string]any) internalmcp.QueryUserProfileRequest {
	return internalmcp.QueryUserProfileRequest{
		UserID:  getString(params, "user_id"),
		GroupID: getString(params, "group_id"),
	}
}

func mapToAliasRequest(params map[string]any) internalmcp.LearnUserAliasRequest {
	return internalmcp.LearnUserAliasRequest{
		UserID:     getString(params, "user_id"),
		GroupID:    getString(params, "group_id"),
		CalledName: getString(params, "called_name"),
	}
}

func mapToFactRequest(params map[string]any) internalmcp.ExtractAndStoreFactRequest {
	return internalmcp.ExtractAndStoreFactRequest{
		UserID:   getString(params, "user_id"),
		Category: getString(params, "category"),
		FactText: getString(params, "fact_text"),
	}
}

func mapToSearchRequest(params map[string]any) internalmcp.SearchMemoryRequest {
	return internalmcp.SearchMemoryRequest{
		Keyword: getString(params, "keyword"),
	}
}

func getString(params map[string]any, key string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func formatResult(result any) string {
	data, err := json.Marshal(result)
	if err != nil {
		return "{}"
	}
	return string(data)
}
