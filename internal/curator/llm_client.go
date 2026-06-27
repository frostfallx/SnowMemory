package curator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vmwin11/snowmemory/models"
)

// LLMClient LLM API 客户端
type LLMClient struct {
	config *CuratorConfig
	client *http.Client
}

// NewLLMClient 创建 LLM 客户端
func NewLLMClient(config *CuratorConfig) *LLMClient {
	return &LLMClient{
		config: config,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// Analyze 调用 LLM 分析对话
// 发送现有事实、别名和对话文本，返回建议的动作列表
func (c *LLMClient) Analyze(ctx context.Context, systemPrompt string,
	existingFacts []models.LongTermFact,
	existingAliases []models.UserAlias,
	conversationText string) ([]models.CuratorAction, error) {

	// 构造用户消息内容
	userMsg := fmt.Sprintf(`# 已有的记忆

## 用户已有事实
%s

## 用户已有别名
%s

## 刚才的对话
%s

请分析这段对话，决定是否需要更新、新增记忆或学习新的别名。

输出格式要求：
- 返回 JSON 格式，包含一个 "actions" 数组
- 每个动作必须是以下类型之一：
  - "create_fact": 创建新事实，需提供 user_id, category, fact_text
  - "update_fact": 更新已有事实，需提供 fact_id, fact_text (合并后的完整文本)
  - "learn_alias": 学习新的别名，需提供 user_id, group_id, called_name
  - "noop": 无需操作，需提供 reason 说明原因
- 如果对话中没有值得记住的信息，返回空数组 []
- 尽量使用 update_fact 来合并细化信息，而不是 create_fact`,
			formatFactsForPrompt(existingFacts),
			formatAliasesForPrompt(existingAliases),
			conversationText)

	// 构造请求
	reqBody := map[string]interface{}{
		"model":        c.config.LLMModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMsg},
		},
		"temperature": 0.1,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.LLMEndpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.LLMAPIKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// 提取内容
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid choice format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no message in choice")
	}

	content, ok := message["content"].(string)
	if !ok {
		return nil, fmt.Errorf("no content in message")
	}

	// 清理可能的 markdown 包裹
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
	}

	// 解析动作列表
	var actions []models.CuratorAction
	if err := json.Unmarshal([]byte(content), &actions); err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	return actions, nil
}

// formatFactsForPrompt 格式化事实用于 prompt
func formatFactsForPrompt(facts []models.LongTermFact) string {
	if len(facts) == 0 {
		return "(无)"
	}

	var sb strings.Builder
	for _, fact := range facts {
		sb.WriteString(fmt.Sprintf("- [ID: %d] %s: %s\n", fact.ID, fact.Category, fact.FactText))
	}
	return sb.String()
}

// formatAliasesForPrompt 格式化别名用于 prompt
func formatAliasesForPrompt(aliases []models.UserAlias) string {
	if len(aliases) == 0 {
		return "(无)"
	}

	var sb strings.Builder
	for _, alias := range aliases {
		sb.WriteString(fmt.Sprintf("- 群组 %s: %s\n", alias.GroupID, alias.CalledName))
	}
	return sb.String()
}
