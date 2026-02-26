package sandboxes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// Client provides methods for interacting with the Sandboxes API
type Client struct {
	baseClient *client.Client
}

// NewClient creates a new Sandboxes API client
func NewClient(baseClient *client.Client) *Client {
	return &Client{
		baseClient: baseClient,
	}
}

// Create creates a new sandbox
func (c *Client) Create(ctx context.Context, req models.CreateSandboxRequest) (*models.Sandbox, error) {
	resp, err := c.baseClient.DoRequest(ctx, "POST", "/v1/sandboxes", req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// List lists sandboxes with optional filters.
// When opts.Page > 0, uses page-based pagination; otherwise uses offset.
func (c *Client) List(ctx context.Context, opts *models.ListSandboxesOptions) (*models.SandboxListResponse, error) {
	queryParams := make(map[string]string)
	if opts != nil {
		if opts.ProjectID != "" {
			queryParams["project_id"] = opts.ProjectID
		}
		if opts.Status != "" {
			queryParams["status"] = opts.Status
		}
		if opts.OwnerUserID != "" {
			queryParams["owner_user_id"] = opts.OwnerUserID
		}
		if opts.Search != "" {
			queryParams["search"] = opts.Search
		}
		if opts.SortBy != "" {
			queryParams["sort_by"] = opts.SortBy
		}
		if opts.SortOrder != "" {
			queryParams["sort_order"] = opts.SortOrder
		}
		if opts.Limit > 0 {
			queryParams["limit"] = strconv.Itoa(opts.Limit)
		}
		if opts.Page > 0 {
			queryParams["page"] = strconv.Itoa(opts.Page)
		} else if opts.Offset > 0 {
			queryParams["offset"] = strconv.Itoa(opts.Offset)
		}
	}

	resp, err := c.baseClient.DoRequest(ctx, "GET", "/v1/sandboxes", nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.SandboxListResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get retrieves a sandbox by ID
func (c *Client) Get(ctx context.Context, sandboxID string) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// GetStatus retrieves lightweight sandbox status
func (c *Client) GetStatus(ctx context.Context, sandboxID string) (*models.SandboxStatus, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/status", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var status models.SandboxStatus
	if err := c.baseClient.ParseResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Update updates a sandbox
func (c *Client) Update(ctx context.Context, sandboxID string, req models.UpdateSandboxRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "PUT", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Delete deletes a sandbox
func (c *Client) Delete(ctx context.Context, sandboxID string, force *bool) (*models.DeletionResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	queryParams := make(map[string]string)
	if force != nil && !*force {
		queryParams["force"] = "false"
	}

	resp, err := c.baseClient.DoRequest(ctx, "DELETE", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.DeletionResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Terminate terminates a sandbox
func (c *Client) Terminate(ctx context.Context, sandboxID string, force *bool) (*models.TerminationResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/terminate", sandboxID)
	queryParams := make(map[string]string)
	if force != nil && *force {
		queryParams["force"] = "true"
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.TerminationResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// BatchDelete deletes multiple sandboxes in one request.
func (c *Client) BatchDelete(ctx context.Context, req models.BatchDeleteRequest) (*models.BatchDeleteResponse, error) {
	resp, err := c.baseClient.DoRequest(ctx, "POST", "/v1/sandboxes/batch-delete", req, nil)
	if err != nil {
		return nil, err
	}
	var result models.BatchDeleteResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// BatchTerminate terminates multiple sandboxes in one request.
func (c *Client) BatchTerminate(ctx context.Context, req models.BatchTerminateRequest) (*models.BatchTerminateResponse, error) {
	resp, err := c.baseClient.DoRequest(ctx, "POST", "/v1/sandboxes/batch-terminate", req, nil)
	if err != nil {
		return nil, err
	}
	var result models.BatchTerminateResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Pause pauses a sandbox. Pass opts with IsAsync: true to return immediately without waiting.
func (c *Client) Pause(ctx context.Context, sandboxID string, opts ...*models.PauseOptions) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/pause", sandboxID)
	req := models.PauseSandboxRequest{}
	if len(opts) > 0 && opts[0] != nil && opts[0].IsAsync {
		req.IsAsync = &opts[0].IsAsync
	}
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Resume resumes a sandbox. Pass opts with IsAsync: true to return immediately without waiting.
func (c *Client) Resume(ctx context.Context, sandboxID string, opts ...*models.ResumeOptions) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/resume", sandboxID)
	req := models.ResumeSandboxRequest{}
	if len(opts) > 0 && opts[0] != nil && opts[0].IsAsync {
		req.IsAsync = &opts[0].IsAsync
	}
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Connect connects to a sandbox (resumes if paused)
func (c *Client) Connect(ctx context.Context, sandboxID string, req *models.ConnectSandboxRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/connect", sandboxID)
	if req == nil {
		req = &models.ConnectSandboxRequest{}
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// SetTimeout sets the timeout for a sandbox
func (c *Client) SetTimeout(ctx context.Context, sandboxID string, req models.SandboxTimeoutRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/timeout", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// CreateTemplateFromSandbox creates a template from an existing sandbox.
func (c *Client) CreateTemplateFromSandbox(ctx context.Context, sandboxID string, req models.CreateTemplateFromSandboxRequest) (*models.CreateTemplateResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/create-template", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}
	var result models.CreateTemplateResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPorts returns the port configurations for a sandbox.
func (c *Client) GetPorts(ctx context.Context, sandboxID string) ([]models.PortConfig, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/ports", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Ports []models.PortConfig `json:"ports"`
	}
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Ports, nil
}

// AddPort adds a port configuration to a sandbox.
func (c *Client) AddPort(ctx context.Context, sandboxID string, req models.AddPortRequest) (*models.PortConfig, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/ports", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}
	var port models.PortConfig
	if err := c.baseClient.ParseResponse(resp, &port); err != nil {
		return nil, err
	}
	return &port, nil
}

// RemovePort removes a port from a sandbox.
func (c *Client) RemovePort(ctx context.Context, sandboxID string, port int32) error {
	path := fmt.Sprintf("/v1/sandboxes/%s/ports/%d", sandboxID, port)
	resp, err := c.baseClient.DoRequest(ctx, "DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	return c.baseClient.ParseResponse(resp, nil)
}

// DownloadURL fetches sandbox info and returns a download URL for the given path.
// Use for presigned URLs or direct HTTP GET. opts is optional.
func (c *Client) DownloadURL(ctx context.Context, sandboxID string, path string, opts *FileURLOptions) (string, error) {
	sandbox, err := c.Get(ctx, sandboxID)
	if err != nil {
		return "", fmt.Errorf("get sandbox: %w", err)
	}
	return BuildDownloadURL(sandbox, path, opts)
}

// UploadURL fetches sandbox info and returns an upload URL for the given path.
// Use for presigned URLs. opts is optional; path may be empty (set when uploading).
func (c *Client) UploadURL(ctx context.Context, sandboxID string, path string, opts *FileURLOptions) (string, error) {
	sandbox, err := c.Get(ctx, sandboxID)
	if err != nil {
		return "", fmt.Errorf("get sandbox: %w", err)
	}
	return BuildUploadURL(sandbox, path, opts)
}

// GetHost fetches sandbox info and returns the host address.
// If port > 0, returns "domain:port"; otherwise returns "domain".
func (c *Client) GetHost(ctx context.Context, sandboxID string, port int) (string, error) {
	sandbox, err := c.Get(ctx, sandboxID)
	if err != nil {
		return "", fmt.Errorf("get sandbox: %w", err)
	}
	return GetHostFromSandbox(sandbox, port)
}

// ListFiles lists directory contents in a sandbox.
// opts is optional; pass ListFilesOptions with User to run as a specific user.
func (c *Client) ListFiles(ctx context.Context, sandboxID string, path string, depth uint32, opts ...*models.ListFilesOptions) (*models.FileListResponse, error) {
	apiPath := fmt.Sprintf("/v1/sandboxes/%s/files/list", sandboxID)
	req := models.FileListRequest{Path: path}
	if depth > 0 {
		req.Depth = depth
	} else {
		req.Depth = 1
	}
	if len(opts) > 0 && opts[0] != nil && opts[0].User != "" {
		req.User = opts[0].User
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", apiPath, req, nil)
	if err != nil {
		return nil, err
	}

	var result models.FileListResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Stat retrieves file or directory info (alias: GetInfo)
func (c *Client) Stat(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) (*models.EntryInfo, error) {
	return c.GetInfo(ctx, sandboxID, path, opts...)
}

// GetInfo retrieves file or directory info.
// opts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) GetInfo(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) (*models.EntryInfo, error) {
	apiPath := fmt.Sprintf("/v1/sandboxes/%s/files/stat", sandboxID)
	req := models.FileStatRequest{Path: path}
	if len(opts) > 0 && opts[0] != nil && opts[0].User != "" {
		req.User = opts[0].User
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", apiPath, req, nil)
	if err != nil {
		return nil, err
	}

	var statResp models.FileStatResponse
	if err := c.baseClient.ParseResponse(resp, &statResp); err != nil {
		return nil, err
	}
	return &statResp.Entry, nil
}

// Exists checks if a file or directory exists
func (c *Client) Exists(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) (bool, error) {
	_, err := c.GetInfo(ctx, sandboxID, path, opts...)
	if err != nil {
		if client.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Read reads file content and returns bytes. Equivalent to Python files.read(format="bytes").
// For streamed reading, use ReadStream or Download.
func (c *Client) Read(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) ([]byte, error) {
	rc, err := c.Download(ctx, sandboxID, path, opts...)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// Download returns a stream of file content. Caller must close the returned ReadCloser.
// Equivalent to Python files.read(format="stream"). Use Read for loading all bytes at once.
// opts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) Download(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) (io.ReadCloser, error) {
	path = strings.TrimPrefix(path, "/")
	apiPath := fmt.Sprintf("/v1/sandboxes/%s/files/download/%s", sandboxID, path)

	var queryParams map[string]string
	if len(opts) > 0 && opts[0] != nil && opts[0].User != "" {
		queryParams = map[string]string{"user": opts[0].User}
	}

	resp, err := c.baseClient.DoRequestRaw(ctx, "GET", apiPath, nil, nil, queryParams)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := c.baseClient.ParseResponse(resp, nil)
		return nil, err
	}
	return resp.Body, nil
}

// ReadStream returns a stream of file content. Caller must close the returned ReadCloser.
// Equivalent to Python files.read(format="stream"). Same as Download, provided for API parity.
func (c *Client) ReadStream(ctx context.Context, sandboxID string, path string, opts ...*models.FileOperationOptions) (io.ReadCloser, error) {
	return c.Download(ctx, sandboxID, path, opts...)
}

// DownloadToFile downloads a remote file to a local path using streaming io.Copy to reduce memory usage.
func (c *Client) DownloadToFile(ctx context.Context, sandboxID string, remotePath string, localPath string, opts ...*models.FileOperationOptions) error {
	rc, err := c.Download(ctx, sandboxID, remotePath, opts...)
	if err != nil {
		return err
	}
	defer rc.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, rc)
	return err
}

// Write writes data to a file in the sandbox.
// opts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) Write(ctx context.Context, sandboxID string, path string, data []byte, opts ...*models.FileOperationOptions) error {
	return c.WriteFromReader(ctx, sandboxID, path, bytes.NewReader(data), int64(len(data)), opts...)
}

// WriteFromReader writes content from a reader to a file in the sandbox
func (c *Client) WriteFromReader(ctx context.Context, sandboxID string, path string, r io.Reader, size int64, opts ...*models.FileOperationOptions) error {
	_, err := c.UploadFromReader(ctx, sandboxID, path, r, size, nil, opts...)
	return err
}

// WriteBatch writes multiple files to the sandbox. Backend supports single-file upload only,
// so each entry is uploaded individually. Returns WriteInfo for each successful write.
// On first error, returns partial results and the error.
// opts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) WriteBatch(ctx context.Context, sandboxID string, entries []models.WriteEntry, opts ...*models.FileOperationOptions) ([]models.WriteInfo, error) {
	results := make([]models.WriteInfo, 0, len(entries))
	for _, e := range entries {
		r := e.Data
		size := e.ContentLength
		if size < 0 {
			data, err := io.ReadAll(e.Data)
			if err != nil {
				return results, err
			}
			r = bytes.NewReader(data)
			size = int64(len(data))
		}
		if err := c.WriteFromReader(ctx, sandboxID, e.Path, r, size, opts...); err != nil {
			return results, err
		}
		name := filepath.Base(e.Path)
		if name == "" || name == "." {
			name = "file"
		}
		results = append(results, models.WriteInfo{
			Name: name,
			Type: models.FileTypeFile,
			Path: e.Path,
		})
	}
	return results, nil
}

// WriteBatchConcurrent writes multiple files with concurrent uploads.
// concurrency limits parallel uploads (default 4). On first error, returns partial results.
func (c *Client) WriteBatchConcurrent(ctx context.Context, sandboxID string, entries []models.WriteEntry, concurrency int, opts ...*models.FileOperationOptions) ([]models.WriteInfo, error) {
	if concurrency <= 0 {
		concurrency = 4
	}
	// Preprocess: resolve readers with unknown size
	type work struct {
		idx  int
		path string
		data []byte
	}
	workItems := make([]work, 0, len(entries))
	for i, e := range entries {
		data, err := io.ReadAll(e.Data)
		if err != nil {
			return nil, fmt.Errorf("read entry %d: %w", i, err)
		}
		workItems = append(workItems, work{idx: i, path: e.Path, data: data})
	}

	results := make([]models.WriteInfo, len(entries))
	var mu sync.Mutex
	var firstErr error
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, w := range workItems {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}
			r := bytes.NewReader(w.data)
			err := c.WriteFromReader(ctx, sandboxID, w.path, r, int64(len(w.data)), opts...)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			name := filepath.Base(w.path)
			if name == "" || name == "." {
				name = "file"
			}
			mu.Lock()
			results[w.idx] = models.WriteInfo{Name: name, Type: models.FileTypeFile, Path: w.path}
			mu.Unlock()
		}()
	}
	wg.Wait()
	if firstErr != nil {
		return results, firstErr
	}
	return results, nil
}

// Upload uploads data to a file and returns the upload response.
// opts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) Upload(ctx context.Context, sandboxID string, path string, data []byte, opts ...*models.FileOperationOptions) (*models.UploadResponse, error) {
	return c.UploadFromReader(ctx, sandboxID, path, bytes.NewReader(data), int64(len(data)), nil, opts...)
}

// UploadWithProgress uploads data and reports progress via callback.
// The progress callback receives progress percentage 0-100.
// Token is used for SSE auth (EventSource does not support headers); typically pass API key.
func (c *Client) UploadWithProgress(ctx context.Context, sandboxID string, path string, data []byte, onProgress func(percent float64), token string, opts ...*models.FileOperationOptions) (*models.UploadResponse, error) {
	return c.UploadFromReader(ctx, sandboxID, path, bytes.NewReader(data), int64(len(data)), &UploadProgressOptions{
		OnProgress: onProgress,
		Token:      token,
	}, opts...)
}

// UploadProgressOptions configures upload progress reporting
type UploadProgressOptions struct {
	OnProgress func(percent float64)
	Token      string
}

// UploadFromReader uploads from a reader. If progressOpts.OnProgress is set, connects to SSE for progress.
// fileOpts is optional; pass FileOperationOptions with User to run as a specific user.
func (c *Client) UploadFromReader(ctx context.Context, sandboxID string, path string, r io.Reader, size int64, progressOpts *UploadProgressOptions, fileOpts ...*models.FileOperationOptions) (*models.UploadResponse, error) {
	remotePath := strings.TrimPrefix(path, "/")
	if remotePath != "" {
		remotePath = "/" + remotePath
	}

	sessionID := fmt.Sprintf("upload-%d-%d", time.Now().UnixNano(), time.Now().Nanosecond()%10000)
	extraHeaders := map[string]string{
		"X-Upload-Session": sessionID,
	}
	if size > 0 {
		extraHeaders["X-File-Size"] = strconv.FormatInt(size, 10)
	}

	if progressOpts != nil && progressOpts.OnProgress != nil && progressOpts.Token != "" && size > 0 {
		go c.streamUploadProgress(ctx, sandboxID, sessionID, progressOpts.Token, progressOpts.OnProgress)
	}

	apiPath := fmt.Sprintf("/v1/sandboxes/%s/files/upload", sandboxID)
	fields := map[string]string{"path": remotePath}
	if len(fileOpts) > 0 && fileOpts[0] != nil && fileOpts[0].User != "" {
		fields["user"] = fileOpts[0].User
	}
	filename := filepath.Base(path)
	if filename == "" || filename == "." {
		filename = "file"
	}
	fileFields := []client.MultipartField{
		{Name: filename, Reader: r, Size: size},
	}

	resp, err := c.baseClient.DoRequestMultipart(ctx, "POST", apiPath, fields, fileFields, extraHeaders)
	if err != nil {
		return nil, err
	}

	var uploadResp models.UploadResponse
	if err := c.baseClient.ParseResponse(resp, &uploadResp); err != nil {
		return nil, err
	}
	return &uploadResp, nil
}

// UploadResult is the result of an async upload
type UploadResult struct {
	Response *models.UploadResponse
	Error    error
}

// DownloadResult is the result of an async download
type DownloadResult struct {
	Data  []byte
	Error error
}

// UploadAsync performs upload in a goroutine and returns a channel with the result
func (c *Client) UploadAsync(ctx context.Context, sandboxID string, path string, data []byte) <-chan UploadResult {
	ch := make(chan UploadResult, 1)
	go func() {
		resp, err := c.Upload(ctx, sandboxID, path, data)
		ch <- UploadResult{Response: resp, Error: err}
		close(ch)
	}()
	return ch
}

// DownloadAsync performs download in a goroutine and returns a channel with the result
func (c *Client) DownloadAsync(ctx context.Context, sandboxID string, path string) <-chan DownloadResult {
	ch := make(chan DownloadResult, 1)
	go func() {
		data, err := c.Read(ctx, sandboxID, path)
		ch <- DownloadResult{Data: data, Error: err}
		close(ch)
	}()
	return ch
}

func (c *Client) streamUploadProgress(ctx context.Context, sandboxID string, sessionID string, token string, onProgress func(float64)) {
	apiPath := fmt.Sprintf("/v1/sandboxes/%s/files/upload-progress", sandboxID)
	q := make(map[string]string)
	q["session"] = sessionID
	q["token"] = token

	resp, err := c.baseClient.DoRequestRaw(ctx, "GET", apiPath, nil, nil, q)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			var evt struct {
				Progress float64 `json:"progress"`
			}
			if json.Unmarshal([]byte(line[6:]), &evt) == nil && evt.Progress > 0 {
				onProgress(evt.Progress)
			}
		}
	}
}

// GetMetrics retrieves metrics for a sandbox
func (c *Client) GetMetrics(ctx context.Context, sandboxID string, opts *models.GetSandboxMetricsOptions) (*models.SandboxMetricsResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/metrics", sandboxID)
	queryParams := make(map[string]string)

	if opts != nil {
		if opts.Start != nil {
			queryParams["start"] = formatTime(*opts.Start)
		}
		if opts.End != nil {
			queryParams["end"] = formatTime(*opts.End)
		}
		if opts.Step != nil {
			queryParams["step"] = strconv.Itoa(*opts.Step)
		}
	}

	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.SandboxMetricsResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// formatTime formats time for API query parameters
// Supports Unix timestamp, RFC3339, and simple datetime formats
func formatTime(t time.Time) string {
	// Try RFC3339 first (most common)
	return t.Format(time.RFC3339)
}
