package curator

import (
	"os"
)

// CuratorConfig 记忆整理子代理的配置
type CuratorConfig struct {
	LLMEndpoint string // LLM API 端点
	LLMAPIKey   string // LLM API 密钥
	LLMModel    string // LLM 模型名称
}

// LoadConfig 从环境变量加载配置
func LoadConfig() *CuratorConfig {
	return &CuratorConfig{
		LLMEndpoint: getEnv("SNOWMEMORY_LLM_ENDPOINT", "http://localhost:11434/v1/chat/completions"),
		LLMAPIKey:   getEnv("SNOWMEMORY_LLM_API_KEY", "ollama"),
		LLMModel:    getEnv("SNOWMEMORY_LLM_MODEL", "qwen2.5:7b"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
