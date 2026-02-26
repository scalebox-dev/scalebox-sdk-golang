---
name: ""
overview: ""
todos: []
isProject: false
---

# 文件系统功能开发进度追踪

## 开发原则

**增量开发**：一个功能一个功能地实现，跑通一个功能后再添加下一个。禁止一次性添加多个未验证的功能。

**测试要求**：新增功能后，必须生成单元测试和集成测试用例，并跑通。未通过单元测试和集成测试的功能视为未完成。

**测试范围**：为提高效率，每次新增功能只需跑新增的测试用例，无需跑全量测试。待所有功能完成后，再跑全量的单元测试和集成测试。

## 用户要点

1. **文件上传/下载**：在 scalebox-sdk-golang 中完成沙盒中文件上传、文件下载功能。
2. **文件系统功能**：完成 Python SDK 中 [SDK_FEATURES.md:425-426](https://github.com/scalebox-dev/scalebox-sdk-python/blob/main/SDK_FEATURES.md) 的文件系统功能，包括：
  - read() - 读取文件
  - write() - 写入文件（单文件/批量）
  - list() - 列出目录内容
  - exists() - 检查文件是否存在
  - get_info() - 获取文件信息
  - make_dir()、rename()、remove()：后端未实现，Go SDK 不提供
  - watch_dir()：依赖 gRPC 流式，Go SDK 使用 REST 客户端，暂不实现
3. **非阻塞耗时操作**：部分操作较耗时，需利用 Go 协程以非阻塞方式运行，执行完毕时唤醒协程。
4. **接口设计**：不需要像 Python SDK 那样实现同步和异步两种接口，仅实现一种即可。
5. **参考实现**：
  - Python SDK：`scalebox-sdk-python`，参考 `SDK_FEATURES.md`
  - 后端实现：`scalebox/back-end`
  - 已有 Go 实现：`scalebox-sdk-golang/api/sandboxes/client.go`

---

## 后端 API 端点


| 方法   | 路径                                                   | 说明            |
| ---- | ---------------------------------------------------- | ------------- |
| POST | `/v1/sandboxes/:sandbox_id/files/list`               | 列出目录          |
| POST | `/v1/sandboxes/:sandbox_id/files/stat`               | 获取文件信息        |
| POST | `/v1/sandboxes/:sandbox_id/files/mkdir`              | 创建目录（后端未实现）   |
| POST | `/v1/sandboxes/:sandbox_id/files/move`               | 移动/重命名（后端未实现） |
| POST | `/v1/sandboxes/:sandbox_id/files/remove`             | 删除（后端未实现）     |
| GET  | `/v1/sandboxes/:sandbox_id/files/download/*filepath` | 下载文件          |
| POST | `/v1/sandboxes/:sandbox_id/files/upload`             | 上传文件          |
| GET  | `/v1/sandboxes/:sandbox_id/files/upload-progress`    | 上传进度（SSE）     |


---

## 开发进度

- 创建 `models/filesystem.go` - 数据模型（EntryInfo, FileType, FileListRequest, FileListResponse）
- 实现 **List** - 列出目录（`ListFiles` 在 api/sandboxes/client.go）
- ListFiles 单元测试（`api/sandboxes/client_test.go` 中 `TestListFiles`）
- ListFiles 集成测试（`integration_test/sandboxes_test.go` 中 `TestIntegrationListFiles`）
- EntryInfo 后端兼容：FlexibleInt64（size 支持 string/number）、FileType 支持字符串枚举
- ListFiles 集成测试：Sandbox Agent 重试逻辑（2 分钟、每 10 秒重试）
- 增强 `client/client.go` - DoRequestRaw、DoRequestMultipart、CheckResponse
- 实现基础操作：Read, Write, Exists, GetInfo（Stat）
- 目录操作 MakeDir/Rename/Remove：后端未实现，SDK 不提供
- 实现上传：Upload, UploadWithProgress, Write, WriteFromReader, WriteBatch（批量，后端单文件则循环调用）
- 实现下载：Download, DownloadToFile
- 实现异步接口：UploadAsync, DownloadAsync（协程 + channel）
- 文件方法支持可选 user 参数（ListFilesOptions、FileOperationOptions）
- 单元测试：Stat, Exists, Read, Write, Upload, Download 等
- 集成测试：TestIntegrationStatReadWrite（含 Download/DownloadToFile）, TestIntegrationUpload, TestIntegrationUploadDownloadAsync

---

## 实现说明

### 数据模型（models/filesystem.go）

- `EntryInfo` - 文件/目录信息
- `FileType` - 文件类型（FILE, DIRECTORY），支持 JSON 中 string（如 "FILE_TYPE_FILE"）或 number
- `FlexibleInt64` - size 等字段，支持 JSON 中 string 或 number（后端兼容）
- `FileListRequest` / `FileListResponse` - 列出请求/响应
- `FileStatRequest` / `FileStatResponse` - Stat 请求/响应
- `WriteEntry` / `WriteInfo` - 写入条目/结果
- `UploadResponse` / `UploadProgress` - 上传响应/进度

### 协程设计

耗时操作使用 channel 返回结果：

- `UploadAsync(...) <-chan UploadResult`
- `DownloadAsync(...) <-chan DownloadResult`

### 与 Python SDK 的差异

- **watch_dir**：Python SDK 通过 gRPC/Connect 流式 API 实现；Go SDK 使用 REST API，后端无 REST/SSE 的 watch 端点，故不提供此功能。
- **user 参数**：Go SDK 已为 ListFiles、GetInfo、Stat、Read、Download、Write、Upload、WriteBatch 等增加可选 user 参数（`ListFilesOptions.User`、`FileOperationOptions.User`），需后端 REST 支持方可生效。
- **批量写入**：Go SDK 提供 `WriteBatch(ctx, sandboxID, entries []WriteEntry, opts ...*FileOperationOptions) ([]WriteInfo, error)`，后端仅支持单文件上传，实现为循环调用 Write/Upload。

---

## 更新记录


| 日期         | 更新内容                                                                                                                                                               |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-02-05 | 创建文档，记录用户要点与实现计划                                                                                                                                                   |
| 2026-02-26 | List 完成：单元测试、集成测试；EntryInfo 后端兼容（FlexibleInt64、FileType 字符串）；集成测试 Sandbox Agent 重试（2 分钟、10 秒间隔）                                                                    |
| 2026-02-05 | 完成全部文件系统功能（与后端保持一致）：client 层增强；Stat、Exists、Read、Write、Download、DownloadToFile；Upload、UploadWithProgress；UploadAsync、DownloadAsync；mkdir/move/remove 后端未实现故 SDK 不提供 |
| 2026-02-05 | 计划落地：WriteBatch 批量写入；user 参数（ListFilesOptions、FileOperationOptions）；watch_dir 不实现原因及与 Python 差异记录于「与 Python SDK 的差异」                                               |


