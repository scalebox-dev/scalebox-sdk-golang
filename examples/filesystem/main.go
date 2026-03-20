package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

func main() {
	baseURL := "https://api.scalebox.com"
	apiKey := "your-api-key-here"

	baseClient := client.NewClient(baseURL, apiKey)
	sandboxClient := sandboxes.NewClient(baseClient)
	ctx := context.Background()

	// 创建沙箱
	fmt.Println("=== 创建沙箱 ===")
	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      "filesystem-demo",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 10,
	})
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()
	fmt.Printf("创建成功: %s\n", sandbox.SandboxID)

	// 等待沙箱 running（简化示例，生产环境建议轮询 GetStatus）
	fmt.Println("等待沙箱就绪...")
	time.Sleep(30 * time.Second)

	remotePath := "/workspace/sdk-demo.txt"
	content := []byte("hello from Go SDK filesystem example")

	// Write
	fmt.Println("\n=== 写入文件 ===")
	if err := sandboxClient.Write(ctx, sandbox.SandboxID, remotePath, content); err != nil {
		if client.StatusCode(err) == 400 {
			log.Println("沙箱可能不允许互联网访问或域名不可用，跳过文件操作")
			return
		}
		log.Fatalf("Write 失败: %v", err)
	}
	fmt.Printf("已写入 %s\n", remotePath)

	// Read
	fmt.Println("\n=== 读取文件 ===")
	data, err := sandboxClient.Read(ctx, sandbox.SandboxID, remotePath)
	if err != nil {
		log.Fatalf("Read 失败: %v", err)
	}
	fmt.Printf("内容: %s\n", string(data))

	// Stat
	fmt.Println("\n=== 获取文件信息 ===")
	info, err := sandboxClient.GetInfo(ctx, sandbox.SandboxID, remotePath)
	if err != nil {
		log.Fatalf("GetInfo 失败: %v", err)
	}
	fmt.Printf("名称: %s, 路径: %s, 大小: %d\n", info.Name, info.Path, info.Size.Int64())

	// ListFiles
	fmt.Println("\n=== 列出目录 ===")
	listResult, err := sandboxClient.ListFiles(ctx, sandbox.SandboxID, "/workspace", 1, nil)
	if err != nil {
		log.Fatalf("ListFiles 失败: %v", err)
	}
	fmt.Printf("目录项数量: %d\n", len(listResult.Entries))
	for _, e := range listResult.Entries {
		fmt.Printf("  - %s (%s) size=%d\n", e.Name, e.Path, e.Size.Int64())
	}
}
