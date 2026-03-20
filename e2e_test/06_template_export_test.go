//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestTemplateExportImportSingle tests template export/import for single container
func TestTemplateExportImportSingle(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-ts"),
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

	templateName := shortTemplateName("export-single")

	exportResp, err := sandboxClient.CreateTemplateFromSandbox(ctx, sandbox.SandboxID, models.CreateTemplateFromSandboxRequest{
		Name:        templateName,
		Description: "E2E exported template",
	})
	if err != nil {
		t.Fatalf("CreateTemplateFromSandbox failed: %v", err)
	}

	t.Logf("Template exported: %s", exportResp.TemplateID)

	defer sandboxClient.DeleteTemplate(ctx, exportResp.TemplateID)

	// Poll for template to become available (can take several minutes)
	pollCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	templateAvailable := false
	for !templateAvailable {
		select {
		case <-pollCtx.Done():
			t.Fatalf("Template polling timed out after 15 minutes")
		default:
			pollCtx2, cancel2 := context.WithTimeout(ctx, 30*time.Second)
			template, err := sandboxClient.GetTemplate(pollCtx2, exportResp.TemplateID)
			cancel2()
			if err != nil {
				t.Logf("GetTemplate error (retrying): %v", err)
				time.Sleep(10 * time.Second)
				continue
			}
			if template.Status == "available" {
				t.Logf("Template is now available")
				templateAvailable = true
				break
			}
			if template.Status == "failed" {
				t.Fatalf("Template build failed")
			}
			t.Logf("Template status: %s, waiting...", template.Status)
			time.Sleep(10 * time.Second)
		}
	}

	newSandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-ft"),
		Template:  exportResp.TemplateID,
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox from template failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, newSandbox.SandboxID, nil)

	if newSandbox.TemplateID != exportResp.TemplateID {
		t.Errorf("Expected template ID %s, got %s", exportResp.TemplateID, newSandbox.TemplateID)
	}

	t.Logf("Sandbox created from exported template: %s", newSandbox.SandboxID)
}

// TestTemplateExportImportDual tests template export/import for dual container (redis)
func TestTemplateExportImportDual(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-rd"),
		Template:  getRedisTemplate(),
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create redis sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 10*time.Minute); err != nil {
		t.Fatalf("Sandbox did not reach running: %v", err)
	}

	templateName := shortTemplateName("export-redis")

	exportResp, err := sandboxClient.CreateTemplateFromSandbox(ctx, sandbox.SandboxID, models.CreateTemplateFromSandboxRequest{
		Name:        templateName,
		Description: "E2E exported redis template",
	})
	if err != nil {
		t.Fatalf("CreateTemplateFromSandbox failed: %v", err)
	}

	t.Logf("Redis template exported: %s", exportResp.TemplateID)

	defer sandboxClient.DeleteTemplate(ctx, exportResp.TemplateID)

	// Poll for template to become available (can take several minutes)
	pollCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	templateAvailable := false
	for !templateAvailable {
		select {
		case <-pollCtx.Done():
			t.Fatalf("Template polling timed out after 15 minutes")
		default:
			pollCtx2, cancel2 := context.WithTimeout(ctx, 30*time.Second)
			template, err := sandboxClient.GetTemplate(pollCtx2, exportResp.TemplateID)
			cancel2()
			if err != nil {
				t.Logf("GetTemplate error (retrying): %v", err)
				time.Sleep(10 * time.Second)
				continue
			}
			if template.Status == "available" {
				t.Logf("Template is now available")
				templateAvailable = true
				break
			}
			if template.Status == "failed" {
				t.Fatalf("Template build failed")
			}
			t.Logf("Template status: %s, waiting...", template.Status)
			time.Sleep(10 * time.Second)
		}
	}

	newSandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-fr"),
		Template:  exportResp.TemplateID,
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox from template failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, newSandbox.SandboxID, nil)

	t.Logf("Sandbox created from exported redis template: %s", newSandbox.SandboxID)
}
