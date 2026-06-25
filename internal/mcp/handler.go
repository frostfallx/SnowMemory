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
			User:    &models.User{UserID: request.UserID},
			Aliases: []models.UserAlias{},
			Facts:   []models.LongTermFact{},
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

	return &models.UserProfile{
		User:    user,
		Aliases: aliases,
		Facts:   facts,
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
