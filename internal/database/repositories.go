package database

import (
	"database/sql"
	"fmt"

	"github.com/vmwin11/snowmemory/models"
	"github.com/vmwin11/snowmemory/utils"
)

// UserRepository 用户数据访问层
type UserRepository struct{}

// NewUserRepository 创建用户仓库
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// GetUserByID 根据 ID 获取用户（带缓存）
func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	// 尝试从缓存读取
	if cache := getUserCache(); cache != nil {
		if data, ok := cache.Get("user:" + userID); ok {
			if user, ok := data.(*models.User); ok {
				return user, nil
			}
		}
	}

	query := `SELECT user_id, created_at, updated_at, notes FROM users WHERE user_id = ?`
	var user models.User
	var notes sql.NullString
	err := GetDB().QueryRow(query, userID).Scan(
		&user.UserID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&notes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// 处理 NULL notes 字段
	if notes.Valid {
		user.Notes = notes.String
	} else {
		user.Notes = ""
	}

	// 写入缓存
	if cache := getUserCache(); cache != nil {
		cache.Set("user:" + userID, &user)
	}

	return &user, nil
}

// CreateUser 创建新用户
func (r *UserRepository) CreateUser(userID string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO users (user_id) VALUES (?)`
	_, err = tx.Exec(query, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("CREATE_USER", userID, "创建新用户")

	// 清除缓存
	InvalidateUserCache(userID)
	return nil
}

// UpdateUserNotes 更新用户备注
func (r *UserRepository) UpdateUserNotes(userID, notes string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `UPDATE users SET notes = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`
	_, err = tx.Exec(query, notes, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("UPDATE_USER_NOTES", userID, fmt.Sprintf("更新用户备注: %s", notes))

	// 清除缓存
	InvalidateUserCache(userID)
	return nil
}

// DeleteUser 删除用户
func (r *UserRepository) DeleteUser(userID string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除关联数据
	if err := r.deleteUserAliasesLocked(tx, userID); err != nil {
		return err
	}
	if err := r.deleteUserFactsLocked(tx, userID); err != nil {
		return err
	}

	// 删除用户
	query := `DELETE FROM users WHERE user_id = ?`
	_, err = tx.Exec(query, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("DELETE_USER", userID, "删除用户及其关联数据")

	// 清除缓存
	InvalidateUserCache(userID)
	return nil
}

// deleteUserAliasesLocked 在事务中删除用户别名
func (r *UserRepository) deleteUserAliasesLocked(tx *sql.Tx, userID string) error {
	query := `DELETE FROM user_aliases WHERE user_id = ?`
	_, err := tx.Exec(query, userID)
	return err
}

// deleteUserFactsLocked 在事务中删除用户事实
func (r *UserRepository) deleteUserFactsLocked(tx *sql.Tx, userID string) error {
	query := `DELETE FROM long_term_facts WHERE user_id = ?`
	_, err := tx.Exec(query, userID)
	return err
}

// ListAllUsers 列出所有用户
func (r *UserRepository) ListAllUsers() ([]models.User, error) {
	query := `SELECT user_id, created_at, updated_at, notes FROM users ORDER BY created_at DESC`
	rows, err := GetDB().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var notes sql.NullString
		if err := rows.Scan(&user.UserID, &user.CreatedAt, &user.UpdatedAt, &notes); err != nil {
			return nil, err
		}
		// 处理 NULL notes 字段
		if notes.Valid {
			user.Notes = notes.String
		} else {
			user.Notes = ""
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// AliasRepository 用户别名数据访问层
type AliasRepository struct{}

// GetUserAliases 获取用户在所有群的别名（带缓存）
func (r *AliasRepository) GetUserAliases(userID string) ([]models.UserAlias, error) {
	// 尝试从缓存读取
	if cache := getAliasCache(); cache != nil {
		if data, ok := cache.Get("aliases:" + userID); ok {
			if aliases, ok := data.([]models.UserAlias); ok {
				return aliases, nil
			}
		}
	}

	query := `SELECT id, user_id, group_id, called_name, created_at, updated_at FROM user_aliases WHERE user_id = ? ORDER BY updated_at DESC`
	rows, err := GetDB().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []models.UserAlias
	for rows.Next() {
		var alias models.UserAlias
		err := rows.Scan(&alias.ID, &alias.UserID, &alias.GroupID, &alias.CalledName, &alias.CreatedAt, &alias.UpdatedAt)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, alias)
	}

	// 写入缓存
	if cache := getAliasCache(); cache != nil {
		cache.Set("aliases:" + userID, aliases)
	}

	return aliases, rows.Err()
}

// GetUserAliasInGroup 获取用户在特定群的别名（带缓存）
func (r *AliasRepository) GetUserAliasInGroup(userID, groupID string) (*models.UserAlias, error) {
	cacheKey := fmt.Sprintf("alias:%s:%s", userID, groupID)

	// 尝试从缓存读取
	if cache := getAliasCache(); cache != nil {
		if data, ok := cache.Get(cacheKey); ok {
			if alias, ok := data.(*models.UserAlias); ok {
				return alias, nil
			}
		}
	}

	query := `SELECT id, user_id, group_id, called_name, created_at, updated_at FROM user_aliases WHERE user_id = ? AND group_id = ?`
	var alias models.UserAlias
	err := GetDB().QueryRow(query, userID, groupID).Scan(
		&alias.ID,
		&alias.UserID,
		&alias.GroupID,
		&alias.CalledName,
		&alias.CreatedAt,
		&alias.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 写入缓存
	if cache := getAliasCache(); cache != nil {
		cache.Set(cacheKey, &alias)
	}

	return &alias, nil
}

// UpsertAlias 插入或更新别名（带事务）
func (r *AliasRepository) UpsertAlias(alias *models.UserAlias) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	existing, err := r.GetUserAliasInGroup(alias.UserID, alias.GroupID)
	if err != nil {
		return err
	}

	var operation string
	if existing == nil {
		query := `INSERT INTO user_aliases (user_id, group_id, called_name) VALUES (?, ?, ?)`
		_, err = tx.Exec(query, alias.UserID, alias.GroupID, alias.CalledName)
		operation = "CREATE_ALIAS"
	} else {
		query := `UPDATE user_aliases SET called_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
		_, err = tx.Exec(query, alias.CalledName, existing.ID)
		operation = "UPDATE_ALIAS"
	}
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog(operation, alias.UserID, fmt.Sprintf("群组: %s, 称呼: %s", alias.GroupID, alias.CalledName))

	// 清除缓存
	InvalidateAliasCache(alias.UserID)
	return nil
}

// DeleteAlias 删除别名（带事务）
func (r *AliasRepository) DeleteAlias(aliasID int) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM user_aliases WHERE id = ?`
	result, err := tx.Exec(query, aliasID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志（需要查询别名来获取 user_id）
	var userID string
	err = GetDB().QueryRow(`SELECT user_id FROM user_aliases WHERE id = ?`, aliasID).Scan(&userID)
	if err == nil {
		utils.DefaultLogger.AuditLog("DELETE_ALIAS", userID, fmt.Sprintf("删除别名 ID: %d", aliasID))
		InvalidateAliasCache(userID)
	}
	return nil
}

// DeleteAliasByUserAndGroup 删除用户在特定群的别名（带事务）
func (r *AliasRepository) DeleteAliasByUserAndGroup(userID, groupID string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM user_aliases WHERE user_id = ? AND group_id = ?`
	_, err = tx.Exec(query, userID, groupID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("DELETE_ALIAS", userID, fmt.Sprintf("删除群组 %s 的别名", groupID))

	// 清除缓存
	InvalidateAliasCache(userID)
	return nil
}

// DeleteAllAliasesByUser 删除用户的所有别名（带事务）
func (r *AliasRepository) DeleteAllAliasesByUser(userID string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM user_aliases WHERE user_id = ?`
	_, err = tx.Exec(query, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("DELETE_ALL_ALIASES", userID, "删除用户所有别名")

	// 清除缓存
	InvalidateAliasCache(userID)
	return nil
}

// FactRepository 长期事实数据访问层
type FactRepository struct{}

// GetUserFacts 获取用户的所有长期事实（带缓存）
func (r *FactRepository) GetUserFacts(userID string) ([]models.LongTermFact, error) {
	// 尝试从缓存读取
	if cache := getFactCache(); cache != nil {
		if data, ok := cache.Get("facts:" + userID); ok {
			if facts, ok := data.([]models.LongTermFact); ok {
				return facts, nil
			}
		}
	}

	query := `SELECT id, user_id, category, fact_text, created_at FROM long_term_facts WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := GetDB().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.LongTermFact
	for rows.Next() {
		var fact models.LongTermFact
		err := rows.Scan(&fact.ID, &fact.UserID, &fact.Category, &fact.FactText, &fact.CreatedAt)
		if err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}

	// 写入缓存
	if cache := getFactCache(); cache != nil {
		cache.Set("facts:" + userID, facts)
	}

	return facts, rows.Err()
}

// GetUserFactsByCategory 获取用户指定分类的长期事实（带缓存）
func (r *FactRepository) GetUserFactsByCategory(userID, category string) ([]models.LongTermFact, error) {
	cacheKey := fmt.Sprintf("facts:%s:%s", userID, category)

	// 尝试从缓存读取
	if cache := getFactCache(); cache != nil {
		if data, ok := cache.Get(cacheKey); ok {
			if facts, ok := data.([]models.LongTermFact); ok {
				return facts, nil
			}
		}
	}

	query := `SELECT id, user_id, category, fact_text, created_at FROM long_term_facts WHERE user_id = ? AND category = ? ORDER BY created_at DESC`
	rows, err := GetDB().Query(query, userID, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.LongTermFact
	for rows.Next() {
		var fact models.LongTermFact
		err := rows.Scan(&fact.ID, &fact.UserID, &fact.Category, &fact.FactText, &fact.CreatedAt)
		if err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}

	// 写入缓存
	if cache := getFactCache(); cache != nil {
		cache.Set(cacheKey, facts)
	}

	return facts, rows.Err()
}

// CreateFact 创建长期事实（带事务）
func (r *FactRepository) CreateFact(fact *models.LongTermFact) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO long_term_facts (user_id, category, fact_text) VALUES (?, ?, ?)`
	_, err = tx.Exec(query, fact.UserID, fact.Category, fact.FactText)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("CREATE_FACT", fact.UserID, fmt.Sprintf("分类: %s, 内容: %s", fact.Category, fact.FactText))

	// 清除缓存
	InvalidateFactCache(fact.UserID)
	return nil
}

// DeleteFact 删除长期事实（带事务）
func (r *FactRepository) DeleteFact(factID int) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM long_term_facts WHERE id = ?`
	result, err := tx.Exec(query, factID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志（需要查询事实来获取 user_id）
	var userID string
	err = GetDB().QueryRow(`SELECT user_id FROM long_term_facts WHERE id = ?`, factID).Scan(&userID)
	if err == nil {
		utils.DefaultLogger.AuditLog("DELETE_FACT", userID, fmt.Sprintf("删除事实 ID: %d", factID))
		InvalidateFactCache(userID)
	}
	return nil
}

// DeleteFactsByUser 删除用户的所有长期事实（带事务）
func (r *FactRepository) DeleteFactsByUser(userID string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM long_term_facts WHERE user_id = ?`
	_, err = tx.Exec(query, userID)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("DELETE_ALL_FACTS", userID, "删除用户所有长期事实")

	// 清除缓存
	InvalidateFactCache(userID)
	return nil
}

// DeleteFactsByCategory 删除用户指定分类的长期事实（带事务）
func (r *FactRepository) DeleteFactsByCategory(userID, category string) error {
	// 使用事务
	tx, err := GetDB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `DELETE FROM long_term_facts WHERE user_id = ? AND category = ?`
	_, err = tx.Exec(query, userID, category)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 审计日志
	utils.DefaultLogger.AuditLog("DELETE_FACTS_BY_CATEGORY", userID, fmt.Sprintf("删除分类 %s 的长期事实", category))

	// 清除缓存
	InvalidateFactCache(userID)
	return nil
}

// ListAllFacts 列出所有长期事实
func (r *FactRepository) ListAllFacts() ([]models.LongTermFact, error) {
	query := `SELECT id, user_id, category, fact_text, created_at FROM long_term_facts ORDER BY created_at DESC`
	rows, err := GetDB().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []models.LongTermFact
	for rows.Next() {
		var fact models.LongTermFact
		if err := rows.Scan(&fact.ID, &fact.UserID, &fact.Category, &fact.FactText, &fact.CreatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, fact)
	}
	return facts, rows.Err()
}

// Cache 辅助函数（导出给外部使用）
func GetUserCache() *MemoryCache {
	return userCache
}

func GetAliasCache() *MemoryCache {
	return aliasCache
}

func GetFactCache() *MemoryCache {
	return factCache
}

// 缓存辅助函数
func getUserCache() *MemoryCache {
	return userCache
}

func getAliasCache() *MemoryCache {
	return aliasCache
}

func getFactCache() *MemoryCache {
	return factCache
}
