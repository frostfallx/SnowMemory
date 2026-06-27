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
		var cid int
		var name string
		var colType string
		var notnull int
		var pk int
		var defaultValue any

		if err := rows.Scan(&cid, &name, &colType, &notnull, &pk, &defaultValue); err != nil {
			continue
		}
		if name == "is_common" {
			columnExists = true
		}
	}

	if !columnExists {
		// 添加 is_common 字段
		if _, err := db.Exec("ALTER TABLE long_term_facts ADD COLUMN is_common INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("failed to add is_common column: %w", err)
		}
	}

	// 检查 updated_at 字段是否存在
	rows2, err := db.Query("PRAGMA table_info(long_term_facts)")
	if err != nil {
		return err
	}
	defer rows2.Close()

	updatedAtExists := false
	for rows2.Next() {
		var cid int
		var name string
		var colType string
		var notnull int
		var pk int
		var defaultValue any

		if err := rows2.Scan(&cid, &name, &colType, &notnull, &pk, &defaultValue); err != nil {
			continue
		}
		if name == "updated_at" {
			updatedAtExists = true
		}
	}

	if !updatedAtExists {
		// 添加 updated_at 字段 (SQLite 限制：不能直接添加带默认值的列)
		if _, err := db.Exec("ALTER TABLE long_term_facts ADD COLUMN updated_at TIMESTAMP"); err != nil {
			return fmt.Errorf("failed to add updated_at column: %w", err)
		}
		// 设置触发器自动更新
		if _, err := db.Exec(`
			CREATE TRIGGER IF NOT EXISTS trigger_long_term_facts_updated_at
			AFTER UPDATE ON long_term_facts
			FOR EACH ROW
			BEGIN
				UPDATE long_term_facts SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
			END;`); err != nil {
			return fmt.Errorf("failed to create trigger: %w", err)
		}
		// 回填现有数据的 updated_at
		if _, err := db.Exec("UPDATE long_term_facts SET updated_at = created_at WHERE updated_at IS NULL"); err != nil {
			return fmt.Errorf("failed to backfill updated_at: %w", err)
		}
	}

	return nil
}
