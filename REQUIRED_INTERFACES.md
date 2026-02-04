# Scalebox Go SDK 需要实现的接口

本文档列出了 Scalebox Go SDK 需要实现的接口，基于后端实际实现定义。

## Sandboxes 接口

### 1. 创建沙箱
- **方法**: `POST`
- **路径**: `/sandboxes`
- **功能**: 从模板创建沙箱
- **请求体**: `CreateSandboxRequest`
  - `name` (string): 沙箱名称
  - `description` (string): 描述
  - `template` (string): 模板名称或 ID，默认为 "base"
  - `project_id` (string, 可选): 项目 ID，默认为用户的默认项目
  - `cpu_count` (int): CPU 核心数
  - `memory_mb` (int): 内存大小（MB）
  - `storage_gb` (int): 存储大小（GB）
  - `metadata` (map[string]string, 可选): 元数据键值对
  - `timeout` (int, 可选): 超时时间（秒），默认 300（5分钟）
  - `auto_pause` (*bool, 可选): 如果为 true，超时后自动暂停；如果为 false/null，超时后终止（创建时设置，不可变，默认 false）
  - `env_vars` (map[string]string, 可选): 环境变量
  - `secure` (*bool, 可选): 安全设置，默认 true
  - `allow_internet_access` (*bool, 可选): 是否允许互联网访问，默认 true
  - `object_storage` (*ObjectStorageConfig, 可选): 对象存储挂载配置
  - `custom_ports` ([]PortConfig, 可选): 自定义端口列表
  - `net_proxy_country` (string, 可选): 首选代理国家（ISO 代码：us, hk, jp, ca, my, br, fr, it, cn）
  - `locality` (*LocalityRequest, 可选): 基于位置的调度偏好
- **响应**: `Sandbox` (201) 或 `Error` (400, 401, 500)

### 2. 列出沙箱
- **方法**: `GET`
- **路径**: `/sandboxes`
- **功能**: 列出沙箱列表，支持多种过滤和排序
- **查询参数**: 
  - `project_id` (可选): 项目 ID 过滤
  - `status` (可选): 沙箱状态过滤（如 "running", "paused", "terminated"）
  - `owner_user_id` (可选): 所有者用户 ID 过滤（仅 root 用户或 admin）
  - `search` (可选): 名称搜索（模糊匹配）
  - `sort_by` (可选): 排序字段
  - `sort_order` (可选): 排序顺序（"asc" 或 "desc"）
  - `limit` (可选): 每页数量限制
  - `offset` (可选): 偏移量
- **响应**: `{"sandboxes": []Sandbox}` (200) 或 `Error` (400, 401, 500)

### 3. 获取沙箱详情
- **方法**: `GET`
- **路径**: `/sandboxes/{sandbox_id}`
- **功能**: 根据 ID 获取沙箱详情
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **响应**: `Sandbox` (200) 或 `Error` (401, 404, 500)

### 4. 获取沙箱状态（轻量级）
- **方法**: `GET`
- **路径**: `/sandboxes/{sandbox_id}/status`
- **功能**: 获取沙箱的轻量级状态信息，用于轮询
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **响应**: `{"sandbox_id": string, "status": string, "substatus": string, "reason": string, "updated_at": time}` (200) 或 `Error` (404)

### 5. 更新沙箱
- **方法**: `PUT`
- **路径**: `/sandboxes/{sandbox_id}`
- **功能**: 更新沙箱信息
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **请求体**: `UpdateSandboxRequest`
  - `timeout` (int): 新的超时时间（秒），必须大于当前已使用的时间
- **响应**: `Sandbox` (200) 或 `Error` (400, 401, 404, 500)

### 6. 删除沙箱
- **方法**: `DELETE`
- **路径**: `/sandboxes/{sandbox_id}`
- **功能**: 删除指定沙箱（异步操作）
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **查询参数**:
  - `force` (可选): 是否强制删除，默认为 true（除非显式设置为 false）
- **响应**: `{"sandbox_id": string, "status": "deletion_in_progress", "note": string}` (202) 或 `Error` (400, 401, 404, 500)

### 7. 终止沙箱
- **方法**: `POST`
- **路径**: `/sandboxes/{sandbox_id}/terminate`
- **功能**: 终止沙箱但保留在数据库中（异步操作）
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **查询参数**:
  - `force` (可选): 是否强制终止，默认为 false
- **响应**: `{"sandbox_id": string, "status": "termination_in_progress"}` (202) 或 `Error` (400, 401, 404, 500)

### 8. 暂停沙箱
- **方法**: `POST`
- **路径**: `/sandboxes/{sandbox_id}/pause`
- **功能**: 暂停指定沙箱（异步操作）
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **请求体**: `PauseSandboxRequest` (空结构体，可为空)
- **响应**: `Sandbox` (200) 或 `Error` (400, 401, 403, 404, 500)

### 9. 恢复沙箱
- **方法**: `POST`
- **路径**: `/sandboxes/{sandbox_id}/resume`
- **功能**: 恢复已暂停的沙箱（异步操作）
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **请求体**: `ResumeSandboxRequest` (空结构体，可为空)
- **响应**: `Sandbox` (200) 或 `Error` (400, 401, 403, 404, 500)

### 10. 连接沙箱
- **方法**: `POST`
- **路径**: `/sandboxes/{sandbox_id}/connect`
- **功能**: 连接沙箱，如果沙箱已暂停则自动恢复，可选择性更新超时时间
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **请求体**: `ConnectSandboxRequest`
  - `timeout` (*int, 可选): 超时时间（秒），如果提供则更新沙箱超时
- **响应**: `Sandbox` (200, 201) 或 `Error` (400, 401, 404, 500)

### 11. 设置沙箱超时
- **方法**: `POST`
- **路径**: `/sandboxes/{sandbox_id}/timeout`
- **功能**: 设置沙箱超时时间
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **请求体**: `SandboxTimeoutRequest`
  - `timeout` (int): 新的超时时间（秒），必须大于或等于已使用的生命周期
- **响应**: `Sandbox` (200) 或 `Error` (400, 401, 404, 500)

### 12. 获取沙箱指标
- **方法**: `GET`
- **路径**: `/sandboxes/{sandbox_id}/metrics`
- **功能**: 获取单个沙箱的时间序列指标
- **路径参数**: 
  - `sandbox_id`: 沙箱 ID
- **查询参数**: 
  - `start` (可选): 起始时间（支持 Unix 时间戳、RFC3339 格式、简单日期时间格式），默认 5 分钟前
  - `end` (可选): 结束时间（格式同上），默认当前时间
  - `step` (可选): 采样间隔（秒），默认 5 秒
- **响应**: `SandboxMetricsResponse` (200) 或 `Error` (400, 401, 403, 404, 500)
  - `sandbox_id`: 沙箱 ID
  - `timestamp`: 当前时间戳
  - `status`: 沙箱状态
  - `uptime_seconds`: 运行时间（秒）
  - `metrics`: 指标数据点数组

## 实现状态

- [x] 1. 创建沙箱 (`POST /sandboxes`) - ✅ 已实现 (`Create()`)
- [x] 2. 列出沙箱 (`GET /sandboxes`) - ✅ 已实现 (`List()`)
- [x] 3. 获取沙箱详情 (`GET /sandboxes/{sandbox_id}`) - ✅ 已实现 (`Get()`)
- [x] 4. 获取沙箱状态 (`GET /sandboxes/{sandbox_id}/status`) - ✅ 已实现 (`GetStatus()`)
- [x] 5. 更新沙箱 (`PUT /sandboxes/{sandbox_id}`) - ✅ 已实现 (`Update()`)
- [x] 6. 删除沙箱 (`DELETE /sandboxes/{sandbox_id}`) - ✅ 已实现 (`Delete()`)
- [x] 7. 终止沙箱 (`POST /sandboxes/{sandbox_id}/terminate`) - ✅ 已实现 (`Terminate()`)
- [x] 8. 暂停沙箱 (`POST /sandboxes/{sandbox_id}/pause`) - ✅ 已实现 (`Pause()`)
- [x] 9. 恢复沙箱 (`POST /sandboxes/{sandbox_id}/resume`) - ✅ 已实现 (`Resume()`)
- [x] 10. 连接沙箱 (`POST /sandboxes/{sandbox_id}/connect`) - ✅ 已实现 (`Connect()`)
- [x] 11. 设置沙箱超时 (`POST /sandboxes/{sandbox_id}/timeout`) - ✅ 已实现 (`SetTimeout()`)
- [x] 12. 获取沙箱指标 (`GET /sandboxes/{sandbox_id}/metrics`) - ✅ 已实现 (`GetMetrics()`)

**✅ 所有接口已实现！** 详细检查报告请参考 `IMPLEMENTATION_CHECK.md`

## 参考

- 后端实现位置: `scalebox/back-end/internal/api/sandboxes.go`
- 后端路由定义: `scalebox/back-end/internal/api/server.go` (第 1433-1477 行)
- API 文档: `scalebox/docs/documentations/zh-cn/api/sandboxes.md`
