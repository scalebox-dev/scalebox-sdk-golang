//go:build e2e

package e2e

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestCodeInterpreterExecute tests code interpreter execution
func TestCodeInterpreterExecute(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:                "e2e-ci-" + strconv.FormatInt(time.Now().Unix(), 10),
		Template:            getTestTemplate(),
		ProjectID:           projectID,
		AllowInternetAccess: boolPtr(true),
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

	result, err := agent.CodeInterpreter().RunCode(ctx, "", "print('hello')", nil)
	if err != nil {
		t.Fatalf("RunCode failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(string(result.Stdout), "hello") {
		t.Errorf("Expected output to contain 'hello', got: %s", string(result.Stdout))
	}

	t.Logf("Code interpreter executed successfully: %s", string(result.Stdout))
}

func boolPtr(b bool) *bool {
	return &b
}
