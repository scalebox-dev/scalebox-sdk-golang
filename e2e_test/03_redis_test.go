//go:build e2e

package e2e

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

const redisTemplateName = "redis-e2e"

// TestRedisTemplateImport imports redis-e2e template if not exists
func TestRedisTemplateImport(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// Simply try to create a sandbox with redis-e2e template to check if it exists
	// If template doesn't exist, the create will fail and we skip
	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	testSandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-rc"),
		Template:  redisTemplateName,
		ProjectID: projectID,
	})
	if err != nil {
		t.Skipf("redis-e2e template not available: %v", err)
	}
	defer sandboxClient.Delete(ctx, testSandbox.SandboxID, nil)

	t.Logf("redis-e2e template is available: %s", testSandbox.TemplateID)
}

// TestRedisConnect tests Redis TCP connection with TLS+SNI
func TestRedisConnect(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	projectID := os.Getenv("SCALEBOX_PROJECT_ID")
	if projectID == "" {
		projectID = "prj-e2e0000000000001"
	}

	sandbox, err := sandboxClient.Create(ctx, models.CreateSandboxRequest{
		Name:      shortSandboxName("e2e-rd"),
		Template:  redisTemplateName,
		ProjectID: projectID,
	})
	if err != nil {
		t.Fatalf("Create sandbox failed: %v", err)
	}
	defer sandboxClient.Delete(ctx, sandbox.SandboxID, nil)

	var domain string
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		status, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		if status.Status == "running" {
			sb, err := sandboxClient.Get(ctx, sandbox.SandboxID)
			if err == nil && sb.SandboxDomain != nil && *sb.SandboxDomain != "" {
				domain = *sb.SandboxDomain
			}
			break
		}
		if status.Status == "failed" {
			t.Fatalf("Sandbox failed to start")
		}
		time.Sleep(2 * time.Second)
	}

	if domain == "" {
		t.Fatalf("Could not get sandbox domain")
	}

	redisHost := fmt.Sprintf("6379-%s", domain)
	t.Logf("Connecting to Redis at %s:8443", redisHost)

	deadline = time.Now().Add(180 * time.Second)
	for time.Now().Before(deadline) {
		err := redisPing(redisHost)
		if err == nil {
			t.Logf("Redis PING successful!")
			return
		}
		t.Logf("Redis not ready yet: %v", err)
		time.Sleep(2 * time.Second)
	}

	t.Fatalf("Redis connection timeout after 180 seconds")
}

// redisPing performs Redis PING over TLS with SNI
func redisPing(host string) error {
	address := fmt.Sprintf("%s:8443", host)

	conn, err := tls.Dial("tcp", address, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return fmt.Errorf("TLS dial failed: %w", err)
	}
	defer conn.Close()

	_, err = conn.Write([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}

	response := string(buf[:n])
	if !strings.Contains(strings.ToUpper(response), "PONG") {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}
