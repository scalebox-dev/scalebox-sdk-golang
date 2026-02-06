//go:build integration

package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/api/sandboxes"
	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// setupClient 创建测试客户端，从环境变量读取配置
func setupClient(t *testing.T) *sandboxes.Client {
	baseURL := os.Getenv("SCALEBOX_BASE_URL")
	apiKey := os.Getenv("SCALEBOX_API_KEY")

	if baseURL == "" {
		t.Skip("跳过集成测试: SCALEBOX_BASE_URL 环境变量未设置")
	}
	if apiKey == "" {
		t.Skip("跳过集成测试: SCALEBOX_API_KEY 环境变量未设置")
	}

	baseClient := client.NewClient(baseURL, apiKey)
	return sandboxes.NewClient(baseClient)
}

// TestIntegrationCreateSandbox 测试创建沙箱
func TestIntegrationCreateSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "integration-test-sandbox",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2, // 不超过订阅计划限制（常见为 2GB）
		Timeout:   300,
		Metadata: map[string]string{
			"environment": "integration-test",
			"test":        "true",
		},
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}

	if sandbox.SandboxID == "" {
		t.Error("沙箱 ID 不应为空")
	}
	// 后端会在名称后追加随机后缀，故只校验非空且以请求名开头
	if sandbox.Name == "" {
		t.Error("沙箱名称不应为空")
	}
	if createReq.Name != "" && !strings.HasPrefix(sandbox.Name, createReq.Name) {
		t.Errorf("沙箱名称应以 '%s' 开头, 得到 '%s'", createReq.Name, sandbox.Name)
	}
	if sandbox.Status == "" {
		t.Error("沙箱状态不应为空")
	}

	t.Logf("创建成功! Sandbox ID: %s, 状态: %s", sandbox.SandboxID, sandbox.Status)

	// 清理：删除创建的沙箱
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()
}

// TestIntegrationGetSandbox 测试获取沙箱详情
func TestIntegrationGetSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "get-detail-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	// 测试获取详情
	gotSandbox, err := sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取沙箱详情失败: %v", err)
	}

	if gotSandbox.SandboxID != sandbox.SandboxID {
		t.Errorf("期望 Sandbox ID '%s', 得到 '%s'", sandbox.SandboxID, gotSandbox.SandboxID)
	}
	// 后端返回的名称带随机后缀，只需与创建返回的 name 一致
	if gotSandbox.Name != sandbox.Name {
		t.Errorf("期望沙箱名称 '%s', 得到 '%s'", sandbox.Name, gotSandbox.Name)
	}

	t.Logf("沙箱名称: %s", gotSandbox.Name)
	t.Logf("状态: %s", gotSandbox.Status)
	t.Logf("CPU: %d 核心", gotSandbox.CPUCount)
	t.Logf("内存: %d MB", gotSandbox.MemoryMB)
}

// TestIntegrationListSandboxes 测试列出沙箱
func TestIntegrationListSandboxes(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个测试沙箱，确保有数据可测试
	createReq := models.CreateSandboxRequest{
		Name:      "list-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	testSandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, testSandbox.SandboxID, nil)
	}()

	// 等待一小段时间，确保沙箱状态已更新
	time.Sleep(2 * time.Second)

	// 列出所有沙箱（不限制状态）
	listOpts := &models.ListSandboxesOptions{
		Limit: 10,
	}

	result, err := sandboxClient.List(ctx, listOpts)
	if err != nil {
		t.Fatalf("列出沙箱失败: %v", err)
	}

	if result.Sandboxes == nil {
		t.Error("沙箱列表不应为 nil")
	}

	t.Logf("找到 %d 个沙箱", len(result.Sandboxes))

	// 验证创建的测试沙箱在列表中
	found := false
	for _, sb := range result.Sandboxes {
		if sb.SandboxID == testSandbox.SandboxID {
			found = true
			t.Logf("✓ 找到测试沙箱: %s (%s) - %s", sb.SandboxID, sb.Name, sb.Status)
			break
		}
	}

	if !found {
		t.Errorf("创建的测试沙箱（ID: %s）应出现在沙箱列表中", testSandbox.SandboxID)
	}
}

// TestIntegrationGetSandboxStatus 测试获取沙箱状态
func TestIntegrationGetSandboxStatus(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "status-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	status, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取状态失败: %v", err)
	}

	if status.Status == "" {
		t.Error("状态不应为空")
	}

	// 打印完整的状态信息
	t.Logf("沙箱状态信息:")
	t.Logf("  Sandbox ID: %s", status.SandboxID)
	t.Logf("  状态: %s", status.Status)
	if status.Substatus != nil {
		t.Logf("  子状态: %s", *status.Substatus)
	}
	if status.Reason != nil {
		t.Logf("  原因: %s", *status.Reason)
	}
	t.Logf("  更新时间: %s", status.UpdatedAt.Format(time.RFC3339))
}

// TestIntegrationGetSandboxMetrics 测试获取沙箱指标
func TestIntegrationGetSandboxMetrics(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "metrics-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	// 等待沙箱启动并生成一些指标数据
	time.Sleep(5 * time.Second)

	// 查询最近1分钟的指标数据（沙箱刚创建，数据时间范围较短）
	// 如果查询5分钟前的数据，沙箱刚创建时肯定没有数据
	end := time.Now()
	start := end.Add(-1 * time.Minute) // 查询过去1分钟的数据
	step := 5                          // 5秒间隔
	metricsOpts := &models.GetSandboxMetricsOptions{
		Start: &start,
		End:   &end,
		Step:  &step,
	}

	metrics, err := sandboxClient.GetMetrics(ctx, sandbox.SandboxID, metricsOpts)
	if err != nil {
		t.Fatalf("获取指标失败: %v", err)
	}

	if metrics.SandboxID != sandbox.SandboxID {
		t.Errorf("期望 Sandbox ID '%s', 得到 '%s'", sandbox.SandboxID, metrics.SandboxID)
	}

	// 打印指标响应信息
	t.Logf("指标响应信息:")
	t.Logf("  Sandbox ID: %s", metrics.SandboxID)
	t.Logf("  状态: %s", metrics.Status)
	t.Logf("  运行时长: %d 秒", metrics.UptimeSeconds)
	t.Logf("  响应时间戳: %s", metrics.Timestamp.Format(time.RFC3339))
	t.Logf("  指标数据点数量: %d", len(metrics.Metrics))

	if len(metrics.Metrics) > 0 {
		t.Logf("  指标数据详情:")
		// 打印最新的几个数据点
		startIdx := 0
		if len(metrics.Metrics) > 3 {
			startIdx = len(metrics.Metrics) - 3 // 只打印最后3个
		}
		for i := startIdx; i < len(metrics.Metrics); i++ {
			point := metrics.Metrics[i]
			t.Logf("    数据点 %d:", i+1)
			t.Logf("      时间戳: %s", point.Timestamp.Format(time.RFC3339))
			t.Logf("      CPU: %.2f%% (请求: %d 核心)", point.CPUUsedPct, point.CPUCount)
			t.Logf("      内存: %d / %d MB (%.2f%%)",
				point.MemUsed/1024/1024,
				point.MemTotal/1024/1024,
				float64(point.MemUsed)/float64(point.MemTotal)*100)
			t.Logf("      磁盘: %d / %d MB (%.2f%%)",
				point.DiskUsed/1024/1024,
				point.DiskTotal/1024/1024,
				float64(point.DiskUsed)/float64(point.DiskTotal)*100)
		}
	} else {
		t.Logf("  注意: 暂无指标数据点（可能沙箱刚创建，指标数据尚未生成）")
	}
}

// TestIntegrationPauseSandbox 测试暂停沙箱
func TestIntegrationPauseSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "pause-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	// 定时检查沙箱状态，直到状态变为 running
	// 暂停操作需要 DaemonSet 保护可写层，只有 running 状态的沙箱才能暂停
	maxWaitTime := 30 * time.Second
	checkInterval := 2 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	t.Logf("等待沙箱状态变为 running...")
	for time.Now().Before(deadline) {
		sandboxStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			t.Fatalf("获取沙箱状态失败: %v", err)
		}

		if sandboxStatus.Status == "running" {
			t.Logf("沙箱状态已变为 running，可以执行暂停操作")
			break
		}

		if sandboxStatus.Status == "failed" || sandboxStatus.Status == "terminated" {
			t.Fatalf("沙箱状态异常（%s），无法执行暂停操作", sandboxStatus.Status)
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
		t.Logf("当前状态: %s，继续等待...", sandboxStatus.Status)
	}

	// 最终检查：确保状态是 running
	finalStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取沙箱状态失败: %v", err)
	}
	if finalStatus.Status != "running" {
		t.Fatalf("沙箱在 %v 内未达到 running 状态（当前: %s），测试失败", maxWaitTime, finalStatus.Status)
	}

	// 调用暂停 API（异步操作，立即返回）
	pausedSandbox, err := sandboxClient.Pause(ctx, sandbox.SandboxID)
	if err != nil {
		// 检查错误类型，提供更详细的错误信息
		statusCode := client.StatusCode(err)
		switch statusCode {
		case 404:
			t.Skip("沙箱不存在，跳过测试")
		case 403:
			t.Skip("沙箱不支持暂停操作，跳过测试")
		case 500:
			// 500 错误通常是后端 Kubernetes 基础设施问题
			// 可能原因：DaemonSet 未正常运行、网络问题、权限问题、资源不足等
			t.Fatalf("暂停沙箱失败（后端错误 %d）: %v\n"+
				"注意：暂停操作需要 Kubernetes DaemonSet 保护可写层，"+
				"如果 DaemonSet 在 30 秒内未完成保护操作会失败。", statusCode, err)
		default:
			t.Fatalf("暂停沙箱失败（状态码 %d）: %v", statusCode, err)
		}
	}

	if pausedSandbox.Status == "" {
		t.Error("沙箱状态不应为空")
	}

	t.Logf("暂停请求已提交，当前沙箱状态: %s", pausedSandbox.Status)
	t.Logf("注意：暂停操作是异步的，实际暂停可能需要一些时间完成")
}

// TestIntegrationResumeSandbox 测试恢复沙箱
func TestIntegrationResumeSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "resume-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	// 定时检查沙箱状态，直到状态变为 running
	maxWaitTime := 30 * time.Second
	checkInterval := 2 * time.Second
	deadline := time.Now().Add(maxWaitTime)

	t.Logf("等待沙箱状态变为 running...")
	for time.Now().Before(deadline) {
		sandboxStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			t.Fatalf("获取沙箱状态失败: %v", err)
		}

		if sandboxStatus.Status == "running" {
			t.Logf("沙箱状态已变为 running，可以执行暂停操作")
			break
		}

		if sandboxStatus.Status == "failed" || sandboxStatus.Status == "terminated" {
			t.Fatalf("沙箱状态异常（%s），无法执行暂停操作", sandboxStatus.Status)
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
		t.Logf("当前状态: %s，继续等待...", sandboxStatus.Status)
	}

	// 最终检查：确保状态是 running
	finalStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取沙箱状态失败: %v", err)
	}
	if finalStatus.Status != "running" {
		t.Fatalf("沙箱在 %v 内未达到 running 状态（当前: %s），测试失败", maxWaitTime, finalStatus.Status)
	}

	// 先暂停沙箱
	pausedSandbox, err := sandboxClient.Pause(ctx, sandbox.SandboxID)
	if err != nil {
		statusCode := client.StatusCode(err)
		switch statusCode {
		case 404:
			t.Skip("沙箱不存在，跳过测试")
		case 403:
			t.Skip("沙箱不支持暂停操作，跳过测试")
		case 500:
			t.Fatalf("暂停沙箱失败（后端错误 %d）: %v\n"+
				"注意：暂停操作需要 Kubernetes DaemonSet 保护可写层，"+
				"如果 DaemonSet 在 30 秒内未完成保护操作会失败。", statusCode, err)
		default:
			t.Fatalf("暂停沙箱失败（状态码 %d）: %v", statusCode, err)
		}
	}
	t.Logf("暂停请求已提交，当前沙箱状态: %s", pausedSandbox.Status)

	// 等待沙箱状态变为 paused
	deadline = time.Now().Add(maxWaitTime)
	t.Logf("等待沙箱状态变为 paused...")
	for time.Now().Before(deadline) {
		sandboxStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			t.Fatalf("获取沙箱状态失败: %v", err)
		}

		if sandboxStatus.Status == "paused" {
			t.Logf("沙箱状态已变为 paused，可以执行恢复操作")
			break
		}

		if sandboxStatus.Status == "failed" || sandboxStatus.Status == "terminated" {
			t.Fatalf("沙箱状态异常（%s），无法执行恢复操作", sandboxStatus.Status)
		}

		// 等待一段时间后再次检查
		time.Sleep(checkInterval)
		t.Logf("当前状态: %s，继续等待...", sandboxStatus.Status)
	}

	// 最终检查：确保状态是 paused
	pausedStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取沙箱状态失败: %v", err)
	}
	if pausedStatus.Status != "paused" {
		t.Fatalf("沙箱在 %v 内未达到 paused 状态（当前: %s），测试失败", maxWaitTime, pausedStatus.Status)
	}

	// 调用恢复 API（异步操作，立即返回）
	resumedSandbox, err := sandboxClient.Resume(ctx, sandbox.SandboxID)
	if err != nil {
		// 检查错误类型，提供更详细的错误信息
		statusCode := client.StatusCode(err)
		switch statusCode {
		case 404:
			t.Skip("沙箱不存在，跳过测试")
		case 403:
			t.Skip("沙箱不支持恢复操作，跳过测试")
		case 500:
			// 500 错误通常是后端 Kubernetes 基础设施问题
			t.Fatalf("恢复沙箱失败（后端错误 %d）: %v", statusCode, err)
		default:
			t.Fatalf("恢复沙箱失败（状态码 %d）: %v", statusCode, err)
		}
	}

	if resumedSandbox.Status == "" {
		t.Error("沙箱状态不应为空")
	}

	t.Logf("恢复请求已提交，当前沙箱状态: %s", resumedSandbox.Status)
	t.Logf("注意：恢复操作是异步的，实际恢复可能需要一些时间完成")
}

// TestIntegrationSetTimeout 测试设置超时
func TestIntegrationSetTimeout(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "timeout-test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
		Timeout:   300,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	timeoutReq := models.SandboxTimeoutRequest{
		Timeout: 600, // 10 分钟
	}

	updatedSandbox, err := sandboxClient.SetTimeout(ctx, sandbox.SandboxID, timeoutReq)
	if err != nil {
		t.Fatalf("设置超时失败: %v", err)
	}

	if updatedSandbox.Timeout != timeoutReq.Timeout {
		t.Errorf("期望超时时间 %d 秒, 得到 %d 秒", timeoutReq.Timeout, updatedSandbox.Timeout)
	}

	t.Logf("设置超时请求成功，返回的超时时间: %d 秒", updatedSandbox.Timeout)

	// 再次查询沙箱，验证超时是否真正设置成功
	time.Sleep(1 * time.Second) // 等待一小段时间，确保后端更新完成
	verifySandbox, err := sandboxClient.Get(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("查询沙箱失败: %v", err)
	}

	if verifySandbox.Timeout != timeoutReq.Timeout {
		t.Errorf("验证失败：期望超时时间 %d 秒, 实际查询得到 %d 秒", timeoutReq.Timeout, verifySandbox.Timeout)
	} else {
		t.Logf("✓ 验证成功：超时时间已正确设置为 %d 秒", verifySandbox.Timeout)
	}
}

// TestIntegrationErrorHandling 测试错误处理
func TestIntegrationErrorHandling(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 测试获取不存在的沙箱
	_, err := sandboxClient.Get(ctx, "nonexistent-sandbox-id")
	if err == nil {
		t.Fatal("期望返回错误，但得到 nil")
	}

	if client.IsNotFound(err) {
		t.Log("正确识别 404 错误")
	} else if client.IsUnauthorized(err) {
		t.Log("正确识别 401 错误")
	} else if apiErr, ok := err.(*client.APIError); ok {
		t.Logf("API 错误 (状态码 %d): %s", apiErr.StatusCode, apiErr.Message)
	} else {
		t.Logf("其他错误: %v", err)
	}
}
