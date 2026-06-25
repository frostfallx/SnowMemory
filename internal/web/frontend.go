package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/index.html
var frontendContent embed.FS

// ServeFrontend 注册前端页面路由
func ServeFrontend(router *gin.Engine) {
	// 从 embed 文件系统读取 index.html
	sub, err := fs.Sub(frontendContent, "frontend")
	if err != nil {
		panic("failed to read embedded frontend: " + err.Error())
	}

	router.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/index.html")
	})

	router.GET("/index.html", func(c *gin.Context) {
		file, err := sub.Open("index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer file.Close()

		// 读取内容到内存以便多次读取
		content, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Data(http.StatusOK, "text/html; charset=utf-8", content)
	})
}

// serviceStartTime 服务启动时间
var serviceStartTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
