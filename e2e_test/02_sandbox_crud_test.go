//go:build e2e

package e2e

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestSandboxLifecycle tests sandbox CRUD lifecycle: create → get → list → delete → get 404
func TestSandboxLifecycle(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	createReq := models.CreateSandboxRequest{
		Name:      "e2e-crud-" + strconv.FormatInt(time.Now().Unix(), 10),
		Template:  getTestTemplate(),
		ProjectID: projectID,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	t.Logf("Created sandbox: %s", sandbox.SandboxID)

	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	gotSandbox, err := sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("Get sandbox failed: %v", err)
	}
	if gotSandbox.SandboxID != sandbox.SandboxID {
		t.Errorf("Expected sandbox ID %s, got %s", sandbox.SandboxID, gotSandbox.SandboxID)
	}
	if gotSandbox.Status == "" {
		t.Error("Expected non-empty status")
	}

	result, err := sandboxClient.List(ctx, &models.ListSandboxesOptions{Limit: 100})
	if err != nil {
		t.Fatalf("List sandboxes failed: %v", err)
	}

	found := false
	for _, sb := range result.Sandboxes {
		if sb.SandboxID == sandbox.SandboxID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Created sandbox %s not found in list", sandbox.SandboxID)
	}

	_, err = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	if err != nil {
		t.Fatalf("Delete sandbox failed: %v", err)
	}

	for i := 0; i < 12; i++ {
		time.Sleep(2 * time.Second)
		_, err = sandboxClient.Get(ctx, sandbox.SandboxID)
		if err != nil && client.IsNotFound(err) {
			t.Logf("Sandbox %s properly deleted (got 404)", sandbox.SandboxID)
			return
		}
	}

	t.Error("Sandbox still exists after delete")
}

// TestPauseResume tests sandbox pause and resume
func TestPauseResume(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-pr"),
		Template:  getTestTemplate(),
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 30*time.Second); err != nil {
		t.Fatalf("Sandbox did not reach running state: %v", err)
	}

	pausedSandbox, err := sandboxClient.Pause(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
	t.Logf("Paused sandbox, status: %s", pausedSandbox.Status)

	for i := 0; i < 30; i++ {
		time.Sleep(2 * time.Second)
		status, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			continue
		}
		if status.Status == "paused" {
			t.Logf("Sandbox is now paused")
			break
		}
		if status.Status == "failed" || status.Status == "terminated" {
			t.Fatalf("Sandbox entered unexpected status: %s", status.Status)
		}
	}

	resumedSandbox, err := sandboxClient.Resume(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	t.Logf("Resumed sandbox, status: %s", resumedSandbox.Status)
}

// TestSetTimeout tests setting sandbox timeout
func TestSetTimeout(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      "e2e-timeout-" + strconv.FormatInt(time.Now().Unix(), 10),
		Template:  getTestTemplate(),
		ProjectID: projectID,
		Timeout:   300,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	newTimeout := 600
	updatedSandbox, err := sandboxClient.SetTimeout(ctx, sandbox.SandboxID, models.SandboxTimeoutRequest{Timeout: newTimeout})
	if err != nil {
		t.Fatalf("SetTimeout failed: %v", err)
	}

	if updatedSandbox.Timeout != newTimeout {
		t.Errorf("Expected timeout %d, got %d", newTimeout, updatedSandbox.Timeout)
	}

	time.Sleep(1 * time.Second)
	verifySandbox, err := sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("Get sandbox failed: %v", err)
	}
	if verifySandbox.Timeout != newTimeout {
		t.Errorf("Verification failed: expected timeout %d, got %d", newTimeout, verifySandbox.Timeout)
	}
}

// waitForSandboxRunning waits for sandbox to reach running state
func waitForSandboxRunning(sbClient *sandboxes.Client, ctx context.Context, sandboxID string, maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		status, err := sbClient.GetStatus(ctx, sandboxID)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if status.Status == "running" {
			return nil
		}
		if status.Status == "failed" || status.Status == "terminated" {
			return &timeoutError{status: status.Status}
		}
		time.Sleep(2 * time.Second)
	}
	return &timeoutError{status: "timeout"}
}

type timeoutError struct {
	status string
}

func (e *timeoutError) Error() string {
	return "sandbox did not reach running state: " + e.status
}
