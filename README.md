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

运行测试：

```bash
go test ./... -v
```

## 许可证

[添加许可证信息]
