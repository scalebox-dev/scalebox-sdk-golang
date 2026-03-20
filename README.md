# Scalebox Go SDK

Scalebox Go SDK 提供与 Scalebox API 交互的 Go 客户端库，采用 **REST + gRPC/Connect 双通道** 架构：

- **REST**：沙箱 CRUD、文件操作、端口管理、批量操作（经 Backend）
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

## SDK 功能

### 沙箱管理
- 创建、获取、列表、删除沙箱
- 暂停、恢复沙箱（支持 `BatchPause` / `BatchResume` 批量操作）
- 设置超时、端口管理

### 文件操作
- 读写文件（`Write` / `Read`）
- 批量写入（`WriteBatch` / `WriteBatchConcurrent`）
- 文件列表、Stat

### Agent 服务
- 命令执行（Commands）
- PTY 会话
- Code Interpreter
- 目录监控（watch_dir）

### 模板管理
- 导出沙箱为模板
- 导入外部镜像模板
- 模板 CRUD 操作

## 文档

- [API 文档](docs/api.md) — 完整 API 用法与示例
- [测试说明](docs/testing.md) — 单元测试与集成测试
- [E2E 测试](e2e_test/README.md) — E2E 测试说明
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
