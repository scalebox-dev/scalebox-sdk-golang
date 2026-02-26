# 贡献指南

感谢对 Scalebox Go SDK 的兴趣。以下是参与贡献时需遵循的规范。

## 代码风格

- 遵循 [Go 官方代码审查注释](https://github.com/golang/go/wiki/CodeReviewComments)
- 运行 `go fmt ./...` 格式化代码
- 所有导出的类型、函数、方法必须有注释，注释以类型/函数名开头
- 可选字段使用指针类型（`*string`、`*int`、`*bool`）
- API 客户端方法第一个参数为 `context.Context`，方法名使用动词（Create、Get、List、Update、Delete）

## 项目结构

```
client/         # 基础 HTTP 客户端
models/         # 请求/响应数据模型
api/sandboxes/  # 沙箱 API 实现
examples/       # 示例程序
integration_test/  # 集成测试
```

## 分支与提交

- 从 `develop` 分支创建功能分支
- 提交信息使用清晰的动词开头（如 `feat:`, `fix:`, `docs:`）
- 每个提交应保持单一职责、可独立理解

## 测试要求

- 新增 API 方法需补充单元测试（`api/sandboxes/client_test.go`）
- 核心功能需补充集成测试（`integration_test/sandboxes_test.go`）
- 提交前运行 `go test ./...` 确保单元测试通过
- 集成测试使用 `go test -tags integration ./integration_test/... -v`

## 新增 API 端点

1. 在 `models/` 中添加请求/响应结构体
2. 在 `api/sandboxes/client.go` 中实现方法
3. 在 `api/sandboxes/client_test.go` 中添加单元测试
4. 在 `integration_test/sandboxes_test.go` 中添加集成测试
5. 在 `docs/api.md` 中补充 API 文档
6. 在 `examples/` 中酌情添加示例

## 禁止事项

- 不要使用 `panic`（除非是程序内部错误）
- 不要忽略错误（使用 `_`）
- 不要使用 `interface{}`，使用具体类型或泛型
- 不要在公共 API 中暴露实现细节
