//go:build e2e

package e2e

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/client"
)

// setupClient creates an SDK client for tests, reading config from environment variables
func setupClient(t *testing.T) *sandboxes.Client {
	baseURL := os.Getenv("SCALEBOX_BASE_URL")
	apiKey := os.Getenv("SCALEBOX_API_KEY")

	if baseURL == "" {
		t.Skip("跳过 E2E 测试: SCALEBOX_BASE_URL 环境变量未设置")
	}
	if apiKey == "" {
		t.Skip("跳过 E2E 测试: SCALEBOX_API_KEY 环境变量未设置")
	}

	baseClient := client.NewClient(baseURL, apiKey)
	return sandboxes.NewClient(baseClient)
}

// getTestTemplate returns the sandbox template name for E2E tests
// helm-deployment's push-public-template.sh only registers code-interpreter, so default is code-interpreter
// Can be overridden via SCALEBOX_TEMPLATE environment variable
func getTestTemplate() string {
	if tpl := os.Getenv("SCALEBOX_TEMPLATE"); tpl != "" {
		return tpl
	}
	return "code-interpreter"
}

// getRedisTemplate returns the redis template name for E2E tests
func getRedisTemplate() string {
	return "redis-e2e"
}

// getNekoWebRTCTemplate returns the neko-webrtc template name for E2E tests
func getNekoWebRTCTemplate() string {
	return "neko-webrtc-e2e"
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
