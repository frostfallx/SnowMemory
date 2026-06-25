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
	User      *User         `json:"user"`
	Aliases   []UserAlias   `json:"aliases"`
	Facts     []LongTermFact `json:"facts"`
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
