package curator

// Action types for curator actions
type (
	// CuratorAction 记忆整理动作
	CuratorAction = struct {
		Action     string `json:"action"`     // create_fact | update_fact | learn_alias | noop
		UserID     string `json:"user_id,omitempty"`
		FactID     int    `json:"fact_id,omitempty"`
		Category   string `json:"category,omitempty"`
		FactText   string `json:"fact_text,omitempty"`
		GroupID    string `json:"group_id,omitempty"`
		CalledName string `json:"called_name,omitempty"`
		Reason     string `json:"reason,omitempty"`
	}

	// ActionResult 动作执行结果
	ActionResult = struct {
		Action  string `json:"action"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	// AnalyzeConversationRequest 分析对话请求
	AnalyzeConversationRequest = struct {
		UserID           string `json:"user_id"`
		GroupID          string `json:"group_id,omitempty"`
		ConversationText string `json:"conversation_text"`
	}

	// AnalyzeConversationResponse 分析对话响应
	AnalyzeConversationResponse = struct {
		Actions []CuratorAction `json:"actions"`
		Results []ActionResult  `json:"results"`
		Summary string          `json:"summary"`
	}
)
