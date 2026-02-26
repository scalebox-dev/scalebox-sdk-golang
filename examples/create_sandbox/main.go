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
	createReq := models.CreateSandboxRequest{
		Name:      "示例沙箱",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 10,
		Timeout:   300,
		Metadata: map[string]string{
			"environment": "development",
			"team":        "backend",
		},
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()
	fmt.Printf("创建成功! Sandbox ID: %s, 状态: %s\n", sandbox.SandboxID, sandbox.Status)

	// 获取沙箱详情
	fmt.Println("\n=== 获取沙箱详情 ===")
	sandbox, err = sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		log.Fatalf("获取沙箱详情失败: %v", err)
	}
	fmt.Printf("沙箱名称: %s\n", sandbox.Name)
	fmt.Printf("状态: %s\n", sandbox.Status)
	fmt.Printf("CPU: %d 核心\n", sandbox.CPUCount)
	fmt.Printf("内存: %d MB\n", sandbox.MemoryMB)

	// 列出沙箱
	fmt.Println("\n=== 列出沙箱 ===")
	result, err := sandboxClient.List(ctx, &models.ListSandboxesOptions{Status: "running", Limit: 10})
	if err != nil {
		log.Fatalf("列出沙箱失败: %v", err)
	}
	fmt.Printf("找到 %d 个沙箱\n", len(result.Sandboxes))
	for _, sb := range result.Sandboxes {
		fmt.Printf("  - %s: %s (%s)\n", sb.SandboxID, sb.Name, sb.Status)
	}

	// 获取沙箱状态（轻量级）
	fmt.Println("\n=== 获取沙箱状态 ===")
	status, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		log.Fatalf("获取状态失败: %v", err)
	}
	fmt.Printf("状态: %s\n", status.Status)
	if status.Substatus != nil {
		fmt.Printf("子状态: %s\n", *status.Substatus)
	}

	// 设置超时
	fmt.Println("\n=== 设置超时 ===")
	timeoutReq := models.SandboxTimeoutRequest{Timeout: 600}
	updatedSandbox, err := sandboxClient.SetTimeout(ctx, sandbox.SandboxID, timeoutReq)
	if err != nil {
		log.Printf("设置超时失败: %v", err)
	} else {
		fmt.Printf("超时时间已更新: %d 秒\n", updatedSandbox.Timeout)
	}

	// 错误处理示例
	fmt.Println("\n=== 错误处理示例 ===")
	_, err = sandboxClient.Get(ctx, "nonexistent-sandbox-id")
	if err != nil {
		if client.IsNotFound(err) {
			fmt.Println("沙箱不存在 (404)")
		} else if client.IsUnauthorized(err) {
			fmt.Println("未授权 (401)")
		} else if apiErr, ok := err.(*client.APIError); ok {
			fmt.Printf("API 错误 (状态码 %d): %s\n", apiErr.StatusCode, apiErr.Message)
		} else {
			fmt.Printf("其他错误: %v\n", err)
		}
	}

	// 获取指标（可选，沙箱需 running）
	fmt.Println("\n=== 获取沙箱指标 ===")
	start := time.Now().Add(-5 * time.Minute)
	end := time.Now()
	step := 5
	metricsOpts := &models.GetSandboxMetricsOptions{Start: &start, End: &end, Step: &step}
	metrics, err := sandboxClient.GetMetrics(ctx, sandbox.SandboxID, metricsOpts)
	if err != nil {
		log.Printf("获取指标失败: %v (可能沙箱尚未运行)", err)
	} else {
		fmt.Printf("指标数据点数量: %d\n", len(metrics.Metrics))
		if len(metrics.Metrics) > 0 {
			latest := metrics.Metrics[len(metrics.Metrics)-1]
			fmt.Printf("最新 CPU 使用率: %.2f%%\n", latest.CPUUsedPct)
		}
	}
}
