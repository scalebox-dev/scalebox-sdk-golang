# API 文档

Scalebox Go SDK 采用 **REST + gRPC/Connect 双通道** 架构：

- **REST**：沙箱 CRUD、文件操作、端口管理（经 Backend）
- **gRPC/Connect**：Commands、PTY、Code Interpreter、watch_dir（直连 Sandbox Agent）

## 创建沙箱

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

## 列出沙箱

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

## 获取沙箱详情

```go
sandbox, err := sandboxClient.Get(ctx, "sbx-xxx")
```

## 获取沙箱状态（轻量级）

```go
status, err := sandboxClient.GetStatus(ctx, "sbx-xxx")
fmt.Printf("Status: %s\n", status.Status)
```

## 更新沙箱

```go
req := models.UpdateSandboxRequest{
    Timeout: 600, // 新的超时时间（秒）
}
sandbox, err := sandboxClient.Update(ctx, "sbx-xxx", req)
```

## 删除沙箱

```go
force := true
result, err := sandboxClient.Delete(ctx, "sbx-xxx", &force)
```

## 终止沙箱

```go
force := false
result, err := sandboxClient.Terminate(ctx, "sbx-xxx", &force)
```

## 暂停沙箱

```go
sandbox, err := sandboxClient.Pause(ctx, "sbx-xxx")
```

## 恢复沙箱

```go
sandbox, err := sandboxClient.Resume(ctx, "sbx-xxx")
```

## 连接沙箱

```go
timeout := 600
req := &models.ConnectSandboxRequest{
    Timeout: &timeout, // 可选
}
sandbox, err := sandboxClient.Connect(ctx, "sbx-xxx", req)
```

## 直连 Sandbox Agent（Commands / PTY / Code Interpreter / watch_dir）

需沙箱 running 且 allow_internet_access=true。内部通过 gRPC/Connect 直连 Sandbox Agent。

```go
agent, err := sandboxClient.ConnectToAgent(ctx, "sbx-xxx")
if err != nil {
    log.Fatal(err)
}

// 执行命令
result, err := agent.Commands().Run(ctx, "echo hello", nil)
if err == nil {
    if r, ok := result.(*sandboxes.CommandResult); ok {
        fmt.Println(string(r.Stdout))
    }
}

// PTY 创建
ptyHandle, err := agent.PTY().Create(ctx, &sandboxes.PtyOptions{Size: sandboxes.PtySize{Cols: 80, Rows: 24}})

// Code Interpreter
ctxObj, _ := agent.CodeInterpreter().CreateContext(ctx, nil)
execResult, _ := agent.CodeInterpreter().RunCode(ctx, ctxObj.ID, "print(1+1)", nil)

// 监控目录
handle, _ := agent.WatchDir(ctx, "/home/user", false, func(ev sandboxes.FilesystemEvent) {
    fmt.Printf("Event: %s %v\n", ev.Name, ev.Type)
})
defer handle.Stop()
```

## 设置超时

```go
req := models.SandboxTimeoutRequest{
    Timeout: 600,
}
sandbox, err := sandboxClient.SetTimeout(ctx, "sbx-xxx", req)
```

## 获取指标

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
