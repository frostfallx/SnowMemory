package curator

import (
	"context"

	"github.com/vmwin11/snowmemory/models"
)

// DefaultCurator 全局默认的 Curator 实例
var DefaultCurator *Curator

// InitDefaultCurator 初始化全局 Curator 实例
func InitDefaultCurator(config *CuratorConfig) {
	DefaultCurator = NewCurator(config)
}

// AnalyzeConversationMCP MCP handler 使用的包装函数
func AnalyzeConversationMCP(ctx context.Context, userID, groupID, conversationText string) (*models.AnalyzeConversationResponse, error) {
	if DefaultCurator == nil {
		return &models.AnalyzeConversationResponse{
			Actions: []models.CuratorAction{},
			Results: []models.ActionResult{},
			Summary: "Curator not initialized",
		}, nil
	}

	req := models.AnalyzeConversationRequest{
		UserID:           userID,
		GroupID:          groupID,
		ConversationText: conversationText,
	}

	return DefaultCurator.AnalyzeConversation(ctx, req)
}

// AnalyzeConversationHTTP HTTP handler 使用的包装函数
func AnalyzeConversationHTTP(ctx context.Context, req models.AnalyzeConversationRequest) (*models.AnalyzeConversationResponse, error) {
	if DefaultCurator == nil {
		return &models.AnalyzeConversationResponse{
			Actions: []models.CuratorAction{},
			Results: []models.ActionResult{},
			Summary: "Curator not initialized",
		}, nil
	}

	return DefaultCurator.AnalyzeConversation(ctx, req)
}
