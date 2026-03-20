//go:build e2e

package e2e

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// getBatchSize returns the batch size from environment variable or default
func getBatchSize() int {
	if size := os.Getenv("E2E_BATCH_SIZE"); size != "" {
		if i, err := strconv.Atoi(size); err == nil && i > 0 {
			return i
		}
	}
	return 2 // default
}

// TestBatchSingleNoOSS tests batch single container without OSS
func TestBatchSingleNoOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	batchSize := getBatchSize()

	sandboxIDs := make([]string, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		sb, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
			Name:      shortSandboxName("e2e-bs"),
			Template:  getTestTemplate(),
			ProjectID: projectID,
		})
		if err != nil {
			t.Fatalf("Create sandbox %d failed: %v", i, err)
		}
		sandboxIDs = append(sandboxIDs, sb.SandboxID)
		t.Logf("Created sandbox %d: %s", i, sb.SandboxID)
	}

	defer func() {
		for _, id := range sandboxIDs {
			sandboxClient.Delete(ctx, id, nil)
		}
	}()

	t.Logf("Waiting for %d sandboxes to be running...", batchSize)

	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running: %v", id, err)
		}
		t.Logf("Sandbox %s is running", id)
	}

	t.Logf("%d sandboxes are running", batchSize)

	// Simple write/read test with timeout
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/workspace/pause-resume-test.txt", []byte("pause-resume-verify-content"))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// Read test
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/pause-resume-test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "pause-resume-verify-content" {
			t.Errorf("File content mismatch for sandbox %s", id)
		}
	}

	resp, err := sandboxClient.BatchDelete(ctx, models.BatchDeleteRequest{
		SandboxIDs: sandboxIDs,
		Force:      boolPtr(true),
	})
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	t.Logf("BatchDelete: total=%d, successful=%d, failed=%d, results=%v", resp.Total, resp.Successful, resp.Failed, resp.Results)

	if resp.Successful != batchSize {
		t.Errorf("Expected %d successful deletions, got %d", batchSize, resp.Successful)
	}
}

// TestBatchDualNoOSS tests batch dual container without OSS
func TestBatchDualNoOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	batchSize := getBatchSize()

	sandboxIDs := make([]string, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		sb, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
			Name:      shortSandboxName("e2e-bd"),
			Template:  getRedisTemplate(),
			ProjectID: projectID,
		})
		if err != nil {
			t.Fatalf("Create sandbox %d failed: %v", i, err)
		}
		sandboxIDs = append(sandboxIDs, sb.SandboxID)
		t.Logf("Created redis sandbox %d: %s", i, sb.SandboxID)
	}

	defer func() {
		for _, id := range sandboxIDs {
			sandboxClient.Delete(ctx, id, nil)
		}
	}()

	t.Logf("Waiting for %d redis sandboxes to be running...", batchSize)

	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 10*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running: %v", id, err)
		}
		t.Logf("Redis sandbox %s is running", id)
	}

	t.Logf("%d redis sandboxes are running", batchSize)

	// Simple write/read test with timeout
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/workspace/test.txt", []byte("test-content"))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// Read test
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "test-content" {
			t.Errorf("File content mismatch for sandbox %s", id)
		}
	}

	resp, err := sandboxClient.BatchDelete(ctx, models.BatchDeleteRequest{
		SandboxIDs: sandboxIDs,
		Force:      boolPtr(true),
	})
	if err != nil {
		t.Fatalf("BatchDelete failed: %v", err)
	}

	t.Logf("BatchDelete: total=%d, successful=%d, failed=%d, results=%v", resp.Total, resp.Successful, resp.Failed, resp.Results)

	if resp.Successful != batchSize {
		t.Errorf("Expected %d successful deletions, got %d", batchSize, resp.Successful)
	}
}
