package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var (
	db      *sql.DB
	dbMutex sync.RWMutex
)

// Init 初始化数据库连接
func Init(dbPath string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		return nil
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// 初始化缓存层（默认 TTL 5 分钟）
	InitCache(5 * time.Minute)

	// 初始化表结构
	if err := initSchema(); err != nil {
		return err
	}

	return nil
}

// Close 关闭数据库连接
func Close() error {
	dbMutex.Lock()
	defer dbMutex.Unlock()

	if db != nil {
		return db.Close()
	}
	return nil
}

// GetDB 获取数据库实例
func GetDB() *sql.DB {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return db
}

// initSchema 初始化数据库表结构
func initSchema() error {
	// Users 表 - 存储用户基本信息
	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		user_id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		notes TEXT
	);`

	// UserAliases 表 - 存储用户在不同群的称呼
	aliasesTable := `
	CREATE TABLE IF NOT EXISTS user_aliases (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		group_id TEXT NOT NULL,
		called_name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(user_id),
		UNIQUE(user_id, group_id)
	);`

	// LongTermFacts 表 - 存储长期事实
	factsTable := `
	CREATE TABLE IF NOT EXISTS long_term_facts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT NOT NULL,
		category TEXT NOT NULL,
		fact_text TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(user_id)
	);`

	// 创建索引提升查询性能
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_user_aliases_user_id ON user_aliases(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_user_aliases_group_id ON user_aliases(group_id);",
		"CREATE INDEX IF NOT EXISTS idx_long_term_facts_user_id ON long_term_facts(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_long_term_facts_category ON long_term_facts(category);",
	}

	statements := []string{usersTable, aliasesTable, factsTable}
	statements = append(statements, indexes...)

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}

	return nil
}
