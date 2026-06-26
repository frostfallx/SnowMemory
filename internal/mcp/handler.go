package mcp

import (
	"context"
	"fmt"

	"github.com/vmwin11/snowmemory/internal/database"
	"github.com/vmwin11/snowmemory/models"
)

// QueryUserProfileRequest 查询用户资料请求
type QueryUserProfileRequest struct {
	UserID  string `json:"user_id"`
	GroupID string `json:"group_id"`
}

// QueryUserProfileMCP 查询用户资料的 MCP 工具
func QueryUserProfileMCP(ctx context.Context, request QueryUserProfileRequest) (any, error) {
	if request.UserID == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "user_id is required"}
	}

	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to query user: %v", err)}
	}

	if user == nil {
		return &models.UserProfile{
			User:        &models.User{UserID: request.UserID},
			Aliases:     []models.UserAlias{},
			Facts:       []models.LongTermFact{},
			GlobalCalls: []string{},
			CurrentCall: "",
			CallSummary: map[string]string{},
		}, nil
	}

	aliasRepo := &database.AliasRepository{}
	aliases, err := aliasRepo.GetUserAliases(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to query aliases: %v", err)}
	}

	factRepo := &database.FactRepository{}
	facts, err := factRepo.GetUserFacts(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to query facts: %v", err)}
	}

	// 构建群组称呼映射
	callSummary := make(map[string]string)
	var globalCalls []string
	var currentCall string

	for _, alias := range aliases {
		callSummary[alias.GroupID] = alias.CalledName
		globalCalls = append(globalCalls, alias.CalledName)

		// 如果是当前群，记录当前称呼
		if alias.GroupID == request.GroupID {
			currentCall = alias.CalledName
		}
	}

	// 如果当前群没有称呼，使用全局第一个称呼或空字符串
	if currentCall == "" && len(globalCalls) > 0 {
		currentCall = globalCalls[0]
	}

	return &models.UserProfile{
		User:        user,
		Aliases:     aliases,
		Facts:       facts,
		GlobalCalls: globalCalls,
		CurrentCall: currentCall,
		CallSummary: callSummary,
	}, nil
}

// LearnUserAliasRequest 学习用户别名请求
type LearnUserAliasRequest struct {
	UserID     string `json:"user_id"`
	GroupID    string `json:"group_id"`
	CalledName string `json:"called_name"`
}

// LearnUserAliasMCP 学习用户别名的 MCP 工具
func LearnUserAliasMCP(ctx context.Context, request LearnUserAliasRequest) (any, error) {
	if request.UserID == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "user_id is required"}
	}
	if request.GroupID == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "group_id is required"}
	}
	if request.CalledName == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "called_name is required"}
	}

	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to query user: %v", err)}
	}

	if user == nil {
		if err := userRepo.CreateUser(request.UserID); err != nil {
			return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to create user: %v", err)}
		}
	}

	aliasRepo := &database.AliasRepository{}
	alias := &models.UserAlias{
		UserID:     request.UserID,
		GroupID:    request.GroupID,
		CalledName: request.CalledName,
	}

	if err := aliasRepo.UpsertAlias(alias); err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to update alias: %v", err)}
	}

	return map[string]string{"status": "ok"}, nil
}

// ExtractAndStoreFactRequest 提取并存储事实请求
type ExtractAndStoreFactRequest struct {
	UserID   string `json:"user_id"`
	Category string `json:"category"`
	FactText string `json:"fact_text"`
}

// ExtractAndStoreFactMCP 提取并存储长期事实的 MCP 工具
func ExtractAndStoreFactMCP(ctx context.Context, request ExtractAndStoreFactRequest) (any, error) {
	if request.UserID == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "user_id is required"}
	}
	if request.FactText == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "fact_text is required"}
	}

	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to query user: %v", err)}
	}

	if user == nil {
		if err := userRepo.CreateUser(request.UserID); err != nil {
			return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to create user: %v", err)}
		}
	}

	factRepo := &database.FactRepository{}
	fact := &models.LongTermFact{
		UserID:   request.UserID,
		Category: request.Category,
		FactText: request.FactText,
	}

	if err := factRepo.CreateFact(fact); err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to store fact: %v", err)}
	}

	return map[string]string{"status": "ok"}, nil
}

// CreateCommonFactRequest 创建常识请求
type CreateCommonFactRequest struct {
	Category string `json:"category"`
	FactText string `json:"fact_text"`
}

// CreateCommonFactMCP 创建常识（所有人都适用）的 MCP 工具
func CreateCommonFactMCP(ctx context.Context, request CreateCommonFactRequest) (any, error) {
	if request.FactText == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "fact_text is required"}
	}

	factRepo := &database.FactRepository{}
	fact := &models.LongTermFact{
		UserID:   "common", // 特殊用户ID表示常识
		Category: request.Category,
		FactText: request.FactText,
		IsCommon: true,
	}

	if err := factRepo.CreateFact(fact); err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to store fact: %v", err)}
	}

	return map[string]string{"status": "ok"}, nil
}

// GetUserFactsRequest 获取用户事实请求
type GetUserFactsRequest struct {
	UserID string `json:"user_id"`
}

// GetUserFactsMCP 获取指定用户所有事实的 MCP 工具
func GetUserFactsMCP(ctx context.Context, request GetUserFactsRequest) (any, error) {
	if request.UserID == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "user_id is required"}
	}

	factRepo := &database.FactRepository{}
	facts, err := factRepo.ListUserFacts(request.UserID)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to get facts: %v", err)}
	}

	return facts, nil
}

// SearchMemoryRequest 搜索记忆请求
type SearchMemoryRequest struct {
	Keyword string `json:"keyword"`
}

// SearchMemoryMCP 根据关键词搜索记忆的 MCP 工具
func SearchMemoryMCP(ctx context.Context, request SearchMemoryRequest) (any, error) {
	if request.Keyword == "" {
		return nil, &JSONRPCError{Code: -32602, Message: "keyword is required"}
	}

	userRepo := &database.UserRepository{}
	aliasRepo := &database.AliasRepository{}
	factRepo := &database.FactRepository{}

	// 搜索用户（通过别名或事实匹配）
	users, err := userRepo.SearchUsersByKeyword(request.Keyword)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to search users: %v", err)}
	}

	// 搜索别名
	aliases, err := aliasRepo.SearchAliasesByKeyword(request.Keyword)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to search aliases: %v", err)}
	}

	// 搜索长期事实
	facts, err := factRepo.SearchFactsByKeyword(request.Keyword)
	if err != nil {
		return nil, &JSONRPCError{Code: -32000, Message: fmt.Sprintf("failed to search facts: %v", err)}
	}

	return &models.SearchResult{
		Users:   users,
		Aliases: aliases,
		Facts:   facts,
	}, nil
}
