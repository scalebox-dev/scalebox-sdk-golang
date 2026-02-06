//go:build integration

package integration

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func init() {
	// 尝试加载 .env，避免每次运行集成测试前手动 source .env
	// 按顺序尝试：integration/.env、当前工作目录 .env
	wd, _ := os.Getwd()
	tryPaths := []string{
		filepath.Join(wd, "integration", ".env"),
		filepath.Join(wd, ".env"),
		".env",
		filepath.Join("integration", ".env"),
	}
	for _, p := range tryPaths {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}
}
