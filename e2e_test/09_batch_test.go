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
	projectID := requireProjectID(t)

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
		sandboxClient.BatchDelete(ctx, models.BatchDeleteRequest{
			SandboxIDs: sandboxIDs,
			Force:      boolPtr(true),
		})
	}()

	t.Logf("Waiting for %d sandboxes to be running...", batchSize)

	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running: %v", id, err)
		}
		t.Logf("Sandbox %s is running", id)
	}

	t.Logf("%d sandboxes are running", batchSize)

	// Write file to each sandbox
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/workspace/pause-resume-test.txt", []byte("pause-resume-verify-content"))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// First round: pause -> resume -> verify
	t.Logf("First round: batch pause")
	pauseResp, err := sandboxClient.BatchPause(ctx, models.BatchPauseRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchPause failed: %v", err)
	}
	t.Logf("BatchPause: total=%d, successful=%d, failed=%d", pauseResp.Total, pauseResp.Successful, pauseResp.Failed)

	// Wait for all sandboxes to be paused
	for _, id := range sandboxIDs {
		if err := waitForSandboxStatus(sandboxClient, ctx, id, "paused", 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach paused: %v", id, err)
		}
		t.Logf("Sandbox %s is paused", id)
	}

	t.Logf("First round: batch resume")
	resumeResp, err := sandboxClient.BatchResume(ctx, models.BatchResumeRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchResume failed: %v", err)
	}
	t.Logf("BatchResume: total=%d, successful=%d, failed=%d", resumeResp.Total, resumeResp.Successful, resumeResp.Failed)

	// Wait for all sandboxes to be running
	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running after resume: %v", id, err)
		}
		t.Logf("Sandbox %s is running after first resume", id)
	}

	// Verify file content after first resume
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/pause-resume-test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "pause-resume-verify-content" {
			t.Errorf("File content mismatch for sandbox %s after first resume", id)
		}
	}

	// Second round: pause -> resume -> verify
	t.Logf("Second round: batch pause")
	pauseResp, err = sandboxClient.BatchPause(ctx, models.BatchPauseRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchPause failed: %v", err)
	}
	t.Logf("BatchPause: total=%d, successful=%d, failed=%d", pauseResp.Total, pauseResp.Successful, pauseResp.Failed)

	// Wait for all sandboxes to be paused
	for _, id := range sandboxIDs {
		if err := waitForSandboxStatus(sandboxClient, ctx, id, "paused", 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach paused: %v", id, err)
		}
		t.Logf("Sandbox %s is paused", id)
	}

	t.Logf("Second round: batch resume")
	resumeResp, err = sandboxClient.BatchResume(ctx, models.BatchResumeRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchResume failed: %v", err)
	}
	t.Logf("BatchResume: total=%d, successful=%d, failed=%d", resumeResp.Total, resumeResp.Successful, resumeResp.Failed)

	// Wait for all sandboxes to be running
	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running after resume: %v", id, err)
		}
		t.Logf("Sandbox %s is running after second resume", id)
	}

	// Verify file content after second resume
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/pause-resume-test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "pause-resume-verify-content" {
			t.Errorf("File content mismatch for sandbox %s after second resume", id)
		}
	}

	// Batch delete
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

	projectID := requireProjectID(t)

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
		sandboxClient.BatchDelete(ctx, models.BatchDeleteRequest{
			SandboxIDs: sandboxIDs,
			Force:      boolPtr(true),
		})
	}()

	t.Logf("Waiting for %d redis sandboxes to be running...", batchSize)

	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 10*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running: %v", id, err)
		}
		t.Logf("Redis sandbox %s is running", id)
	}

	t.Logf("%d redis sandboxes are running", batchSize)

	// Write file to each sandbox
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/workspace/test.txt", []byte("test-content"))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// First round: pause -> resume -> verify
	t.Logf("First round: batch pause")
	pauseResp, err := sandboxClient.BatchPause(ctx, models.BatchPauseRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchPause failed: %v", err)
	}
	t.Logf("BatchPause: total=%d, successful=%d, failed=%d", pauseResp.Total, pauseResp.Successful, pauseResp.Failed)

	// Wait for all sandboxes to be paused
	for _, id := range sandboxIDs {
		if err := waitForSandboxStatus(sandboxClient, ctx, id, "paused", 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach paused: %v", id, err)
		}
		t.Logf("Sandbox %s is paused", id)
	}

	t.Logf("First round: batch resume")
	resumeResp, err := sandboxClient.BatchResume(ctx, models.BatchResumeRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchResume failed: %v", err)
	}
	t.Logf("BatchResume: total=%d, successful=%d, failed=%d", resumeResp.Total, resumeResp.Successful, resumeResp.Failed)

	// Wait for all sandboxes to be running
	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running after resume: %v", id, err)
		}
		t.Logf("Sandbox %s is running after first resume", id)
	}

	// Verify file content after first resume
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "test-content" {
			t.Errorf("File content mismatch for sandbox %s after first resume", id)
		}
	}

	// Second round: pause -> resume -> verify
	t.Logf("Second round: batch pause")
	pauseResp, err = sandboxClient.BatchPause(ctx, models.BatchPauseRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchPause failed: %v", err)
	}
	t.Logf("BatchPause: total=%d, successful=%d, failed=%d", pauseResp.Total, pauseResp.Successful, pauseResp.Failed)

	// Wait for all sandboxes to be paused
	for _, id := range sandboxIDs {
		if err := waitForSandboxStatus(sandboxClient, ctx, id, "paused", 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach paused: %v", id, err)
		}
		t.Logf("Sandbox %s is paused", id)
	}

	t.Logf("Second round: batch resume")
	resumeResp, err = sandboxClient.BatchResume(ctx, models.BatchResumeRequest{
		SandboxIDs: sandboxIDs,
	})
	if err != nil {
		t.Fatalf("BatchResume failed: %v", err)
	}
	t.Logf("BatchResume: total=%d, successful=%d, failed=%d", resumeResp.Total, resumeResp.Successful, resumeResp.Failed)

	// Wait for all sandboxes to be running
	for _, id := range sandboxIDs {
		if err := waitForSandboxRunning(sandboxClient, ctx, id, 5*time.Minute); err != nil {
			t.Fatalf("Sandbox %s did not reach running after resume: %v", id, err)
		}
		t.Logf("Sandbox %s is running after second resume", id)
	}

	// Verify file content after second resume
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/workspace/test.txt")
		if err != nil {
			t.Fatalf("Read failed for sandbox %s: %v", id, err)
		}
		if string(data) != "test-content" {
			t.Errorf("File content mismatch for sandbox %s after second resume", id)
		}
	}

	// Batch delete
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
