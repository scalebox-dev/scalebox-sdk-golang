//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestHealth tests backend health check
func TestHealth(t *testing.T) {
	baseURL := os.Getenv("SCALEBOX_BASE_URL")
	apiKey := os.Getenv("SCALEBOX_API_KEY")

	if baseURL == "" {
		t.Skip("跳过 E2E 测试: SCALEBOX_BASE_URL 环境变量未设置")
	}
	if apiKey == "" {
		t.Skip("跳过 E2E 测试: SCALEBOX_API_KEY 环境变量未设置")
	}

	baseClient := client.NewClient(baseURL, apiKey)

	ctx := context.Background()
	if err := baseClient.Health(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}

// TestCodeInterpreterTemplateExists verifies code-interpreter template exists
func TestCodeInterpreterTemplateExists(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-ci"),
		Template:  "code-interpreter",
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("code-interpreter 模板不存在，请执行 helm-deployment/scripts/push-public-template.sh: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	t.Logf("code-interpreter template is available: %s", sandbox.TemplateID)
}
