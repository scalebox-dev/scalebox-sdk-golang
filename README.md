# Scalebox Go SDK

Scalebox Go SDK 提供与 Scalebox API 交互的 Go 客户端库，采用 **REST + gRPC/Connect 双通道** 架构：

- **REST**：沙箱 CRUD、文件操作、端口管理（经 Backend）
- **gRPC/Connect**：Commands、PTY、Code Interpreter、watch_dir（直连 Sandbox Agent）

## 安装

```bash
go get github.com/scalebox/scalebox-sdk-golang
```

## 快速开始

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
    "github.com/scalebox/scalebox-sdk-golang/client"
    "github.com/scalebox/scalebox-sdk-golang/models"
)

func main() {
    baseClient := client.NewClient("https://api.scalebox.com", "your-api-key")
    sandboxClient := sandboxes.NewClient(baseClient)

    ctx := context.Background()
    req := models.CreateSandboxRequest{
        Name:      "My Sandbox",
        Template:  "base",
        CPUCount:  2,
        MemoryMB:  512,
        StorageGB: 10,
    }

    sandbox, err := sandboxClient.Create(ctx, req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created sandbox: %s\n", sandbox.SandboxID)
}
```

## 文档

- [API 文档](docs/api.md) — 完整 API 用法与示例
- [测试说明](docs/testing.md) — 单元测试与集成测试
- [贡献指南](docs/CONTRIBUTING.md) — 代码风格、提交流程

## 示例

```bash
# 创建与查询沙箱
go run ./examples/create_sandbox

# 文件操作（Write / Read / ListFiles / Stat）
go run ./examples/filesystem

# 直连 Agent（Commands / Code Interpreter）
go run ./examples/agent_demo
```

## 许可证

本项目采用 [MIT License](LICENSE) 许可证。
