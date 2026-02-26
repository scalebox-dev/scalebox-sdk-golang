# 测试

本项目包含两种类型的测试：**单元测试**和**集成测试**。

## 单元测试（Unit Tests）

单元测试使用模拟的 HTTP 服务器（`httptest`）来测试 SDK 的逻辑，不需要真实的 API 服务器。

**位置**: `api/sandboxes/client_test.go`、`client/client_test.go`

**运行方式**:

```bash
# 运行所有单元测试
go test ./api/sandboxes/... ./client/... -v

# 运行所有测试（不包括集成测试）
go test ./... -v

# 查看测试覆盖率
go test ./api/sandboxes/... -cover
```

**特点**:

- 运行速度快
- 不依赖网络连接
- 稳定性高
- 测试 SDK 代码逻辑的正确性

## 集成测试（Integration Tests）

集成测试连接到真实的 API 服务器，验证 SDK 与真实环境的集成。

**位置**: `integration_test/sandboxes_test.go`

**前置条件**: 设置 `SCALEBOX_BASE_URL` 和 `SCALEBOX_API_KEY`（详见 [integration_test/README.md](../integration_test/README.md)）

**运行方式**:

```bash
# 运行所有集成测试
go test -tags integration ./integration_test/... -v

# 运行特定的集成测试
go test -tags integration ./integration_test/... -v -run TestIntegrationCreateSandbox
```

**特点**:

- 测试真实环境集成
- 验证完整的业务流程
- 需要网络连接和有效的 API 凭证
- 运行速度较慢

**详细说明**（环境变量、用例列表、故障排查）请查看 [integration_test/README.md](../integration_test/README.md)。

## 测试对比

| 特性       | 单元测试               | 集成测试                           |
| ---------- | ---------------------- | ---------------------------------- |
| 位置       | `api/sandboxes/*_test.go` | `integration_test/sandboxes_test.go` |
| 运行命令   | `go test ./api/sandboxes/...` | `go test -tags integration ./integration_test/...` |
| HTTP 服务器 | 模拟 (`httptest`)      | 真实 API 服务器                    |
| 网络要求   | 不需要                 | 需要                               |
| 运行速度   | 快                     | 较慢                               |
| 稳定性     | 高                     | 中等                               |
| 用途       | 测试代码逻辑           | 测试真实环境集成                   |

## 最佳实践

1. **开发时**: 主要使用单元测试进行快速迭代
2. **提交前**: 运行单元测试确保代码正确
3. **合并前**: 运行集成测试确保与真实环境兼容
4. **发布前**: 运行完整的测试套件（单元测试 + 集成测试）
