//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// batchOSSFile generates an OSS file path for a sandbox
func batchOSSFile(sandboxID string) string {
	return fmt.Sprintf("/mnt/oss/%s-oss-test.txt", sandboxID)
}

// TestBatchSingleWithOSS tests batch single container with OSS
func TestBatchSingleWithOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	ossURI := os.Getenv("E2E_OSS_URI")
	ossAccessKey := os.Getenv("E2E_OSS_ACCESS_KEY")
	ossSecretKey := os.Getenv("E2E_OSS_SECRET_KEY")
	ossEndpoint := os.Getenv("E2E_OSS_ENDPOINT")
	ossRegion := os.Getenv("E2E_OSS_REGION")

	if ossURI == "" || ossAccessKey == "" || ossSecretKey == "" {
		t.Fatalf("Required environment variable E2E_OSS_URI/E2E_OSS_ACCESS_KEY/E2E_OSS_SECRET_KEY is not set")
	}

	batchSize := getBatchSize()

	sandboxIDs := make([]string, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		sb, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
			Name:                shortSandboxName("e2e-bs"),
			Template:            getTestTemplate(),
			ProjectID:           projectID,
			AllowInternetAccess: boolPtr(true),
			ObjectStorage: &models.ObjectStorageConfig{
				URI:        ossURI,
				MountPoint: "/mnt/oss",
				AccessKey:  ossAccessKey,
				SecretKey:  ossSecretKey,
				Endpoint:   ossEndpoint,
				Region:     ossRegion,
			},
		})
		if err != nil {
			t.Fatalf("Create sandbox %d failed: %v", i, err)
		}
		sandboxIDs = append(sandboxIDs, sb.SandboxID)
		t.Logf("Created sandbox with OSS %d: %s", i, sb.SandboxID)
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

	// Write OSS file to each sandbox
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	testContent := "oss-pause-resume-verify"
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		err := sandboxClient.Write(writeCtx, id, ossPath, []byte(testContent))
		if err != nil {
			t.Fatalf("Write OSS failed for sandbox %s: %v", id, err)
		}
		t.Logf("Wrote OSS file %s to sandbox %s", ossPath, id)
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

	// Verify OSS file content after first resume
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		data, err := sandboxClient.Read(ctx, id, ossPath)
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s after first resume: got %s, want %s", id, string(data), testContent)
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

	// Verify OSS file content after second resume
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		data, err := sandboxClient.Read(ctx, id, ossPath)
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s after second resume: got %s, want %s", id, string(data), testContent)
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

// TestBatchDualWithOSS tests batch dual container with OSS
func TestBatchDualWithOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := requireProjectID(t)

	ossURI := os.Getenv("E2E_OSS_URI")
	ossAccessKey := os.Getenv("E2E_OSS_ACCESS_KEY")
	ossSecretKey := os.Getenv("E2E_OSS_SECRET_KEY")
	ossEndpoint := os.Getenv("E2E_OSS_ENDPOINT")
	ossRegion := os.Getenv("E2E_OSS_REGION")

	if ossURI == "" || ossAccessKey == "" || ossSecretKey == "" {
		t.Fatalf("Required environment variable E2E_OSS_URI/E2E_OSS_ACCESS_KEY/E2E_OSS_SECRET_KEY is not set")
	}

	batchSize := getBatchSize()

	sandboxIDs := make([]string, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		sb, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
			Name:                shortSandboxName("e2e-bd"),
			Template:            getRedisTemplate(),
			ProjectID:           projectID,
			AllowInternetAccess: boolPtr(true),
			ObjectStorage: &models.ObjectStorageConfig{
				URI:        ossURI,
				MountPoint: "/mnt/oss",
				AccessKey:  ossAccessKey,
				SecretKey:  ossSecretKey,
				Endpoint:   ossEndpoint,
				Region:     ossRegion,
			},
		})
		if err != nil {
			t.Fatalf("Create sandbox %d failed: %v", i, err)
		}
		sandboxIDs = append(sandboxIDs, sb.SandboxID)
		t.Logf("Created redis sandbox with OSS %d: %s", i, sb.SandboxID)
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

	// Write OSS file to each sandbox
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	testContent := "oss-pause-resume-verify"
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		err := sandboxClient.Write(writeCtx, id, ossPath, []byte(testContent))
		if err != nil {
			t.Fatalf("Write OSS failed for sandbox %s: %v", id, err)
		}
		t.Logf("Wrote OSS file %s to sandbox %s", ossPath, id)
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

	// Verify OSS file content after first resume
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		data, err := sandboxClient.Read(ctx, id, ossPath)
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s after first resume: got %s, want %s", id, string(data), testContent)
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

	// Verify OSS file content after second resume
	for _, id := range sandboxIDs {
		ossPath := batchOSSFile(id)
		data, err := sandboxClient.Read(ctx, id, ossPath)
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s after second resume: got %s, want %s", id, string(data), testContent)
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