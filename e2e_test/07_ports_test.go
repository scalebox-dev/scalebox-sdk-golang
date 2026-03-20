//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestDynamicPortSingle tests dynamic port for single container
func TestDynamicPortSingle(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-ps"),
		Template:  getTestTemplate(),
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 30*time.Second); err != nil {
		t.Fatalf("Sandbox did not reach running: %v", err)
	}

	_, err = sandboxClient.AddPort(ctx, sandbox.SandboxID, models.AddPortRequest{
		Port: 9999,
		Name: "e2e-test-port",
	})
	if err != nil {
		t.Fatalf("AddPort failed: %v", err)
	}

	t.Logf("Port 9999 added successfully")
}

// TestDynamicPortDual tests dynamic port for dual container (redis)
func TestDynamicPortDual(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()
	projectID := requireProjectID(t)

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-pd"),
		Template:  getRedisTemplate(),
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 10*time.Minute); err != nil {
		t.Fatalf("Sandbox did not reach running: %v", err)
	}

	_, err = sandboxClient.AddPort(ctx, sandbox.SandboxID, models.AddPortRequest{
		Port: 9999,
		Name: "e2e-test-port",
	})
	if err != nil {
		t.Fatalf("AddPort failed: %v", err)
	}

	t.Logf("Port 9999 added successfully")
}
