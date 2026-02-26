---
name: Go SDK gRPC/Connect 实施计划
overview: Go SDK 引入 gRPC/Connect 客户端直连 Sandbox Agent，实现 Commands、Code Interpreter、PTY、watch_dir 等能力，与 Python SDK 架构对齐。
todos:
  - id: proto-setup
    content: 引入 api.proto 并生成 pb 与 pbconnect 代码
    status: completed
  - id: agent-client
    content: 实现 AgentClient（baseURL、authToken、Process/Filesystem/Execution/Context 客户端）
    status: completed
  - id: commands-impl
    content: 实现 Commands：run、list、kill、send_stdin、connect
    status: completed
  - id: pty-impl
    content: 实现 PTY：create、send_stdin、resize、kill
    status: completed
  - id: code-interpreter-impl
    content: 实现 Code Interpreter：run_code、create_code_context、destroy_context
    status: completed
  - id: watch-dir-impl
    content: 实现 watch_dir（WatchDir）
    status: completed
  - id: integration
    content: 整合 ConnectToAgent 到 SandboxClient
    status: completed
isProject: false
---

# Go SDK gRPC/Connect 实施计划

## 一、目标

通过引入 **gRPC/Connect 客户端直连 Sandbox Agent**，使 Go SDK 支持与 Python SDK 对等的：

- **Commands**：run、list、kill、send_stdin、connect
- **Code Interpreter**：run_code、create_code_context、destroy_context
- **PTY**：create、send_stdin、resize、kill
- **watch_dir**：目录监控

## 二、参考实现

| 项目 | 路径 | 说明 |
|------|------|------|
| Proto 定义 | [scalebox-dev/scalebox-cli/internal/proto/api.proto](../scalebox-dev/scalebox-cli/internal/proto/api.proto) | Sandbox Agent 的 proto |
| Connect 客户端 | [scalebox-dev/scalebox-cli/internal/sandbox/connection.go](../scalebox-dev/scalebox-cli/internal/sandbox/connection.go) | SandboxConnection 使用 ProcessClient、FilesystemClient |
| 后端终端 | [scalebox-dev/back-end/internal/api/sandbox_terminal.go](../scalebox-dev/back-end/internal/api/sandbox_terminal.go) | Backend 使用 pbconnect.ProcessClient 与 Agent 通信 |

## 三、Proto 服务概览

```
service Filesystem {
  Stat, MakeDir, Move, ListDir, Remove
  WatchDir (stream)
  CreateWatcher, GetWatcherEvents, RemoveWatcher
}

service Process {
  List, Connect (stream), Start (stream), Update
  StreamInput (stream), SendInput, SendSignal
}

service ExecutionService {
  Execute (stream)
}

service ContextService {
  CreateContext, DestroyContext
}
```

## 四、认证与连接

### 4.1 直连 Sandbox Agent 的要求

- **Base URL**：`https://{sandbox.Domain}`（从 Backend 获取沙箱详情得到）
- **Auth**：`Authorization: Bearer root` + `X-Access-Token: {sandbox.EnvdAccessToken}`（与 Backend 文件代理一致）

### 4.2 数据流

1. 用户调用 Go SDK：`client.Connect(ctx, sandboxID)` 或 `client.Commands(sandboxID).Run(...)`
2. SDK 通过 Backend REST 获取沙箱详情（含 domain、envd_access_token）
3. 使用 domain + token 创建 ConnectRPC HTTP 客户端，直连 `https://{domain}`
4. 调用 Process/Filesystem/ExecutionService/ContextService

## 五、目录与依赖

### 5.1 建议目录结构

```
scalebox-sdk-golang/
├── proto/                    # 新增：proto 文件
│   └── api.proto             # 从 scalebox-dev 复制或 submodule
├── pb/                       # 新增：生成的 Go 代码（或 internal/pb）
│   ├── api.pb.go
│   └── pbconnect/
│       └── api.connect.go
├── api/
│   └── sandboxes/
│       ├── client.go         # REST 客户端（现有）
│       └── agent_client.go   # 新增：gRPC/Connect Agent 客户端
├── models/
│   └── ...
└── go.mod
```

### 5.2 依赖

```go
// go.mod 新增
connectrpc.com/connect v1.x
google.golang.org/protobuf v1.x
```

生成命令（buf 或 protoc）：

```bash
# 使用 connectrpc 的 protoc-gen-connect-go
buf generate
# 或
protoc --go_out=. --go_opt=paths=source_relative \
       --connect-go_out=. --connect-go_opt=paths=source_relative \
       proto/api.proto
```

## 六、AgentClient 设计

### 6.1 接口

```go
// AgentClient 直连 Sandbox Agent 的 gRPC/Connect 客户端
type AgentClient struct {
    baseURL    string // https://{sandbox.Domain}
    authToken  string // sandbox.EnvdAccessToken
    httpClient *http.Client

    processClient    pbconnect.ProcessClient
    filesystemClient pbconnect.FilesystemClient
    executionClient  pbconnect.ExecutionServiceClient
    contextClient    pbconnect.ContextServiceClient
}

// NewAgentClient 创建 Agent 客户端
// baseURL: https://{sandbox.Domain}
// authToken: sandbox.EnvdAccessToken
func NewAgentClient(baseURL, authToken string, opts ...AgentClientOption) *AgentClient

// Commands 返回 Commands 操作接口
func (c *AgentClient) Commands() *CommandsClient

// PTY 返回 PTY 操作接口
func (c *AgentClient) PTY() *PTYClient

// CodeInterpreter 返回 Code Interpreter 操作接口
func (c *AgentClient) CodeInterpreter() *CodeInterpreterClient

// WatchDir 监控目录变化
func (c *AgentClient) WatchDir(ctx context.Context, path string, recursive bool, onEvent func(FilesystemEvent)) (WatchHandle, error)
```

### 6.2 与 REST Client 的整合

两种整合方式：

| 方式 | 说明 |
|------|------|
| A. 独立 AgentClient | 用户先 `sandboxes.Get()` 取 domain/token，再 `agent.NewAgentClient(domain, token)` |
| B. 整合到 SandboxClient | `client.ConnectToAgent(ctx, sandboxID)` 内部调 Get，返回 `*AgentClient` |

推荐 **B**：在 `api/sandboxes/client.go` 增加 `ConnectToAgent(ctx, sandboxID)`，内部调用 `Get` 获取 domain/token，构造并返回 `*AgentClient`。

## 七、实施步骤

### 步骤 1：Proto 与代码生成

1. 复制 [scalebox-dev/scalebox-cli/internal/proto/api.proto](../scalebox-dev/scalebox-cli/internal/proto/api.proto) 到 `proto/api.proto`
2. 配置 buf 或 Makefile，生成 `pb/` 与 `pbconnect/`
3. 在 go.mod 中加入 connectrpc.com/connect、google.golang.org/protobuf

### 步骤 2：AgentClient 基础

1. 实现 `NewAgentClient(baseURL, authToken)`，创建带 auth 的 HTTP Client
2. 初始化 ProcessClient、FilesystemClient、ExecutionServiceClient、ContextServiceClient
3. 参考 [sandbox_terminal.go](../scalebox-dev/back-end/internal/api/sandbox_terminal.go) 的 authTransport 模式

### 步骤 3：Commands

- `List` → `Process.List`
- `Start`（含 PTY）→ `Process.Start`（stream）
- `Connect`(pid) → `Process.Connect`(pid)
- `SendInput`(pid, data) → `Process.SendInput`
- `Kill`(pid) → `Process.SendSignal`(SIGKILL)

### 步骤 4：PTY

- `Create`(size, user, cwd, envs) → `Process.Start`(process=/bin/bash, pty=size)
- `SendStdin`(pid, data) → `Process.SendInput`(pty)
- `Resize`(pid, size) → `Process.Update`(pty)
- `Kill`(pid) → `Process.SendSignal`(SIGKILL)

### 步骤 5：Code Interpreter

- `CreateContext`(cwd, language) → `ContextService.CreateContext`
- `DestroyContext`(ctxID) → `ContextService.DestroyContext`
- `RunCode`(ctxID, code, language, ...) → `ExecutionService.Execute`（stream）

### 步骤 6：watch_dir

- 使用 `Filesystem.WatchDir`（stream）或 `CreateWatcher` + 轮询 `GetWatcherEvents`
- 暴露 `WatchDir(ctx, path, recursive, onEvent) (WatchHandle, error)`

### 步骤 7：整合与文档

1. 在 `api/sandboxes/client.go` 增加 `ConnectToAgent(ctx, sandboxID) (*AgentClient, error)`
2. 更新 README、STRUCTURE.md，说明双通道架构与使用方式

## 八、关键文件

| 文件 | 用途 |
|------|------|
| [scalebox-dev/scalebox-cli/internal/proto/api.proto](../scalebox-dev/scalebox-cli/internal/proto/api.proto) | Proto 源 |
| [scalebox-dev/scalebox-cli/internal/sandbox/connection.go](../scalebox-dev/scalebox-cli/internal/sandbox/connection.go) | 参考实现 |
| [scalebox-dev/back-end/internal/api/sandbox_terminal.go](../scalebox-dev/back-end/internal/api/sandbox_terminal.go) | 认证与 Process 调用参考 |
| [api/sandboxes/client.go](api/sandboxes/client.go) | 新增 ConnectToAgent |
| [api/sandboxes/agent_client.go](api/sandboxes/agent_client.go) | 新增 AgentClient |

## 九、风险与注意事项

- **网络**：直连 Sandbox Agent 需沙箱有公网 domain（InternetAccess=true），与文件操作一致
- **Token**：EnvAccessToken 由 Backend 管理，需通过 `Get` 沙箱详情获取
- **版本**：Proto 需与 Sandbox Agent 版本匹配，建议与 scalebox-dev 同步更新
