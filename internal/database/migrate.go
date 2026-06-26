package database

import (
	"fmt"
)

// Migrate 数据库迁移
func Migrate() error {
	// 检查 is_common 字段是否存在
	rows, err := db.Query("PRAGMA table_info(long_term_facts)")
	if err != nil {
		return err
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var info columnInfo
		if err := rows.Scan(&info); err == nil && info.Name == "is_common" {
			columnExists = true
		}
	}

	if !columnExists {
		// 添加 is_common 字段
		if _, err := db.Exec("ALTER TABLE long_term_facts ADD COLUMN is_common INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add is_common column: %w", err)
		}
	}

	return nil
}

type columnInfo struct {
	CID        int
	Name       string
	Type       string
	NotNull    int
	PrimaryKey int
	Default    any
}
