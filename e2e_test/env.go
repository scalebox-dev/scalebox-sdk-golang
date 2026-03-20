//go:build e2e

package e2e

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

func init() {
	// 只从 e2e_test 目录加载 .env 文件
	wd, _ := os.Getwd()
	tryPaths := []string{
		filepath.Join(wd, "e2e_test", ".env"),
		filepath.Join(wd, ".env"),
		".env",
	}
	for _, p := range tryPaths {
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			break
		}
	}
}
