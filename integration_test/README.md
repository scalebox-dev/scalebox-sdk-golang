# 集成测试 (Integration Tests)

## 概述

本目录包含针对真实 Scalebox API 环境的集成测试。这些测试与单元测试（`api/sandboxes/client_test.go`）不同：

- **单元测试**: 使用 `httptest` 模拟 HTTP 服务器，测试 SDK 的逻辑
- **集成测试**: 连接到真实的 API 服务器，测试 SDK 与真实环境的集成

## 什么是集成测试？

集成测试（Integration Tests）是一种测试方法，用于测试多个组件之间的集成。对于 SDK 来说，集成测试验证：

1. SDK 能够正确连接到真实的 API 服务器
2. 请求格式正确，能够被服务器理解
3. 响应能够正确解析
4. 错误处理符合预期
5. 完整的业务流程能够正常工作

## 运行集成测试

### 前置条件

1. 确保你有访问测试环境的权限
2. 获取测试环境的 **Base URL**（API 根地址，不含路径，如 `https://api.example.com`）和 **API Key**
3. 后端需有可用集群与配额（否则创建沙箱可能返回 503）

### 配置凭证（二选一即可）

**方式一：使用 .env 文件（推荐，无需每次输入）**

在项目根目录或 `integration_test/` 目录下放置 `.env` 文件，内容示例：

```bash
SCALEBOX_BASE_URL=https://api.scalebox.com
SCALEBOX_API_KEY=your-api-key-here
```

集成测试会在启动时**自动加载**上述位置的 `.env`，因此无需再执行 `source .env`，直接运行测试即可。

**重要**：`.env` 包含敏感信息，**请勿提交到远端仓库**。项目已在 `.gitignore` 中忽略 `.env` 和 `integration_test/.env`。请复制 `integration_test/.env.example` 为 `integration_test/.env` 并填入真实配置。

**方式二：使用环境变量**

也可在运行前手动设置环境变量，或由 CI 注入：

```bash
export SCALEBOX_BASE_URL="https://api.scalebox.com"
export SCALEBOX_API_KEY="your-api-key-here"
```

### 运行测试

使用 `-tags integration` 标志运行集成测试：

```bash
# 运行所有集成测试（若已配置 .env，无需先 source）
go test -tags integration ./integration_test/... -v

# 运行特定的集成测试
go test -tags integration ./integration_test/... -v -run TestIntegrationCreateSandbox

# 运行测试并显示覆盖率
go test -tags integration ./integration_test/... -v -cover
```

### 在 CI/CD 中运行

在 CI/CD 环境中，确保设置环境变量：

```yaml
# GitHub Actions 示例
env:
  SCALEBOX_BASE_URL: ${{ secrets.SCALEBOX_BASE_URL }}
  SCALEBOX_API_KEY: ${{ secrets.SCALEBOX_API_KEY }}

steps:
  - name: Run integration tests
    run: go test -tags integration ./integration_test/... -v
```

## 测试用例说明

### TestIntegrationCreateSandbox
测试创建沙箱功能。测试后会清理创建的沙箱。

### TestIntegrationGetSandbox
测试获取沙箱详情功能。会先创建一个测试沙箱，然后获取其详情。

### TestIntegrationListSandboxes
测试列出沙箱功能。列出状态为 "running" 的沙箱。

### TestIntegrationGetSandboxStatus
测试获取沙箱状态（轻量级）功能。

### TestIntegrationGetSandboxMetrics
测试获取沙箱指标功能。如果沙箱尚未运行，会跳过测试。

### TestIntegrationPauseSandbox
测试暂停沙箱功能。如果沙箱不支持暂停操作，会跳过测试。

### TestIntegrationResumeSandbox
测试恢复沙箱功能。如果沙箱不支持恢复操作，会跳过测试。

### TestIntegrationSetTimeout
测试设置沙箱超时功能。

### TestIntegrationErrorHandling
测试错误处理，包括 404、401 等错误情况。

### TestIntegrationConnect
测试 Connect（连接/恢复沙箱）。

### TestIntegrationUpdate
测试更新沙箱（Update）。

### TestIntegrationTerminate
测试终止沙箱（Terminate）。

### TestIntegrationBatchDelete
测试批量删除。后端不支持 batch-delete 时跳过。

### TestIntegrationBatchTerminate
测试批量终止。后端不支持 batch-terminate 时跳过。

### TestIntegrationCreateTemplateFromSandbox
测试从沙箱创建模板。后端不支持时跳过。

### TestIntegrationGetPorts
测试获取端口列表。后端不支持时跳过。

### TestIntegrationAddPortRemovePort
测试添加端口和删除端口。后端不支持时跳过。

### TestIntegrationDownloadURLUploadURLGetHost
测试 DownloadURL、UploadURL、GetHost（URL 构建与 host 获取）。

### TestIntegrationWriteBatchConcurrent
测试 WriteBatchConcurrent 并发批量写入。

### TestIntegrationUploadWithProgress
测试 UploadWithProgress 带进度的上传。

### TestIntegrationConnectToAgent
测试 ConnectToAgent、Commands.Run、Code Interpreter（CreateContext、RunCode、DestroyContext）。

### TestIntegrationAgentPTYCommandsFilesystem
测试 Agent 扩展能力：Commands.List/Kill/SendStdin/Connect、PTY.Create/SendStdin/Resize/Kill、MakeDir、Move、Remove、WatchDir。

## 注意事项

1. **不要提交 .env**：`.env` 和 `integration_test/.env` 已加入 `.gitignore`，请勿将包含 API Key 的 `.env` 提交到远端仓库。
2. **资源清理**: 所有测试都会在完成后清理创建的沙箱资源
3. **环境隔离**: 建议使用独立的测试环境，避免影响生产环境
4. **测试稳定性**: 集成测试依赖于网络和外部服务，可能比单元测试更不稳定
5. **跳过机制**: 如果环境变量未设置，测试会自动跳过
6. **测试顺序**: 测试之间是独立的，可以并行运行

## 与单元测试的区别

| 特性 | 单元测试 | 集成测试 |
|------|---------|---------|
| 位置 | `api/sandboxes/client_test.go` | `integration_test/sandboxes_test.go` |
| 运行方式 | `go test ./api/sandboxes/...` | `go test -tags integration ./integration_test/...` |
| HTTP 服务器 | 模拟 (`httptest`) | 真实 API 服务器 |
| 网络要求 | 不需要 | 需要网络连接 |
| 运行速度 | 快 | 较慢 |
| 稳定性 | 高 | 中等 |
| 用途 | 测试逻辑正确性 | 测试真实环境集成 |

## 最佳实践

1. **开发时**: 主要使用单元测试进行快速迭代
2. **提交前**: 运行单元测试确保代码正确
3. **合并前**: 运行集成测试确保与真实环境兼容
4. **发布前**: 运行完整的集成测试套件

## 故障排查

### 测试被跳过

如果看到 "跳过集成测试" 的消息，检查环境变量是否正确设置：

```bash
echo $SCALEBOX_BASE_URL
echo $SCALEBOX_API_KEY
```

### 网络错误

如果遇到网络错误，检查：
- 网络连接是否正常
- Base URL 是否正确
- 防火墙设置是否允许访问

### 认证错误

如果遇到 401/403 错误，检查：
- API Key 是否正确
- API Key 是否有足够的权限
- API Key 是否已过期

### template validation failed: template with name 'base' not found

helm-deployment 的 `push-public-template.sh` 仅注册 **code-interpreter** 模板，不包含 `base`。集成测试默认使用 `code-interpreter`。若环境有其他模板，可设置 `SCALEBOX_TEMPLATE`：

```bash
SCALEBOX_TEMPLATE=base   # 若环境有 base 模板
```

### Memory must be at least 2048 MB (template minimum)

code-interpreter 模板要求内存至少 2048 MB，集成测试已按此配置。若使用其他模板，需确保 MemoryMB 满足该模板要求。

### 503 Service Temporarily Unavailable（常见）

当所有测试返回 `503 Service Temporarily Unavailable` 且响应为 nginx 默认 HTML 页时，表示 **API 后端/上游不可用**，常见原因：

1. **公网环境（如 api.scalebox1.com）后端未启动**
   - AWS 集群可能已关机（EC2 stop），需执行 `helm-deployment/scripts/start-clusters.sh` 启动
   - 或 Backend Pod 未就绪，需检查 K8s 集群状态

2. **使用 port-forward 连接本地集群**
   若你已通过 scalebox-test-platform 部署了 control-plane，并配置了 kubeconfig，可先建立 port-forward：

   ```bash
   # 在 dev-server 项目根执行（需已配置 KUBECONFIG 指向 control-plane）
   kubectl port-forward svc/backend-service 8000:8000 -n scalebox
   ```

   然后在 `integration_test/.env` 中配置：

   ```bash
   SCALEBOX_BASE_URL=http://127.0.0.1:8000
   SCALEBOX_API_KEY=sk-e2e0root0cli0sandbox00000000000000000000
   ```

3. **后端返回 503（非 nginx 页面）**
   若响应为 JSON 而非 HTML，可能是后端业务逻辑返回 503，常见于：
   - 无可用集群（需执行 onboard-cluster.sh 注册 onboard 集群）
   - 无可用配额（需执行 create-root-voucher.sh 为 root 用户创建代金券）

**快速验证**：`curl -s -o /dev/null -w "%{http_code}" -H "X-API-KEY: $SCALEBOX_API_KEY" "$SCALEBOX_BASE_URL/health"`，期望 200。
