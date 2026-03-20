//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/client"
)

// requireEnv returns the value of an environment variable or fails the test
func requireEnv(t *testing.T, key string) string {
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// setupClient creates an SDK client for tests, reading config from environment variables
func setupClient(t *testing.T) *sandboxes.Client {
	baseURL := requireEnv(t, "SCALEBOX_BASE_URL")
	apiKey := requireEnv(t, "SCALEBOX_API_KEY")

	baseClient := client.NewClient(baseURL, apiKey)
	return sandboxes.NewClient(baseClient)
}

// requireProjectID returns the project ID from environment or fails the test
func requireProjectID(t *testing.T) string {
	return requireEnv(t, "SCALEBOX_PROJECT_ID")
}

// getTestTemplate returns the sandbox template name for E2E tests
func getTestTemplate() string {
	return os.Getenv("SCALEBOX_TEMPLATE")
}

// getRedisTemplate returns the redis template name for E2E tests
func getRedisTemplate() string {
	return os.Getenv("SCALEBOX_REDIS_TEMPLATE")
}

// getNekoWebRTCTemplate returns the neko-webrtc template name for E2E tests
func getNekoWebRTCTemplate() string {
	return os.Getenv("SCALEBOX_NEKO_TEMPLATE")
}

// shortSandboxName generates a unique sandbox name with ≤24 character limit (API requirement)
// Format: {prefix}-{timestamp}-{suffix}[:24]
func shortSandboxName(prefix string) string {
	ts := time.Now().Unix()
	suffix := time.Now().UnixNano() % 10000
	name := fmt.Sprintf("%s-%d-%d", prefix, ts, suffix)
	if len(name) > 24 {
		return name[:24]
	}
	return name
}

// shortTemplateName generates a template name with ≤32 character limit
// Format: e2e-{prefix}-{timestamp}[:32]
func shortTemplateName(prefix string) string {
	ts := time.Now().Unix()
	name := fmt.Sprintf("e2e-%s-%d", prefix, ts)
	if len(name) > 32 {
		return name[:32]
	}
	return name
}

// waitForSandboxStatus waits for a sandbox to reach the expected status
func waitForSandboxStatus(sbClient *sandboxes.Client, ctx context.Context, sandboxID string, expectedStatus string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, err := sbClient.GetStatus(ctx, sandboxID)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if status.Status == expectedStatus {
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("sandbox %s did not reach status %s within %v", sandboxID, expectedStatus, timeout)
}
