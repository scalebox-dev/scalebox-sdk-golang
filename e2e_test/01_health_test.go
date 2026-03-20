//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestHealth tests backend health check
func TestHealth(t *testing.T) {
	baseURL := requireEnv(t, "SCALEBOX_BASE_URL")
	apiKey := requireEnv(t, "SCALEBOX_API_KEY")

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
	projectID := requireProjectID(t)

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
