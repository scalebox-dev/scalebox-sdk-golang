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
   - make_dir() - 创建目录
   - rename() - 重命名/移动
   - remove() - 删除文件或目录
   - watch_dir() - 监控目录变化（可选）

3. **非阻塞耗时操作**：部分操作较耗时，需利用 Go 协程以非阻塞方式运行，执行完毕时唤醒协程。

4. **接口设计**：不需要像 Python SDK 那样实现同步和异步两种接口，仅实现一种即可。

5. **参考实现**：
   - Python SDK：`scalebox-sdk-python`，参考 `SDK_FEATURES.md`
   - 后端实现：`scalebox/back-end`
   - 已有 Go 实现：`scalebox-sdk-golang/api/sandboxes/client.go`

---

## 后端 API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/v1/sandboxes/:sandbox_id/files/list` | 列出目录 |
| POST | `/v1/sandboxes/:sandbox_id/files/stat` | 获取文件信息 |
| GET | `/v1/sandboxes/:sandbox_id/files/download/*filepath` | 下载文件 |
| POST | `/v1/sandboxes/:sandbox_id/files/upload` | 上传文件 |
| GET | `/v1/sandboxes/:sandbox_id/files/upload-progress` | 上传进度（SSE） |

---

## 开发进度

- [x] 创建 `models/filesystem.go` - 数据模型（EntryInfo, FileType, FileListRequest, FileListResponse）
- [x] 实现 **List** - 列出目录（`ListFiles` 在 api/sandboxes/client.go）
- [ ] 创建 `api/sandboxes/filesystem.go` - 文件系统客户端（按需）
- [ ] 增强 `client/client.go` - 支持 multipart/form-data、流式下载、SSE
- [ ] 实现基础操作：Read, Write, Exists, GetInfo
- [ ] 实现目录操作：MakeDir, Rename, Remove
- [ ] 实现上传：Upload, UploadWithProgress
- [ ] 实现下载：Download, DownloadToFile
- [ ] 实现异步接口（协程 + channel）
- [ ] 编写单元测试 `api/sandboxes/filesystem_test.go`
- [ ] 编写集成测试 `integration_test/filesystem_test.go`

---

## 实现说明

### 数据模型（models/filesystem.go）

- `EntryInfo` - 文件/目录信息
- `FileType` - 文件类型（FILE, DIRECTORY）
- `FileListRequest` / `FileListResponse` - 列出请求/响应
- `FileStatRequest` / `FileStatResponse` - Stat 请求/响应
- `WriteEntry` / `WriteInfo` - 写入条目/结果
- `UploadResponse` / `UploadProgress` - 上传响应/进度

### 协程设计

耗时操作使用 channel 返回结果：
- `UploadAsync(...) <-chan UploadResult`
- `DownloadAsync(...) <-chan DownloadResult`

---

## 更新记录

| 日期 | 更新内容 |
|------|----------|
| 2026-02-05 | 创建文档，记录用户要点与实现计划 |
