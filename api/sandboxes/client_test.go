package sandboxes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

func TestCreate(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sandboxes" {
			t.Errorf("Expected /v1/sandboxes, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-KEY") == "" {
			t.Error("Expected X-API-KEY header")
		}

		// Read request body
		var req models.CreateSandboxRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		// Verify request
		if req.Name == "" {
			t.Error("Expected name in request")
		}

		// Return mock response
		sandbox := models.Sandbox{
			SandboxID: "sbx-test123",
			Name:      req.Name,
			Status:    "starting",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	// Create client
	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	// Test create
	req := models.CreateSandboxRequest{
		Name:      "Test Sandbox",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 10,
	}

	sandbox, err := sandboxClient.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if sandbox.SandboxID != "sbx-test123" {
		t.Errorf("Expected sandbox ID 'sbx-test123', got '%s'", sandbox.SandboxID)
	}
	if sandbox.Name != "Test Sandbox" {
		t.Errorf("Expected name 'Test Sandbox', got '%s'", sandbox.Name)
	}
}

func TestGet(t *testing.T) {
	sandboxID := "sbx-test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Name:      "Test Sandbox",
			Status:    "running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.Get(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if sandbox.SandboxID != sandboxID {
		t.Errorf("Expected sandbox ID '%s', got '%s'", sandboxID, sandbox.SandboxID)
	}
}

func TestList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/sandboxes" {
			t.Errorf("Expected /v1/sandboxes, got %s", r.URL.Path)
		}

		// Check query parameters
		if status := r.URL.Query().Get("status"); status != "running" {
			t.Errorf("Expected status=running, got status=%s", status)
		}

		response := models.SandboxListResponse{
			Sandboxes: []models.Sandbox{
				{
					SandboxID: "sbx-1",
					Name:      "Sandbox 1",
					Status:    "running",
				},
				{
					SandboxID: "sbx-2",
					Name:      "Sandbox 2",
					Status:    "running",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	opts := &models.ListSandboxesOptions{
		Status: "running",
	}

	result, err := sandboxClient.List(context.Background(), opts)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Sandboxes) != 2 {
		t.Errorf("Expected 2 sandboxes, got %d", len(result.Sandboxes))
	}
}

func TestDelete(t *testing.T) {
	sandboxID := "sbx-test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		response := models.DeletionResponse{
			SandboxID: sandboxID,
			Status:    "deletion_in_progress",
			Note:      "Deletion is being processed asynchronously",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.Delete(context.Background(), sandboxID, nil)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if result.SandboxID != sandboxID {
		t.Errorf("Expected sandbox ID '%s', got '%s'", sandboxID, result.SandboxID)
	}
	if result.Status != "deletion_in_progress" {
		t.Errorf("Expected status 'deletion_in_progress', got '%s'", result.Status)
	}
}

func TestPause(t *testing.T) {
	sandboxID := "sbx-test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/pause"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Name:      "Test Sandbox",
			Status:    "pausing",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.Pause(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}

	if sandbox.Status != "pausing" {
		t.Errorf("Expected status 'pausing', got '%s'", sandbox.Status)
	}
}

func TestResume(t *testing.T) {
	sandboxID := "sbx-test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/resume"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Name:      "Test Sandbox",
			Status:    "starting",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.Resume(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}

	if sandbox.Status != "starting" {
		t.Errorf("Expected status 'starting', got '%s'", sandbox.Status)
	}
}

func TestGetStatus(t *testing.T) {
	sandboxID := "sbx-test123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/status"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		status := models.SandboxStatus{
			SandboxID: sandboxID,
			Status:    "running",
			UpdatedAt: time.Now(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	status, err := sandboxClient.GetStatus(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status.SandboxID != sandboxID || status.Status != "running" {
		t.Errorf("Expected sandbox_id=%s status=running, got %s %s", sandboxID, status.SandboxID, status.Status)
	}
}

func TestUpdate(t *testing.T) {
	sandboxID := "sbx-test123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		var req models.UpdateSandboxRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Timeout != 600 {
			t.Errorf("Expected timeout 600, got %d", req.Timeout)
		}
		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Name:      "Test Sandbox",
			Status:    "running",
			Timeout:   600,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.Update(context.Background(), sandboxID, models.UpdateSandboxRequest{Timeout: 600})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if sandbox.SandboxID != sandboxID || sandbox.Timeout != 600 {
		t.Errorf("Expected sandbox_id=%s timeout=600, got %s timeout=%d", sandboxID, sandbox.SandboxID, sandbox.Timeout)
	}
}

func TestTerminate(t *testing.T) {
	sandboxID := "sbx-test123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/terminate"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		if r.URL.Query().Get("force") != "true" {
			t.Error("Expected force=true in query")
		}
		resp := models.TerminationResponse{
			SandboxID: sandboxID,
			Status:    "terminating",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	force := true
	result, err := sandboxClient.Terminate(context.Background(), sandboxID, &force)
	if err != nil {
		t.Fatalf("Terminate failed: %v", err)
	}
	if result.SandboxID != sandboxID || result.Status != "terminating" {
		t.Errorf("Expected sandbox_id=%s status=terminating, got %s %s", sandboxID, result.SandboxID, result.Status)
	}
}

func TestConnect(t *testing.T) {
	sandboxID := "sbx-test123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/connect"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Name:      "Test Sandbox",
			Status:    "running",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.Connect(context.Background(), sandboxID, nil)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if sandbox.SandboxID != sandboxID || sandbox.Status != "running" {
		t.Errorf("Expected sandbox_id=%s status=running, got %s %s", sandboxID, sandbox.SandboxID, sandbox.Status)
	}
}

func TestSetTimeout(t *testing.T) {
	sandboxID := "sbx-test123"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/timeout"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		var req models.SandboxTimeoutRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Timeout != 900 {
			t.Errorf("Expected timeout 900, got %d", req.Timeout)
		}
		sandbox := models.Sandbox{
			SandboxID: sandboxID,
			Status:    "running",
			Timeout:   900,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	sandbox, err := sandboxClient.SetTimeout(context.Background(), sandboxID, models.SandboxTimeoutRequest{Timeout: 900})
	if err != nil {
		t.Fatalf("SetTimeout failed: %v", err)
	}
	if sandbox.SandboxID != sandboxID || sandbox.Timeout != 900 {
		t.Errorf("Expected sandbox_id=%s timeout=900, got %s timeout=%d", sandboxID, sandbox.SandboxID, sandbox.Timeout)
	}
}

func TestGetMetrics(t *testing.T) {
	sandboxID := "sbx-test123"
	now := time.Now()
	start := now.Add(-5 * time.Minute)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		expectedPath := "/v1/sandboxes/" + sandboxID + "/metrics"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		// Check query parameters
		if r.URL.Query().Get("start") == "" {
			t.Error("Expected start parameter")
		}
		if r.URL.Query().Get("end") == "" {
			t.Error("Expected end parameter")
		}

		response := models.SandboxMetricsResponse{
			SandboxID:     sandboxID,
			Timestamp:     now,
			Status:        "running",
			UptimeSeconds: 300,
			Metrics: []models.MetricsDataPoint{
				{
					Timestamp:  start,
					CPUCount:   2,
					CPUUsedPct: 25.5,
					DiskTotal:  10737418240, // 10GB
					DiskUsed:   5368709120,  // 5GB
					MemTotal:   536870912,   // 512MB
					MemUsed:    268435456,   // 256MB
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	opts := &models.GetSandboxMetricsOptions{
		Start: &start,
		End:   &now,
		Step:  intPtr(5),
	}

	result, err := sandboxClient.GetMetrics(context.Background(), sandboxID, opts)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	if result.SandboxID != sandboxID {
		t.Errorf("Expected sandbox ID '%s', got '%s'", sandboxID, result.SandboxID)
	}
	if len(result.Metrics) != 1 {
		t.Errorf("Expected 1 metric data point, got %d", len(result.Metrics))
	}
}

func TestErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Sandbox not found",
		})
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d, got %d", http.StatusNotFound, apiErr.StatusCode)
	}
}

func TestListFiles(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/home/user"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/list", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}

		var req models.FileListRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Path != path {
			t.Errorf("Expected path %q, got %q", path, req.Path)
		}

		// Return wrapped response (backend format)
		entries := []models.EntryInfo{
			{Name: "file1.txt", Type: models.FileTypeFile, Path: "/home/user/file1.txt", Size: models.FlexibleInt64(100)},
			{Name: "dir1", Type: models.FileTypeDirectory, Path: "/home/user/dir1", Size: models.FlexibleInt64(0)},
		}
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"entries": entries},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.ListFiles(context.Background(), sandboxID, path, 1)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(result.Entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(result.Entries))
	}
	if result.Entries[0].Name != "file1.txt" || result.Entries[0].Type != models.FileTypeFile {
		t.Errorf("Expected first entry file1.txt (FILE), got %s (%d)", result.Entries[0].Name, result.Entries[0].Type)
	}
	if result.Entries[1].Name != "dir1" || result.Entries[1].Type != models.FileTypeDirectory {
		t.Errorf("Expected second entry dir1 (DIRECTORY), got %s (%d)", result.Entries[1].Name, result.Entries[1].Type)
	}
}

func TestStat(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/file.txt"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/stat", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		var req models.FileStatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Path != path {
			t.Errorf("Expected path %q, got %q", path, req.Path)
		}
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"entry": map[string]interface{}{"name": "file.txt", "type": "FILE_TYPE_FILE", "path": path, "size": 42}},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	info, err := sandboxClient.Stat(context.Background(), sandboxID, path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}
	if info.Name != "file.txt" || info.Path != path {
		t.Errorf("Expected name file.txt path %s, got %s %s", path, info.Name, info.Path)
	}
}

func TestExists(t *testing.T) {
	sandboxID := "sbx-test123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/stat", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		var req models.FileStatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Path == "/missing" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Not found"})
			return
		}
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"entry": map[string]interface{}{"name": "f", "path": "/f", "type": 1}},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	exists, err := sandboxClient.Exists(context.Background(), sandboxID, "/f")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Expected exists true for /f")
	}

	exists2, err2 := sandboxClient.Exists(context.Background(), sandboxID, "/missing")
	if err2 != nil {
		t.Fatalf("Exists (not found) failed: %v", err2)
	}
	if exists2 {
		t.Error("Expected exists false for missing path")
	}
}

func TestRead(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/hello.txt"
	content := []byte("hello world")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/download/workspace/hello.txt", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	data, err := sandboxClient.Read(context.Background(), sandboxID, path)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Expected content %q, got %q", content, data)
	}
}

func TestWrite(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/out.txt"
	content := []byte("write test")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/upload", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Error("Expected multipart/form-data")
		}
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"message": "File uploaded successfully", "sessionId": "s1"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	err := sandboxClient.Write(context.Background(), sandboxID, path, content)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
}

func TestWriteBatch(t *testing.T) {
	sandboxID := "sbx-test123"
	entries := []models.WriteEntry{
		{Path: "/workspace/a.txt", Data: bytes.NewReader([]byte("a")), ContentLength: 1},
		{Path: "/workspace/b.txt", Data: bytes.NewReader([]byte("bb")), ContentLength: 2},
	}

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/upload", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Error("Expected multipart/form-data")
		}
		callCount++
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"message": "File uploaded successfully", "sessionId": fmt.Sprintf("s%d", callCount)},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	results, err := sandboxClient.WriteBatch(context.Background(), sandboxID, entries)
	if err != nil {
		t.Fatalf("WriteBatch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}
	if results[0].Path != "/workspace/a.txt" || results[0].Name != "a.txt" {
		t.Errorf("Expected first result path=/workspace/a.txt name=a.txt, got path=%q name=%q", results[0].Path, results[0].Name)
	}
	if results[1].Path != "/workspace/b.txt" || results[1].Name != "b.txt" {
		t.Errorf("Expected second result path=/workspace/b.txt name=b.txt, got path=%q name=%q", results[1].Path, results[1].Name)
	}
	if callCount != 2 {
		t.Errorf("Expected 2 upload calls, got %d", callCount)
	}
}

func TestUpload(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/upload.txt"
	content := []byte("upload content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/upload", sandboxID)
		if r.URL.Path != expectedPath {
			t.Errorf("Expected %s, got %s", expectedPath, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Error("Expected multipart/form-data")
		}
		uploadResp := models.UploadResponse{Message: "uploaded", SessionID: "sess-1"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(uploadResp)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	resp, err := sandboxClient.Upload(context.Background(), sandboxID, path, content)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if resp.SessionID != "sess-1" {
		t.Errorf("Expected SessionID sess-1, got %s", resp.SessionID)
	}
}

func TestUploadWithProgress(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/progress.txt"
	content := []byte("progress content")
	progressCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/files/upload-progress") {
			// Progress SSE endpoint - return minimal response so goroutine can finish
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.URL.Path != fmt.Sprintf("/v1/sandboxes/%s/files/upload", sandboxID) {
			t.Errorf("Expected upload path, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		uploadResp := models.UploadResponse{Message: "uploaded", SessionID: "sess-2"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(uploadResp)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	resp, err := sandboxClient.UploadWithProgress(context.Background(), sandboxID, path, content, func(pct float64) {
		progressCalls++
	}, "test-token")
	if err != nil {
		t.Fatalf("UploadWithProgress failed: %v", err)
	}
	if resp.SessionID != "sess-2" {
		t.Errorf("Expected SessionID sess-2, got %s", resp.SessionID)
	}
}

func TestUploadAsync(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/async.txt"
	content := []byte("async content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uploadResp := models.UploadResponse{Message: "uploaded", SessionID: "sess-async"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(uploadResp)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	ch := sandboxClient.UploadAsync(context.Background(), sandboxID, path, content)
	result := <-ch
	if result.Error != nil {
		t.Fatalf("UploadAsync failed: %v", result.Error)
	}
	if result.Response == nil || result.Response.SessionID != "sess-async" {
		t.Errorf("Expected SessionID sess-async, got %v", result.Response)
	}
}

func TestDownloadAsync(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/async.txt"
	expectedData := []byte("async download content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Download uses GET /v1/sandboxes/{id}/files/download/{path}
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/download/workspace/async.txt", sandboxID)
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedData)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	ch := sandboxClient.DownloadAsync(context.Background(), sandboxID, path)
	result := <-ch
	if result.Error != nil {
		t.Fatalf("DownloadAsync failed: %v", result.Error)
	}
	if string(result.Data) != string(expectedData) {
		t.Errorf("Expected data %q, got %q", expectedData, result.Data)
	}
}

func TestReadStream(t *testing.T) {
	sandboxID := "sbx-test123"
	path := "/workspace/stream.txt"
	expectedData := []byte("stream content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := fmt.Sprintf("/v1/sandboxes/%s/files/download/workspace/stream.txt", sandboxID)
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(expectedData)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	rc, err := sandboxClient.ReadStream(context.Background(), sandboxID, path)
	if err != nil {
		t.Fatalf("ReadStream failed: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(data) != string(expectedData) {
		t.Errorf("Expected data %q, got %q", expectedData, data)
	}
}

func TestDownloadURL(t *testing.T) {
	domain := "sbx-xyz.example.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/sbx-1" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sandbox := models.Sandbox{
			SandboxID:       "sbx-1",
			SandboxDomain:   &domain,
			EnvdAccessToken: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	url, err := sandboxClient.DownloadURL(context.Background(), "sbx-1", "/home/user/file.txt", nil)
	if err != nil {
		t.Fatalf("DownloadURL failed: %v", err)
	}
	if !strings.Contains(url, "https://sbx-xyz.example.com/download/") {
		t.Errorf("expected download URL to contain base, got %s", url)
	}
	if !strings.Contains(url, "username=root") {
		t.Errorf("expected username=root, got %s", url)
	}
}

func TestUploadURL(t *testing.T) {
	domain := "sbx-xyz.example.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/sbx-1" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sandbox := models.Sandbox{
			SandboxID:       "sbx-1",
			SandboxDomain:   &domain,
			EnvdAccessToken: nil,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	url, err := sandboxClient.UploadURL(context.Background(), "sbx-1", "/home/user/", nil)
	if err != nil {
		t.Fatalf("UploadURL failed: %v", err)
	}
	if !strings.Contains(url, "https://sbx-xyz.example.com/upload") {
		t.Errorf("expected upload URL to contain base, got %s", url)
	}
}

func TestGetHost(t *testing.T) {
	domain := "sbx-host.example.com"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/sbx-1" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sandbox := models.Sandbox{
			SandboxID:     "sbx-1",
			SandboxDomain: &domain,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	host, err := sandboxClient.GetHost(context.Background(), "sbx-1", 0)
	if err != nil {
		t.Fatalf("GetHost failed: %v", err)
	}
	if host != "sbx-host.example.com" {
		t.Errorf("expected sbx-host.example.com, got %s", host)
	}

	host, err = sandboxClient.GetHost(context.Background(), "sbx-1", 8080)
	if err != nil {
		t.Fatalf("GetHost(8080) failed: %v", err)
	}
	if host != "sbx-host.example.com:8080" {
		t.Errorf("expected sbx-host.example.com:8080, got %s", host)
	}
}

func TestListWithPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("Expected page=2, got %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		response := models.SandboxListResponse{
			Sandboxes:  []models.Sandbox{{SandboxID: "sbx-1", Status: "running"}},
			Page:       2,
			Total:      25,
			TotalPages: 3,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.List(context.Background(), &models.ListSandboxesOptions{
		Page:  2,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if result.Page != 2 || result.Total != 25 || result.TotalPages != 3 {
		t.Errorf("Expected page=2 total=25 totalPages=3, got page=%d total=%d totalPages=%d",
			result.Page, result.Total, result.TotalPages)
	}
}

func TestBatchDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/batch-delete" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		response := models.BatchDeleteResponse{
			Total:      2,
			Successful: 2,
			Failed:     0,
			Results: []models.BatchOperationItem{
				{SandboxID: "sbx-1", Status: "success"},
				{SandboxID: "sbx-2", Status: "success"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.BatchDelete(context.Background(), models.BatchDeleteRequest{
		SandboxIDs: []string{"sbx-1", "sbx-2"},
	})
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}
	if result.Successful != 2 || len(result.Results) != 2 {
		t.Errorf("Expected 2 successful, got successful=%d results=%d", result.Successful, len(result.Results))
	}
}

func TestBatchTerminate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/batch-terminate" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		response := models.BatchTerminateResponse{
			Total:      2,
			Successful: 2,
			Failed:     0,
			Results: []models.BatchOperationItem{
				{SandboxID: "sbx-1", Status: "success"},
				{SandboxID: "sbx-2", Status: "success"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.BatchTerminate(context.Background(), models.BatchTerminateRequest{
		SandboxIDs: []string{"sbx-1", "sbx-2"},
	})
	if err != nil {
		t.Fatalf("BatchTerminate failed: %v", err)
	}
	if result.Successful != 2 || len(result.Results) != 2 {
		t.Errorf("Expected 2 successful, got successful=%d results=%d", result.Successful, len(result.Results))
	}
}

func TestCreateTemplateFromSandbox(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/sandboxes/" + sandboxID + "/create-template"
		if r.URL.Path != expectedPath || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		response := models.CreateTemplateResponse{
			TemplateID:   "tpl-123",
			TemplateName: "my-template",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	result, err := sandboxClient.CreateTemplateFromSandbox(context.Background(), sandboxID, models.CreateTemplateFromSandboxRequest{
		Name: "my-template",
	})
	if err != nil {
		t.Fatalf("CreateTemplateFromSandbox failed: %v", err)
	}
	if result.TemplateID != "tpl-123" || result.TemplateName != "my-template" {
		t.Errorf("expected tpl-123/my-template, got %s/%s", result.TemplateID, result.TemplateName)
	}
}

func TestGetPorts(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/sandboxes/" + sandboxID + "/ports"
		if r.URL.Path != expectedPath || r.Method != "GET" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		response := map[string]interface{}{
			"ports": []models.PortConfig{
				{Port: 8080, Name: "http"},
				{Port: 3000, Name: "app"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	ports, err := sandboxClient.GetPorts(context.Background(), sandboxID)
	if err != nil {
		t.Fatalf("GetPorts failed: %v", err)
	}
	if len(ports) != 2 || ports[0].Port != 8080 {
		t.Errorf("expected 2 ports, first 8080; got %d ports", len(ports))
	}
}

func TestAddPort(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/sandboxes/" + sandboxID + "/ports"
		if r.URL.Path != expectedPath || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		response := models.PortConfig{Port: 8080, Name: "http"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	port, err := sandboxClient.AddPort(context.Background(), sandboxID, models.AddPortRequest{
		Port: 8080,
		Name: "http",
	})
	if err != nil {
		t.Fatalf("AddPort failed: %v", err)
	}
	if port.Port != 8080 || port.Name != "http" {
		t.Errorf("expected port 8080 name http, got %d %s", port.Port, port.Name)
	}
}

func TestRemovePort(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/v1/sandboxes/" + sandboxID + "/ports/8080"
		if r.URL.Path != expectedPath || r.Method != "DELETE" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	err := sandboxClient.RemovePort(context.Background(), sandboxID, 8080)
	if err != nil {
		t.Fatalf("RemovePort failed: %v", err)
	}
}

func TestDownloadToFile(t *testing.T) {
	sandboxID := "sbx-1"
	content := []byte("streamed file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/files/download/") {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	tmp := t.TempDir()
	localPath := tmp + "/downloaded.txt"
	err := sandboxClient.DownloadToFile(context.Background(), sandboxID, "remote/file.txt", localPath)
	if err != nil {
		t.Fatalf("DownloadToFile failed: %v", err)
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected %q, got %q", content, data)
	}
}

func TestWriteBatchConcurrent(t *testing.T) {
	sandboxID := "sbx-1"
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/files/upload") {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		callCount++
		wrapped := map[string]interface{}{
			"success": true,
			"data":    map[string]interface{}{"message": "ok", "sessionId": fmt.Sprintf("s%d", callCount)},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(wrapped)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	entries := []models.WriteEntry{
		{Path: "/a.txt", Data: bytes.NewReader([]byte("a")), ContentLength: 1},
		{Path: "/b.txt", Data: bytes.NewReader([]byte("bb")), ContentLength: 2},
		{Path: "/c.txt", Data: bytes.NewReader([]byte("ccc")), ContentLength: 3},
	}
	results, err := sandboxClient.WriteBatchConcurrent(context.Background(), sandboxID, entries, 2)
	if err != nil {
		t.Fatalf("WriteBatchConcurrent failed: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
	if callCount != 3 {
		t.Errorf("expected 3 upload calls, got %d", callCount)
	}
}

func TestPauseWithIsAsync(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["is_async"] != true {
			t.Errorf("expected is_async=true in body, got %v", body)
		}
		sandbox := models.Sandbox{SandboxID: sandboxID, Status: "pausing"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.Pause(context.Background(), sandboxID, &models.PauseOptions{IsAsync: true})
	if err != nil {
		t.Fatalf("Pause with IsAsync failed: %v", err)
	}
}

func TestResumeWithIsAsync(t *testing.T) {
	sandboxID := "sbx-1"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["is_async"] != true {
			t.Errorf("expected is_async=true in body, got %v", body)
		}
		sandbox := models.Sandbox{SandboxID: sandboxID, Status: "starting"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer server.Close()

	baseClient := client.NewClient(server.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.Resume(context.Background(), sandboxID, &models.ResumeOptions{IsAsync: true})
	if err != nil {
		t.Fatalf("Resume with IsAsync failed: %v", err)
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}
