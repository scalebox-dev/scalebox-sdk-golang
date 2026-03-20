//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestBatchSingleWithOSS tests batch single container with OSS
func TestBatchSingleWithOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	ossURI := os.Getenv("E2E_OSS_URI")
	ossAccessKey := os.Getenv("E2E_OSS_ACCESS_KEY")
	ossSecretKey := os.Getenv("E2E_OSS_SECRET_KEY")
	ossEndpoint := os.Getenv("E2E_OSS_ENDPOINT")
	ossRegion := os.Getenv("E2E_OSS_REGION")

	if ossURI == "" || ossAccessKey == "" || ossSecretKey == "" {
		t.Skip("OSS credentials not configured, skipping OSS test")
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

	// Write with timeout
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	testContent := "oss-test"
	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/mnt/oss/oss-test.txt", []byte(testContent))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// Read test
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/mnt/oss/oss-test.txt")
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s: got %s, want %s", id, string(data), testContent)
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

// TestBatchDualWithOSS tests batch dual container with OSS
func TestBatchDualWithOSS(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	ossURI := os.Getenv("E2E_OSS_URI")
	ossAccessKey := os.Getenv("E2E_OSS_ACCESS_KEY")
	ossSecretKey := os.Getenv("E2E_OSS_SECRET_KEY")
	ossEndpoint := os.Getenv("E2E_OSS_ENDPOINT")
	ossRegion := os.Getenv("E2E_OSS_REGION")

	if ossURI == "" || ossAccessKey == "" || ossSecretKey == "" {
		t.Skip("OSS credentials not configured, skipping OSS test")
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

	// Write with timeout
	writeCtx, writeCancel := context.WithTimeout(ctx, 60*time.Second)
	defer writeCancel()

	testContent := "oss-redis-test"
	for _, id := range sandboxIDs {
		err := sandboxClient.Write(writeCtx, id, "/mnt/oss/oss-test.txt", []byte(testContent))
		if err != nil {
			t.Fatalf("Write failed for sandbox %s: %v", id, err)
		}
	}

	// Read test
	for _, id := range sandboxIDs {
		data, err := sandboxClient.Read(ctx, id, "/mnt/oss/oss-test.txt")
		if err != nil {
			t.Fatalf("Read OSS file failed for sandbox %s: %v", id, err)
		}
		if string(data) != testContent {
			t.Errorf("OSS file content mismatch for sandbox %s", id)
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