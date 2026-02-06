# Scalebox Go SDK

Scalebox Go SDK 提供了与 Scalebox API 交互的 Go 客户端库。

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
    // 创建基础客户端
    baseClient := client.NewClient(
        "https://api.scalebox.com",  // API 基础 URL
        "your-api-key",               // API 密钥
    )

    // 创建 Sandboxes 客户端
    sandboxClient := sandboxes.NewClient(baseClient)

    // 创建沙箱
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

## API 文档

### 创建沙箱

```go
req := models.CreateSandboxRequest{
    Name:                "My Sandbox",
    Description:         "A test sandbox",
    Template:            "base",
    ProjectID:           "proj-xxx", // 可选
    CPUCount:            2,
    MemoryMB:            512,
    StorageGB:           10,
    Timeout:             300, // 可选，默认 300 秒
    AutoPause:           boolPtr(true), // 可选
    EnvVars:             map[string]string{"KEY": "value"},
    Metadata:             map[string]string{"tag": "test"},
    AllowInternetAccess: boolPtr(true), // 可选
}

sandbox, err := sandboxClient.Create(ctx, req)
```

### 列出沙箱

```go
opts := &models.ListSandboxesOptions{
    Status:    "running",
    ProjectID: "proj-xxx",
    Limit:     10,
    Offset:    0,
}

result, err := sandboxClient.List(ctx, opts)
for _, sandbox := range result.Sandboxes {
    fmt.Printf("Sandbox: %s - %s\n", sandbox.SandboxID, sandbox.Status)
}
```

### 获取沙箱详情

```go
sandbox, err := sandboxClient.Get(ctx, "sbx-xxx")
```

### 获取沙箱状态（轻量级）

```go
status, err := sandboxClient.GetStatus(ctx, "sbx-xxx")
fmt.Printf("Status: %s\n", status.Status)
```

### 更新沙箱

```go
req := models.UpdateSandboxRequest{
    Timeout: 600, // 新的超时时间（秒）
}
sandbox, err := sandboxClient.Update(ctx, "sbx-xxx", req)
```

### 删除沙箱

```go
force := true
result, err := sandboxClient.Delete(ctx, "sbx-xxx", &force)
```

### 终止沙箱

```go
force := false
result, err := sandboxClient.Terminate(ctx, "sbx-xxx", &force)
```

### 暂停沙箱

```go
sandbox, err := sandboxClient.Pause(ctx, "sbx-xxx")
```

### 恢复沙箱

```go
sandbox, err := sandboxClient.Resume(ctx, "sbx-xxx")
```

### 连接沙箱

```go
timeout := 600
req := &models.ConnectSandboxRequest{
    Timeout: &timeout, // 可选
}
sandbox, err := sandboxClient.Connect(ctx, "sbx-xxx", req)
```

### 设置超时

```go
req := models.SandboxTimeoutRequest{
    Timeout: 600,
}
sandbox, err := sandboxClient.SetTimeout(ctx, "sbx-xxx", req)
```

### 获取指标

```go
start := time.Now().Add(-5 * time.Minute)
end := time.Now()
step := 5

opts := &models.GetSandboxMetricsOptions{
    Start: &start,
    End:   &end,
    Step:  &step,
}

metrics, err := sandboxClient.GetMetrics(ctx, "sbx-xxx", opts)
for _, point := range metrics.Metrics {
    fmt.Printf("CPU Usage: %.2f%%\n", point.CPUUsedPct)
}
```

## 错误处理

SDK 使用自定义错误类型 `client.APIError` 来表示 API 错误：

```go
sandbox, err := sandboxClient.Get(ctx, "sbx-xxx")
if err != nil {
    if apiErr, ok := err.(*client.APIError); ok {
        switch apiErr.StatusCode {
        case 404:
            fmt.Println("Sandbox not found")
        case 401:
            fmt.Println("Unauthorized")
        default:
            fmt.Printf("API error: %s\n", apiErr.Message)
        }
    } else {
        fmt.Printf("Other error: %v\n", err)
    }
}
```

也可以使用辅助函数：

```go
if client.IsNotFound(err) {
    fmt.Println("Sandbox not found")
}
```

## 测试

本项目包含两种类型的测试：**单元测试**和**集成测试**。

### 单元测试（Unit Tests）

单元测试使用模拟的 HTTP 服务器（`httptest`）来测试 SDK 的逻辑，不需要真实的 API 服务器。

**位置**: `api/sandboxes/client_test.go`

**运行方式**:
```bash
# 运行所有单元测试
go test ./api/sandboxes/... -v

# 运行所有测试（不包括集成测试）
go test ./... -v

# 查看测试覆盖率
go test ./api/sandboxes/... -cover
```

**特点**:
- ✅ 运行速度快
- ✅ 不依赖网络连接
- ✅ 稳定性高
- ✅ 测试 SDK 代码逻辑的正确性

### 集成测试（Integration Tests）

集成测试连接到真实的 API 服务器，验证 SDK 与真实环境的集成。

**位置**: `integration_test/sandboxes_test.go`

**前置条件**:
需要设置环境变量（二选一即可）：

**方式一：使用 .env 文件（推荐）**
在项目根目录或 `integration_test/` 目录下创建 `.env` 文件：
```bash
SCALEBOX_BASE_URL=https://api.scalebox.com
SCALEBOX_API_KEY=your-api-key-here
```

**方式二：使用环境变量**
```bash
export SCALEBOX_BASE_URL="https://api.scalebox.com"
export SCALEBOX_API_KEY="your-api-key-here"
```

**运行方式**:
```bash
# 运行所有集成测试
go test -tags integration ./integration_test/... -v

# 运行特定的集成测试
go test -tags integration ./integration_test/... -v -run TestIntegrationCreateSandbox
```

**特点**:
- ✅ 测试真实环境集成
- ✅ 验证完整的业务流程
- ✅ 发现环境相关问题
- ⚠️ 需要网络连接和有效的 API 凭证
- ⚠️ 运行速度较慢
- ⚠️ 可能受外部服务影响

**注意事项**:
- 可将 `SCALEBOX_BASE_URL`、`SCALEBOX_API_KEY` 写在项目根或 `integration_test/` 下的 `.env` 中，集成测试会自动加载，无需每次 `source .env`。**请勿将 `.env` 提交到远端仓库**（已加入 `.gitignore`）。
- 如果环境变量未设置，集成测试会自动跳过
- 集成测试会自动清理创建的测试资源
- 建议在独立的测试环境中运行，避免影响生产环境

**详细说明**: 查看 [integration_test/README.md](integration_test/README.md) 了解更多信息。

### 测试对比

| 特性 | 单元测试 | 集成测试 |
|------|---------|---------|
| **位置** | `api/sandboxes/client_test.go` | `integration_test/sandboxes_test.go` |
| **运行命令** | `go test ./api/sandboxes/...` | `go test -tags integration ./integration_test/...` |
| **HTTP 服务器** | 模拟 (`httptest`) | 真实 API 服务器 |
| **网络要求** | 不需要 | 需要 |
| **运行速度** | 快 | 较慢 |
| **稳定性** | 高 | 中等 |
| **用途** | 测试代码逻辑 | 测试真实环境集成 |

### 最佳实践

1. **开发时**: 主要使用单元测试进行快速迭代
2. **提交前**: 运行单元测试确保代码正确
3. **合并前**: 运行集成测试确保与真实环境兼容
4. **发布前**: 运行完整的测试套件（单元测试 + 集成测试）

## 许可证

[添加许可证信息]
