package models

// User 代表一个用户（以 QQ 号为唯一标识）
type User struct {
	UserID    string `json:"user_id" db:"user_id"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
	Notes     string `json:"notes" db:"notes"`
}

// UserAlias 代表用户在特定群的称呼
type UserAlias struct {
	ID        int    `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	GroupID   string `json:"group_id" db:"group_id"`
	CalledName string `json:"called_name" db:"called_name"`
	CreatedAt string `json:"created_at" db:"created_at"`
	UpdatedAt string `json:"updated_at" db:"updated_at"`
}

// LongTermFact 代表用户的长期事实（兴趣、关系等）
type LongTermFact struct {
	ID        int    `json:"id" db:"id"`
	UserID    string `json:"user_id" db:"user_id"`
	Category  string `json:"category" db:"category"`
	FactText  string `json:"fact_text" db:"fact_text"`
	CreatedAt string `json:"created_at" db:"created_at"`
}

// UserProfile 用户完整资料（用于 MCP 工具返回）
type UserProfile struct {
	User        *User            `json:"user"`
	Aliases     []UserAlias      `json:"aliases"`
	Facts       []LongTermFact   `json:"facts"`
	GlobalCalls []string         `json:"global_calls"`        // 全局称呼（所有群的称呼汇总）
	CurrentCall string           `json:"current_call"`        // 当前群称呼
	CallSummary map[string]string `json:"call_summary"`       // 群组称呼映射
}

// AliasRequest 学习用户称呼的请求
type AliasRequest struct {
	UserID     string `json:"user_id"`
	GroupID    string `json:"group_id"`
	CalledName string `json:"called_name"`
}

// FactRequest 存储长期事实的请求
type FactRequest struct {
	UserID   string `json:"user_id"`
	Category string `json:"category"`
	FactText string `json:"fact_text"`
}

// QueryRequest 查询用户资料的请求
type QueryRequest struct {
	UserID string `json:"user_id"`
	GroupID string `json:"group_id"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Users  []User           `json:"users"`
	Aliases []UserAlias      `json:"aliases"`
	Facts   []LongTermFact   `json:"facts"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `json:"keyword"`
}
