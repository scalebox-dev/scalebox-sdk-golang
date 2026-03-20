//go:build e2e

package e2e

import (
	"fmt"
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
	envLoaded := false
	for _, p := range tryPaths {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				fmt.Printf("Warning: failed to load .env from %s: %v\n", p, err)
			} else {
				envLoaded = true
			}
			break
		}
	}
	if !envLoaded {
		fmt.Println("Error: .env file not found. Please copy .env.example to .env and configure it.")
		os.Exit(1)
	}
}
