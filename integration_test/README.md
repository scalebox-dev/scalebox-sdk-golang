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

在项目根目录或 `integration/` 目录下放置 `.env` 文件，内容示例：

```bash
SCALEBOX_BASE_URL=https://api.scalebox.com
SCALEBOX_API_KEY=your-api-key-here
```

集成测试会在启动时**自动加载**上述位置的 `.env`，因此无需再执行 `source .env`，直接运行测试即可。

**重要**：`.env` 包含敏感信息，**请勿提交到远端仓库**。项目已在 `.gitignore` 中忽略 `.env` 和 `integration/.env`。请复制 `integration/.env.example` 为 `integration/.env` 并填入真实配置。

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
go test -tags integration ./integration/... -v

# 运行特定的集成测试
go test -tags integration ./integration/... -v -run TestIntegrationCreateSandbox

# 运行测试并显示覆盖率
go test -tags integration ./integration/... -v -cover
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
    run: go test -tags integration ./integration/... -v
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

## 注意事项

1. **不要提交 .env**：`.env` 和 `integration/.env` 已加入 `.gitignore`，请勿将包含 API Key 的 `.env` 提交到远端仓库。
2. **资源清理**: 所有测试都会在完成后清理创建的沙箱资源
3. **环境隔离**: 建议使用独立的测试环境，避免影响生产环境
4. **测试稳定性**: 集成测试依赖于网络和外部服务，可能比单元测试更不稳定
5. **跳过机制**: 如果环境变量未设置，测试会自动跳过
6. **测试顺序**: 测试之间是独立的，可以并行运行

## 与单元测试的区别

| 特性 | 单元测试 | 集成测试 |
|------|---------|---------|
| 位置 | `api/sandboxes/client_test.go` | `integration/sandboxes_test.go` |
| 运行方式 | `go test ./api/sandboxes/...` | `go test -tags integration ./integration/...` |
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
