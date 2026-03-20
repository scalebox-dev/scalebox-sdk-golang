//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// TestWebRTC tests neko-webrtc-e2e template
func TestWebRTC(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      "e2e-webrtc-" + strconv.FormatInt(time.Now().Unix(), 10),
		Template:  getNekoWebRTCTemplate(),
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	if err := waitForSandboxRunning(sandboxClient, ctx, sandbox.SandboxID, 10*time.Minute); err != nil {
		t.Fatalf("Sandbox did not reach running: %v", err)
	}

	sb, err := sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("Get sandbox failed: %v", err)
	}

	if sb.SandboxDomain == nil || *sb.SandboxDomain == "" {
		t.Fatalf("Sandbox domain is empty")
	}

	domain := *sb.SandboxDomain
	t.Logf("Sandbox domain: %s", domain)

	ports, err := sandboxClient.GetPorts(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("GetPorts failed: %v", err)
	}

	if len(ports) == 0 {
		t.Errorf("Expected at least one port")
	}

	has8080 := false
	for _, p := range ports {
		if p.Port == 8080 {
			has8080 = true
			break
		}
	}

	if !has8080 {
		t.Logf("Port 8080 not found in ports list, checking ports: %+v", ports)
	}

	webURL := fmt.Sprintf("https://8080-%s", domain)
	t.Logf("Testing web interface at: %s", webURL)

	time.Sleep(5 * time.Second)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get("https://" + webURL)
	if err != nil {
		t.Logf("Web interface not yet reachable: %v", err)
	} else {
		resp.Body.Close()
		t.Logf("Web interface responded with status: %d", resp.StatusCode)
	}

	t.Logf("WebRTC test completed for sandbox: %s", sandbox.SandboxID)
}
