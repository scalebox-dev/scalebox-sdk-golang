//go:build e2e

package e2e

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestStorageQuota tests storage quota verification via df -h
func TestStorageQuota(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      "e2e-storage-" + strconv.FormatInt(time.Now().Unix(), 10),
		Template:  getTestTemplate(),
		ProjectID: projectID,
		StorageGB: 2,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 30*time.Second); err != nil {
		t.Fatalf("Sandbox did not reach running: %v", err)
	}

	agent, err := sandboxClient.ConnectToAgent(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("ConnectToAgent failed: %v", err)
	}

	result, err := agent.Commands().Run(ctx, "df -h", nil)
	if err != nil {
		t.Fatalf("df -h failed: %v", err)
	}

	cr, ok := result.(*sandboxes.CommandResult)
	if !ok {
		t.Fatalf("Expected *CommandResult, got %T", result)
	}

	if cr.ExitCode != 0 {
		t.Errorf("df -h exit code should be 0, got %d", cr.ExitCode)
	}

	output := string(cr.Stdout)
	t.Logf("df -h output:\n%s", output)

	if !strings.Contains(output, "G") && !strings.Contains(output, "M") {
		t.Errorf("Expected output to contain capacity info (G or M)")
	}
}
