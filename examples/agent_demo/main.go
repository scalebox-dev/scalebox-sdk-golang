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

	// 创建沙箱（需 allow_internet_access 才能使用 Agent）
	fmt.Println("=== 创建沙箱 ===")
	allowInternet := true
	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:                "agent-demo",
		Template:            "base",
		CPUCount:            2,
		MemoryMB:            512,
		StorageGB:           10,
		AllowInternetAccess: &allowInternet,
	})
	if err != nil {
		log.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()
	fmt.Printf("创建成功: %s\n", sandbox.SandboxID)

	// 等待沙箱 running
	fmt.Println("等待沙箱就绪...")
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		status, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			log.Fatalf("获取状态失败: %v", err)
		}
		if status.Status == "running" {
			break
		}
		if status.Status == "failed" || status.Status == "terminated" {
			log.Fatalf("沙箱状态异常: %s", status.Status)
		}
		time.Sleep(3 * time.Second)
	}

	// 直连 Agent
	fmt.Println("\n=== 连接 Agent 执行命令 ===")
	agent, err := sandboxClient.ConnectToAgent(ctx, sandbox.SandboxID)
	if err != nil {
		log.Fatalf("ConnectToAgent 失败: %v (沙箱需 running 且 allow_internet_access=true)", err)
	}

	result, err := agent.Commands().Run(ctx, "echo hello from agent", nil)
	if err != nil {
		log.Fatalf("Commands.Run 失败: %v", err)
	}
	if r, ok := result.(*sandboxes.CommandResult); ok {
		fmt.Printf("命令输出: %s\n", string(r.Stdout))
	}

	// Code Interpreter（若模板支持）
	fmt.Println("\n=== Code Interpreter ===")
	ctxObj, err := agent.CodeInterpreter().CreateContext(ctx, &sandboxes.CreateContextOptions{Language: "python", Cwd: "/home/user"})
	if err != nil {
		log.Printf("CreateContext 失败: %v (可能模板不支持)", err)
		return
	}
	defer func() { _ = agent.CodeInterpreter().DestroyContext(ctx, ctxObj.ID) }()

	execResult, err := agent.CodeInterpreter().RunCode(ctx, ctxObj.ID, "print(1+1)", nil)
	if err != nil {
		log.Fatalf("RunCode 失败: %v", err)
	}
	fmt.Printf("执行结果: exit=%d, stdout=%q\n", execResult.ExitCode, string(execResult.Stdout))
}
