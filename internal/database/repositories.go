package database

import (
	"database/sql"

	"github.com/vmwin11/snowmemory/models"
)

// UserRepository 用户数据访问层
type UserRepository struct{}

// NewUserRepository 创建用户仓库
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// GetUserByID 根据 ID 获取用户
func (r *UserRepository) GetUserByID(userID string) (*models.User, error) {
	query := `SELECT user_id, created_at, updated_at, notes FROM users WHERE user_id = ?`
	var user models.User
	err := GetDB().QueryRow(query, userID).Scan(
		&user.UserID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.Notes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建新用户
func (r *UserRepository) CreateUser(userID string) error {
	query := `INSERT INTO users (user_id) VALUES (?)`
	_, err := GetDB().Exec(query, userID)
	return err
}

// UpdateUserNotes 更新用户备注
func (r *UserRepository) UpdateUserNotes(userID, notes string) error {
	query := `UPDATE users SET notes = ?, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`
	_, err := GetDB().Exec(query, notes, userID)
	return err
}

// DeleteUser 删除用户
func (r *UserRepository) DeleteUser(userID string) error {
	query := `DELETE FROM users WHERE user_id = ?`
	_, err := GetDB().Exec(query, userID)
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
		if err := rows.Scan(&user.UserID, &user.CreatedAt, &user.UpdatedAt, &user.Notes); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// AliasRepository 用户别名数据访问层
type AliasRepository struct{}

// GetUserAliases 获取用户在所有群的别名
func (r *AliasRepository) GetUserAliases(userID string) ([]models.UserAlias, error) {
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
	return aliases, rows.Err()
}

// GetUserAliasInGroup 获取用户在特定群的别名
func (r *AliasRepository) GetUserAliasInGroup(userID, groupID string) (*models.UserAlias, error) {
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
	return &alias, nil
}

// UpsertAlias 插入或更新别名
func (r *AliasRepository) UpsertAlias(alias *models.UserAlias) error {
	existing, err := r.GetUserAliasInGroup(alias.UserID, alias.GroupID)
	if err != nil {
		return err
	}

	if existing == nil {
		query := `INSERT INTO user_aliases (user_id, group_id, called_name) VALUES (?, ?, ?)`
		_, err := GetDB().Exec(query, alias.UserID, alias.GroupID, alias.CalledName)
		return err
	}

	query := `UPDATE user_aliases SET called_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = GetDB().Exec(query, alias.CalledName, existing.ID)
	return err
}

// DeleteAlias 删除别名
func (r *AliasRepository) DeleteAlias(aliasID int) error {
	query := `DELETE FROM user_aliases WHERE id = ?`
	_, err := GetDB().Exec(query, aliasID)
	return err
}

// DeleteAliasByUserAndGroup 删除用户在特定群的别名
func (r *AliasRepository) DeleteAliasByUserAndGroup(userID, groupID string) error {
	query := `DELETE FROM user_aliases WHERE user_id = ? AND group_id = ?`
	_, err := GetDB().Exec(query, userID, groupID)
	return err
}

// DeleteAllAliasesByUser 删除用户的所有别名
func (r *AliasRepository) DeleteAllAliasesByUser(userID string) error {
	query := `DELETE FROM user_aliases WHERE user_id = ?`
	_, err := GetDB().Exec(query, userID)
	return err
}

// FactRepository 长期事实数据访问层
type FactRepository struct{}

// GetUserFacts 获取用户的所有长期事实
func (r *FactRepository) GetUserFacts(userID string) ([]models.LongTermFact, error) {
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
	return facts, rows.Err()
}

// GetUserFactsByCategory 获取用户指定分类的长期事实
func (r *FactRepository) GetUserFactsByCategory(userID, category string) ([]models.LongTermFact, error) {
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
	return facts, rows.Err()
}

// CreateFact 创建长期事实
func (r *FactRepository) CreateFact(fact *models.LongTermFact) error {
	query := `INSERT INTO long_term_facts (user_id, category, fact_text) VALUES (?, ?, ?)`
	_, err := GetDB().Exec(query, fact.UserID, fact.Category, fact.FactText)
	return err
}

// DeleteFact 删除长期事实
func (r *FactRepository) DeleteFact(factID int) error {
	query := `DELETE FROM long_term_facts WHERE id = ?`
	_, err := GetDB().Exec(query, factID)
	return err
}

// DeleteFactsByUser 删除用户的所有长期事实
func (r *FactRepository) DeleteFactsByUser(userID string) error {
	query := `DELETE FROM long_term_facts WHERE user_id = ?`
	_, err := GetDB().Exec(query, userID)
	return err
}

// DeleteFactsByCategory 删除用户指定分类的长期事实
func (r *FactRepository) DeleteFactsByCategory(userID, category string) error {
	query := `DELETE FROM long_term_facts WHERE user_id = ? AND category = ?`
	_, err := GetDB().Exec(query, userID, category)
	return err
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
