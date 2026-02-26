# Go SDK 优化空间分析

> 基于 scalebox-dev/back-end 对照分析，待后续实现

---

## 一、性能优化空间

### 1. HTTP 层

| 方向 | 现状 | 建议 |
|------|------|------|
| 连接复用 | `NewClient` 使用默认 `http.Client`，无显式 Transport | 配置 `Transport.MaxIdleConns`、`MaxIdleConnsPerHost`、`IdleConnTimeout`，减少连接建立开销 |
| 超时 | 固定 30s 全局限时 | 支持可配置，或按操作区分：大文件上传/下载用更长超时 |
| 重试 | 无 | 对 5xx、网络错误做指数退避重试；429 单独处理（退避 + Retry-After） |

### 2. 大文件传输

| 方向 | 现状 | 建议 |
|------|------|------|
| 下载 | `Read()` 用 `io.ReadAll` 读入内存 | 已提供 `Download()` 流式，`Read()` 适合小文件；`DownloadToFile` 仍用 `ReadAll` 整块读取 |
| 下载到文件 | `DownloadToFile` 先 `io.ReadAll` 再 `os.WriteFile` | 用 `io.Copy(dstFile, resp.Body)` 流式写入，减少内存峰值 |
| 上传 | 已支持 `UploadFromReader` 流式 | 保持，大文件应优先用 Reader |
| WriteBatch | 串行上传，每次单独请求 | 批量场景可用 goroutine + semaphore 并发，后端无 batch 则仅做并发控制 |

### 3. 响应解析

| 方向 | 现状 | 建议 |
|------|------|------|
| 解析方式 | `ParseResponse` 用 `io.ReadAll` + 多次 `json.Unmarshal` | 可考虑 `json.Decoder.Decode` 一次解析；或复用 buffer pool 减少 alloc |
| 错误路径 | 错误时解析 2–3 种格式 | 可按 Content-Type 或首字节分支，减少无效解析 |
| streamUploadProgress | 每行 `[]byte(line[6:])` 产生新切片 | 可用 `strings.TrimPrefix` 或复用 buffer |

### 4. gRPC/Connect Agent

| 方向 | 现状 | 建议 |
|------|------|------|
| Transport | `http.DefaultTransport`，无连接池配置 | 与 REST 共用或独立配置 `Transport` |
| 连接 | 每次 `ConnectToAgent` 新建 `AgentClient` | 长连接场景可缓存 `AgentClient`（按 sandbox 维度） |
| 流式 | Connect 已支持流式 | 保持流式用法，避免把大流全部读入内存 |

### 5. 批量与并发

| 方向 | 现状 | 建议 |
|------|------|------|
| 批量 API | 无 batch-delete、batch-terminate 等 | 实现批量接口，减少单次请求次数 |
| 用户侧批量 | WriteBatch 串行 | 提供可配置并发度（如 `WriteBatchConcurrent`） |
| List + Get | 获取多沙箱需多次 Get | 无批量 Get 时，可提供 `GetBatch` 辅助（goroutine + 限流） |

### 6. 其他

| 方向 | 现状 | 建议 |
|------|------|------|
| URL 构建 | 每次 `url.Parse(c.BaseURL)` + `u.Path = path` | BaseURL 不变时可缓存 `*url.URL`，只改 Path |
| queryParams | `map[string]string` 每次新建 | 小改动，可复用或预分配 |
| JSON Marshal | `DoRequest` 每次 `json.Marshal(body)` | 小请求体影响小；大 body 可考虑流式编码 |

---

## 二、后端已有但 SDK 未实现的能力

### 2.1 Sandbox 相关

| 端点 | 优先级 | 说明 |
|------|--------|------|
| `GET /v1/sandboxes/eligibility` | 中 | 创建资格检查 |
| `GET /v1/sandboxes/stats` | 中 | 沙箱统计 |
| `GET /v1/sandboxes/:id/metrics/chart` | 低 | 图表用指标 |
| `POST /v1/sandboxes/:id/create-template` | 高 | 从沙箱创建模板 |
| `POST /v1/sandboxes/batch-delete` | 高 | 批量删除 |
| `POST /v1/sandboxes/batch-terminate` | 高 | 批量终止 |
| `POST /v1/sandboxes/batch-pause` | 中 | 批量暂停 |
| `POST /v1/sandboxes/batch-resume` | 中 | 批量恢复 |
| `POST /v1/sandboxes/:id/switch-project` | 中 | 切换项目 |

### 2.2 端口管理

| 端点 | 说明 |
|------|------|
| `GET /v1/sandboxes/:id/ports` | 获取端口列表 |
| `POST /v1/sandboxes/:id/ports` | 添加端口 |
| `DELETE /v1/sandboxes/:id/ports/:port` | 删除端口 |

### 2.3 终端 WebSocket

- `GET /v1/sandboxes/:id/connect-terminal?token=<jwt>` 需 JWT，SDK 用 API Key，需适配

## 二、参数与响应结构差异

- List：后端支持 `page`（1-based），SDK 仅 `offset`
- List 响应：后端返回 `page`, `total`, `total_pages`，SDK 未解析
- Pause/Resume：后端支持 `is_async`，SDK 未暴露

## 三、功能优先级建议

1. **高**：端口管理、批量操作、CreateTemplateFromSandbox
2. **中**：Eligibility、Stats、SwitchProject、List 分页增强
3. **低**：ConnectTerminal、MetricsChart
