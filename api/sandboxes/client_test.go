package sandboxes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

// Helper function
func intPtr(i int) *int {
	return &i
}
