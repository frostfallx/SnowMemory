package curator

import (
	"context"
	"fmt"

	"github.com/vmwin11/snowmemory/internal/database"
	"github.com/vmwin11/snowmemory/models"
)

// Curator 记忆整理子代理
type Curator struct {
	Config     *CuratorConfig
	LLMClient  *LLMClient
	repositories repositories
}

// repositories 封装所有 repository
type repositories struct {
	User    *database.UserRepository
	Alias   *database.AliasRepository
	Fact    *database.FactRepository
}

// NewCurator 创建新的记忆整理子代理
func NewCurator(config *CuratorConfig) *Curator {
	return &Curator{
		Config:     config,
		LLMClient:  NewLLMClient(config),
		repositories: repositories{
			User:  &database.UserRepository{},
			Alias: &database.AliasRepository{},
			Fact:  &database.FactRepository{},
		},
	}
}

// AnalyzeConversation 分析对话并自动记忆
func (c *Curator) AnalyzeConversation(ctx context.Context, req models.AnalyzeConversationRequest) (*models.AnalyzeConversationResponse, error) {
	var response models.AnalyzeConversationResponse

	// Step 1: 获取用户现有数据
	facts, err := c.repositories.Fact.ListUserFacts(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user facts: %w", err)
	}

	aliases, err := c.repositories.Alias.GetUserAliases(req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user aliases: %w", err)
	}

	// Step 2: 调用 LLM 分析对话
	actions, err := c.LLMClient.Analyze(ctx, SystemPrompt, facts, aliases, req.ConversationText)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze conversation: %w", err)
	}

	// Step 3: 执行动作
	response.Actions = actions
	response.Results = make([]models.ActionResult, 0, len(actions))

	for _, action := range actions {
		result := models.ActionResult{Action: action.Action}

		switch action.Action {
		case "create_fact":
			if err := c.executeCreateFact(action); err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("create_fact failed: %v", err)
			} else {
				result.Success = true
				result.Message = "fact created successfully"
			}
		case "update_fact":
			if err := c.executeUpdateFact(action); err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("update_fact failed: %v", err)
			} else {
				result.Success = true
				result.Message = "fact updated successfully"
			}
		case "learn_alias":
			if err := c.executeLearnAlias(action); err != nil {
				result.Success = false
				result.Message = fmt.Sprintf("learn_alias failed: %v", err)
			} else {
				result.Success = true
				result.Message = "alias learned successfully"
			}
		case "noop":
			result.Success = true
			result.Message = "no action needed: " + action.Reason
		default:
			result.Success = false
			result.Message = fmt.Sprintf("unknown action type: %s", action.Action)
		}

		response.Results = append(response.Results, result)
	}

	// Step 4: 生成总结
	successCount := 0
	for _, r := range response.Results {
		if r.Success {
			successCount++
		}
	}
	response.Summary = fmt.Sprintf("Processed %d actions, %d successful", len(actions), successCount)

	return &response, nil
}

// executeCreateFact 执行创建事实
func (c *Curator) executeCreateFact(action models.CuratorAction) error {
	if action.UserID == "" {
		return fmt.Errorf("user_id is required for create_fact")
	}
	if action.FactText == "" {
		return fmt.Errorf("fact_text is required for create_fact")
	}

	// 确保用户存在
	user, err := c.repositories.User.GetUserByID(action.UserID)
	if err != nil {
		return fmt.Errorf("failed to check user: %w", err)
	}
	if user == nil {
		if err := c.repositories.User.CreateUser(action.UserID); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	fact := &models.LongTermFact{
		UserID:   action.UserID,
		Category: action.Category,
		FactText: action.FactText,
	}

	return c.repositories.Fact.CreateFact(fact)
}

// executeUpdateFact 执行更新事实
func (c *Curator) executeUpdateFact(action models.CuratorAction) error {
	if action.FactID == 0 {
		return fmt.Errorf("fact_id is required for update_fact")
	}
	if action.FactText == "" {
		return fmt.Errorf("fact_text is required for update_fact")
	}

	return c.repositories.Fact.UpdateFact(action.FactID, action.FactText)
}

// executeLearnAlias 执行学习别名
func (c *Curator) executeLearnAlias(action models.CuratorAction) error {
	if action.UserID == "" {
		return fmt.Errorf("user_id is required for learn_alias")
	}
	if action.GroupID == "" {
		return fmt.Errorf("group_id is required for learn_alias")
	}
	if action.CalledName == "" {
		return fmt.Errorf("called_name is required for learn_alias")
	}

	alias := &models.UserAlias{
		UserID:     action.UserID,
		GroupID:    action.GroupID,
		CalledName: action.CalledName,
	}

	return c.repositories.Alias.UpsertAlias(alias)
}
