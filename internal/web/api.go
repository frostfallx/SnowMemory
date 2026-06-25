package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/vmwin11/snowmemory/internal/database"
	"github.com/vmwin11/snowmemory/models"
)

// APIServer Web API 服务器
type APIServer struct {
	engine *gin.Engine
}

// NewAPIServer 创建新的 API 服务器
func NewAPIServer() *APIServer {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// 中间件
	engine.Use(gin.Recovery())

	// 前端页面
	ServeFrontend(engine)

	// API 路由组
	api := engine.Group("/api")
	{
		// 用户管理
		api.GET("/users", listUsers)
		api.GET("/users/:user_id", getUser)
		api.POST("/users", createUser)
		api.PUT("/users/:user_id", updateUser)
		api.DELETE("/users/:user_id", deleteUser)

		// 别名管理
		api.GET("/aliases", listAliases)
		api.GET("/aliases/:id", getAlias)
		api.POST("/aliases", createAlias)
		api.PUT("/aliases/:id", updateAlias)
		api.DELETE("/aliases/:id", deleteAlias)

		// 事实管理
		api.GET("/facts", listFacts)
		api.GET("/facts/:id", getFact)
		api.POST("/facts", createFact)
		api.PUT("/facts/:id", updateFact)
		api.DELETE("/facts/:id", deleteFact)

		// 数据导出
		api.GET("/export", exportData)
		// 数据导入
		api.POST("/import", importData)
	}

	return &APIServer{engine: engine}
}

// Start 启动服务器
func (s *APIServer) Start(port string) error {
	return s.engine.Run(":" + port)
}

// healthz 健康检查端点
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "snowmemory",
	})
}

// listUsers 获取所有用户（支持分页）
func listUsers(c *gin.Context) {
	page := getQueryParam(c, "page", "1")
	pageSize := getQueryParam(c, "page_size", "10")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	pageSz, err := strconv.Atoi(pageSize)
	if err != nil || pageSz < 1 || pageSz > 100 {
		pageSz = 10
	}

	offset := (pageNum - 1) * pageSz

	userRepo := &database.UserRepository{}
	users, err := userRepo.ListAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if users == nil {
		users = []models.User{}
	}

	// 分页
	total := len(users)
	start := offset
	if start > total {
		start = total
	}
	end := start + pageSz
	if end > total {
		end = total
	}
	pagedUsers := users[start:end]

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pagedUsers,
		"pagination": gin.H{
			"total":     total,
			"page":      pageNum,
			"page_size": pageSz,
			"total_pages": (total + pageSz - 1) / pageSz,
		},
	})
}

// getQueryParam 获取查询参数，提供默认值
func getQueryParam(c *gin.Context, key, defaultValue string) string {
	value := c.Query(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getUser 获取单个用户
func getUser(c *gin.Context) {
	userID := c.Param("user_id")
	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
}

// createUser 创建用户
func createUser(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRepo := &database.UserRepository{}
	user, _ := userRepo.GetUserByID(req.UserID)
	if user != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	if err := userRepo.CreateUser(req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": map[string]string{"user_id": req.UserID}})
}

// updateUser 更新用户
func updateUser(c *gin.Context) {
	userID := c.Param("user_id")
	var req struct {
		Notes string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := userRepo.UpdateUserNotes(userID, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]string{"user_id": userID}})
}

// deleteUser 删除用户
func deleteUser(c *gin.Context) {
	userID := c.Param("user_id")

	userRepo := &database.UserRepository{}
	user, err := userRepo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := deleteUserCascade(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]string{"user_id": userID}})
}

// deleteUserCascade 级联删除用户及其关联数据
func deleteUserCascade(userID string) error {
	factRepo := &database.FactRepository{}
	if err := factRepo.DeleteFactsByUser(userID); err != nil {
		return err
	}

	aliasRepo := &database.AliasRepository{}
	if err := aliasRepo.DeleteAllAliasesByUser(userID); err != nil {
		return err
	}

	userRepo := &database.UserRepository{}
	return userRepo.DeleteUser(userID)
}

// listAliases 获取所有别名（支持分页）
func listAliases(c *gin.Context) {
	page := getQueryParam(c, "page", "1")
	pageSize := getQueryParam(c, "page_size", "10")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	pageSz, err := strconv.Atoi(pageSize)
	if err != nil || pageSz < 1 || pageSz > 100 {
		pageSz = 10
	}

	offset := (pageNum - 1) * pageSz

	query := `SELECT id, user_id, group_id, called_name, created_at, updated_at FROM user_aliases ORDER BY updated_at DESC`
	rows, err := database.GetDB().Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var aliases []models.UserAlias
	for rows.Next() {
		var a models.UserAlias
		if err := rows.Scan(&a.ID, &a.UserID, &a.GroupID, &a.CalledName, &a.CreatedAt, &a.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		aliases = append(aliases, a)
	}
	if aliases == nil {
		aliases = []models.UserAlias{}
	}

	// 分页
	total := len(aliases)
	start := offset
	if start > total {
		start = total
	}
	end := start + pageSz
	if end > total {
		end = total
	}
	pagedAliases := aliases[start:end]

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pagedAliases,
		"pagination": gin.H{
			"total":     total,
			"page":      pageNum,
			"page_size": pageSz,
			"total_pages": (total + pageSz - 1) / pageSz,
		},
	})
}

// getAlias 获取单个别名
func getAlias(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	query := `SELECT id, user_id, group_id, called_name, created_at, updated_at FROM user_aliases WHERE id = ?`
	var a models.UserAlias
	err = database.GetDB().QueryRow(query, id).Scan(&a.ID, &a.UserID, &a.GroupID, &a.CalledName, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alias not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": a})
}

// createAlias 创建别名
func createAlias(c *gin.Context) {
	var req models.UserAlias
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.UserID == "" || req.GroupID == "" || req.CalledName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id, group_id, and called_name are required"})
		return
	}

	aliasRepo := &database.AliasRepository{}
	if err := aliasRepo.UpsertAlias(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// updateAlias 更新别名
func updateAlias(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.UserAlias
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `UPDATE user_aliases SET user_id = ?, group_id = ?, called_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	result, err := database.GetDB().Exec(query, req.UserID, req.GroupID, req.CalledName, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alias not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]int{"id": id}})
}

// deleteAlias 删除别名
func deleteAlias(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	aliasRepo := &database.AliasRepository{}
	if err := aliasRepo.DeleteAlias(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]int{"id": id}})
}

// listFacts 获取所有长期事实（支持分页）
func listFacts(c *gin.Context) {
	page := getQueryParam(c, "page", "1")
	pageSize := getQueryParam(c, "page_size", "10")

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}
	pageSz, err := strconv.Atoi(pageSize)
	if err != nil || pageSz < 1 || pageSz > 100 {
		pageSz = 10
	}

	offset := (pageNum - 1) * pageSz

	factRepo := &database.FactRepository{}
	facts, err := factRepo.ListAllFacts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if facts == nil {
		facts = []models.LongTermFact{}
	}

	// 分页
	total := len(facts)
	start := offset
	if start > total {
		start = total
	}
	end := start + pageSz
	if end > total {
		end = total
	}
	pagedFacts := facts[start:end]

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    pagedFacts,
		"pagination": gin.H{
			"total":     total,
			"page":      pageNum,
			"page_size": pageSz,
			"total_pages": (total + pageSz - 1) / pageSz,
		},
	})
}

// getFact 获取单个长期事实
func getFact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	query := `SELECT id, user_id, category, fact_text, created_at FROM long_term_facts WHERE id = ?`
	var f models.LongTermFact
	err = database.GetDB().QueryRow(query, id).Scan(&f.ID, &f.UserID, &f.Category, &f.FactText, &f.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "fact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": f})
}

// createFact 创建长期事实
func createFact(c *gin.Context) {
	var req models.LongTermFact
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.UserID == "" || req.Category == "" || req.FactText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id, category, and fact_text are required"})
		return
	}

	factRepo := &database.FactRepository{}
	if err := factRepo.CreateFact(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": req})
}

// updateFact 更新长期事实
func updateFact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.LongTermFact
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `UPDATE long_term_facts SET user_id = ?, category = ?, fact_text = ? WHERE id = ?`
	result, err := database.GetDB().Exec(query, req.UserID, req.Category, req.FactText, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "fact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]int{"id": id}})
}

// deleteFact 删除长期事实
func deleteFact(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	factRepo := &database.FactRepository{}
	if err := factRepo.DeleteFact(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": map[string]int{"id": id}})
}

// ExportData 数据导出请求/响应
type ExportData struct {
	Users       []models.User       `json:"users"`
	Aliases     []models.UserAlias  `json:"aliases"`
	Facts       []models.LongTermFact `json:"facts"`
	ExportedAt  string              `json:"exported_at"`
}

// exportData 导出所有数据
func exportData(c *gin.Context) {
	// 获取查询参数决定导出格式
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	userRepo := &database.UserRepository{}
	users, err := userRepo.ListAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if users == nil {
		users = []models.User{}
	}

	aliasRepo := &database.AliasRepository{}
	aliases, err := aliasRepo.GetUserAliases("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if aliases == nil {
		aliases = []models.UserAlias{}
	}

	factRepo := &database.FactRepository{}
	facts, err := factRepo.ListAllFacts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if facts == nil {
		facts = []models.LongTermFact{}
	}

	exportedAt := "now"
	if format == "csv" {
		// 导出为 CSV 格式
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=snowmemory_export.csv")

		// 构建 CSV 内容
		csv := "type,user_id,group_id,called_name,category,fact_text,created_at,updated_at\n"
		for _, u := range users {
			csv += fmt.Sprintf("user,%s,,,,,%s,\n", u.UserID, u.CreatedAt)
		}
		for _, a := range aliases {
			csv += fmt.Sprintf("alias,%s,%s,%s,,,,\n", a.UserID, a.GroupID, a.CalledName)
		}
		for _, f := range facts {
			csv += fmt.Sprintf("fact,%s,,,,%s,%s,\n", f.UserID, f.FactText, f.CreatedAt)
		}
		c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(csv))
	} else {
		// 默认 JSON 格式
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": ExportData{
				Users:      users,
				Aliases:    aliases,
				Facts:      facts,
				ExportedAt: exportedAt,
			},
		})
	}
}

// ImportData 数据导入请求
type ImportData struct {
	Users      []models.User       `json:"users"`
	Aliases    []models.UserAlias  `json:"aliases"`
	Facts      []models.LongTermFact `json:"facts"`
}

// importData 导入数据
func importData(c *gin.Context) {
	var req ImportData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	importedUsers := 0
	importedAliases := 0
	importedFacts := 0

	// 导入用户
	userRepo := &database.UserRepository{}
	for _, user := range req.Users {
		existing, _ := userRepo.GetUserByID(user.UserID)
		if existing == nil {
			if err := userRepo.CreateUser(user.UserID); err == nil {
				importedUsers++
			}
		}
	}

	// 导入别名
	aliasRepo := &database.AliasRepository{}
	for _, alias := range req.Aliases {
		existing, _ := aliasRepo.GetUserAliasInGroup(alias.UserID, alias.GroupID)
		if existing == nil {
			aliasReq := &models.UserAlias{
				UserID:     alias.UserID,
				GroupID:    alias.GroupID,
				CalledName: alias.CalledName,
			}
			if err := aliasRepo.UpsertAlias(aliasReq); err == nil {
				importedAliases++
			}
		}
	}

	// 导入事实
	factRepo := &database.FactRepository{}
	for _, fact := range req.Facts {
		factReq := &models.LongTermFact{
			UserID:   fact.UserID,
			Category: fact.Category,
			FactText: fact.FactText,
		}
		if err := factRepo.CreateFact(factReq); err == nil {
			importedFacts++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"users":      importedUsers,
			"aliases":    importedAliases,
			"facts":      importedFacts,
		},
	})
}

