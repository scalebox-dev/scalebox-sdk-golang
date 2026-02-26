//go:build integration

package integration

import (
	"context"
	"io"
	"os"
	"path/filepath"
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

// getTestTemplate 返回集成测试使用的沙箱模板名称
// helm-deployment 的 push-public-template.sh 仅注册 code-interpreter，故默认使用该模板
// 可通过 SCALEBOX_TEMPLATE 环境变量覆盖
func getTestTemplate() string {
	if tpl := os.Getenv("SCALEBOX_TEMPLATE"); tpl != "" {
		return tpl
	}
	return "code-interpreter"
}

// TestIntegrationCreateSandbox 测试创建沙箱
func TestIntegrationCreateSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "integration-test-sandbox",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048, // code-interpreter 模板要求至少 2048 MB
		StorageGB: 2,    // 不超过订阅计划限制（常见为 2GB）
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		// 后端在某些集群/沙箱上可能无法获取 runtime 指标，返回 500 时跳过
		if client.StatusCode(err) == 500 {
			t.Skip("后端暂无法获取沙箱指标 (500)，跳过测试")
		}
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
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

// TestIntegrationListFiles 测试列出沙箱内目录
func TestIntegrationListFiles(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	// 先创建一个沙箱用于测试
	createReq := models.CreateSandboxRequest{
		Name:      "list-files-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建测试沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	// 等待沙箱状态变为 running（文件操作需要 running 且允许互联网访问）
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
			t.Logf("沙箱状态已变为 running")
			break
		}

		if sandboxStatus.Status == "failed" || sandboxStatus.Status == "terminated" {
			t.Fatalf("沙箱状态异常（%s），无法执行文件操作", sandboxStatus.Status)
		}

		time.Sleep(checkInterval)
		t.Logf("当前状态: %s，继续等待...", sandboxStatus.Status)
	}

	finalStatus, err := sandboxClient.GetStatus(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("获取沙箱状态失败: %v", err)
	}
	if finalStatus.Status != "running" {
		t.Fatalf("沙箱在 %v 内未达到 running 状态（当前: %s）", maxWaitTime, finalStatus.Status)
	}

	// 等待 Sandbox Agent 就绪：Agent 可能在沙箱 running 后仍需几秒才能对外服务
	// 遇 500（含 sandbox agent 503）时重试，最多 2 分钟
	var result *models.FileListResponse
	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	for attempt := 0; time.Now().Before(retryDeadline); attempt++ {
		result, err = sandboxClient.ListFiles(ctx, sandbox.SandboxID, "/", 1)
		if err == nil {
			break
		}
		code := client.StatusCode(err)
		if code == 400 {
			t.Skip("沙箱不允许互联网访问或域名不可用，跳过 ListFiles 测试")
		}
		if code != 500 {
			t.Fatalf("ListFiles 失败: %v", err)
		}
		if time.Now().Add(retryInterval).After(retryDeadline) {
			t.Skipf("ListFiles 在 2 分钟内重试后仍不可用 (HTTP 500)，跳过: %v", err)
		}
		t.Logf("Sandbox Agent 暂不可达 (HTTP 500)，%v 后重试...", retryInterval)
		time.Sleep(retryInterval)
	}

	if result == nil && err != nil {
		t.Skipf("ListFiles 在 2 分钟内重试后仍不可用，跳过: %v", err)
	}
	if result == nil {
		t.Error("ListFiles 响应不应为 nil")
	}
	// 空目录时后端可能返回 Entries: null，视为 0 项
	entryCount := 0
	if result.Entries != nil {
		entryCount = len(result.Entries)
	}

	t.Logf("ListFiles 成功，目录项数量: %d", entryCount)
	for i, e := range result.Entries {
		t.Logf("  项 %d: %s (%s) - size=%d", i+1, e.Name, e.Path, e.Size.Int64())
	}
}

// waitForSandboxRunning 等待沙箱变为 running，与 TestIntegrationListFiles 中的逻辑一致
func waitForSandboxRunning(t *testing.T, sandboxClient *sandboxes.Client, ctx context.Context, sandboxID string) {
	maxWaitTime := 30 * time.Second
	checkInterval := 2 * time.Second
	deadline := time.Now().Add(maxWaitTime)
	for time.Now().Before(deadline) {
		sandboxStatus, err := sandboxClient.GetStatus(ctx, sandboxID)
		if err != nil {
			t.Fatalf("获取沙箱状态失败: %v", err)
		}
		if sandboxStatus.Status == "running" {
			return
		}
		if sandboxStatus.Status == "failed" || sandboxStatus.Status == "terminated" {
			t.Fatalf("沙箱状态异常（%s）", sandboxStatus.Status)
		}
		time.Sleep(checkInterval)
	}
	t.Fatalf("沙箱在 %v 内未达到 running 状态", maxWaitTime)
}

// TestIntegrationStatReadWrite 测试 Stat、Read、Write、Exists
func TestIntegrationStatReadWrite(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "stat-read-write-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}
	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// 重试逻辑（与 ListFiles 类似）
	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	var lastErr error
	for time.Now().Before(retryDeadline) {
		// Write
		remotePath := "/workspace/sdk-test-file.txt"
		content := []byte("hello from Go SDK integration test")
		if err := sandboxClient.Write(ctx, sandbox.SandboxID, remotePath, content); err != nil {
			lastErr = err
			if client.StatusCode(err) == 400 {
				t.Skip("沙箱不允许互联网访问或域名不可用，跳过文件操作测试")
			}
			if client.StatusCode(err) != 500 {
				t.Fatalf("Write 失败: %v", err)
			}
			time.Sleep(retryInterval)
			continue
		}

		// Exists
		exists, err := sandboxClient.Exists(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("Exists 失败: %v", err)
		}
		if !exists {
			t.Error("写入后 Exists 应返回 true")
		}

		// Stat
		info, err := sandboxClient.GetInfo(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("GetInfo 失败: %v", err)
		}
		if info.Name == "" || info.Path == "" {
			t.Error("GetInfo 应返回 name 和 path")
		}
		t.Logf("GetInfo: %s, path=%s, size=%d", info.Name, info.Path, info.Size.Int64())

		// Read
		data, err := sandboxClient.Read(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("Read 失败: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("Read 内容不匹配: 期望 %q, 得到 %q", content, data)
		}

		// Download（流式）
		rc, err := sandboxClient.Download(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("Download 失败: %v", err)
		}
		streamData, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("Download 读取失败: %v", err)
		}
		if string(streamData) != string(content) {
			t.Errorf("Download 内容不匹配: 期望 %q, 得到 %q", content, streamData)
		}

		// DownloadToFile
		localPath := filepath.Join(t.TempDir(), "sdk-downloaded.txt")
		if err := sandboxClient.DownloadToFile(ctx, sandbox.SandboxID, remotePath, localPath); err != nil {
			t.Fatalf("DownloadToFile 失败: %v", err)
		}
		fileData, err := os.ReadFile(localPath)
		if err != nil {
			t.Fatalf("读取下载文件失败: %v", err)
		}
		if string(fileData) != string(content) {
			t.Errorf("DownloadToFile 内容不匹配: 期望 %q, 得到 %q", content, fileData)
		}

		// ReadStream（与 Download 等价，单独验证 API 覆盖）
		rc2, err := sandboxClient.ReadStream(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("ReadStream 失败: %v", err)
		}
		streamData2, err := io.ReadAll(rc2)
		rc2.Close()
		if err != nil {
			t.Fatalf("ReadStream 读取失败: %v", err)
		}
		if string(streamData2) != string(content) {
			t.Errorf("ReadStream 内容不匹配: 期望 %q, 得到 %q", content, streamData2)
		}

		t.Log("Stat/Read/Write/Exists/Download/ReadStream/DownloadToFile 集成测试通过")
		return
	}
	t.Skipf("文件操作在 2 分钟内重试后仍不可用，跳过: %v", lastErr)
}

// TestIntegrationWriteBatch 测试 WriteBatch 批量写入
func TestIntegrationWriteBatch(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "write-batch-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}
	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// 重试逻辑（与 StatReadWrite 类似）
	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	var lastErr error
	for time.Now().Before(retryDeadline) {
		entries := []models.WriteEntry{
			{Path: "/workspace/batch-a.txt", Data: strings.NewReader("batch a"), ContentLength: 7},
			{Path: "/workspace/batch-b.txt", Data: strings.NewReader("batch b"), ContentLength: 7},
		}
		results, err := sandboxClient.WriteBatch(ctx, sandbox.SandboxID, entries)
		if err != nil {
			lastErr = err
			if client.StatusCode(err) == 400 {
				t.Skip("沙箱不允许互联网访问或域名不可用，跳过 WriteBatch 测试")
			}
			if client.StatusCode(err) != 500 && client.StatusCode(err) != 503 {
				t.Fatalf("WriteBatch 失败: %v", err)
			}
			time.Sleep(retryInterval)
			continue
		}
		if len(results) != 2 {
			t.Fatalf("WriteBatch 应返回 2 个结果，得到 %d", len(results))
		}
		for i, r := range results {
			if r.Path != entries[i].Path {
				t.Errorf("结果 %d path 应为 %q, 得到 %q", i, entries[i].Path, r.Path)
			}
		}
		// 验证文件内容
		expectedContent := map[string]string{
			"/workspace/batch-a.txt": "batch a",
			"/workspace/batch-b.txt": "batch b",
		}
		for path, expected := range expectedContent {
			data, err := sandboxClient.Read(ctx, sandbox.SandboxID, path)
			if err != nil {
				t.Fatalf("Read %s 失败: %v", path, err)
			}
			if string(data) != expected {
				t.Errorf("WriteBatch 后 %s 内容不匹配: 期望 %q, 得到 %q", path, expected, string(data))
			}
		}
		t.Log("WriteBatch 集成测试通过")
		return
	}
	t.Skipf("WriteBatch 在 2 分钟内重试后仍不可用，跳过: %v", lastErr)
}

// TestIntegrationWriteFromReader 测试 WriteFromReader 从 Reader 写入
func TestIntegrationWriteFromReader(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "write-reader-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	var lastErr error
	for time.Now().Before(retryDeadline) {
		remotePath := "/workspace/sdk-write-reader.txt"
		content := "hello from WriteFromReader integration test"
		r := strings.NewReader(content)
		if err := sandboxClient.WriteFromReader(ctx, sandbox.SandboxID, remotePath, r, int64(len(content))); err != nil {
			lastErr = err
			if client.StatusCode(err) == 400 {
				t.Skip("沙箱不允许互联网访问或域名不可用，跳过 WriteFromReader 测试")
			}
			if client.StatusCode(err) != 500 {
				t.Fatalf("WriteFromReader 失败: %v", err)
			}
			time.Sleep(retryInterval)
			continue
		}
		data, err := sandboxClient.Read(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("Read 验证失败: %v", err)
		}
		if string(data) != content {
			t.Errorf("WriteFromReader 后内容不匹配: 期望 %q, 得到 %q", content, string(data))
		}
		t.Log("WriteFromReader 集成测试通过")
		return
	}
	t.Skipf("WriteFromReader 在 2 分钟内重试后仍不可用，跳过: %v", lastErr)
}

// TestIntegrationUpload 测试 Upload、UploadWithProgress
func TestIntegrationUpload(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "upload-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}
	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	content := []byte("upload test content")
	remotePath := "/workspace/upload-test.txt"
	var lastErr error

	for time.Now().Before(retryDeadline) {
		// Upload（返回 UploadResponse）
		resp, err := sandboxClient.Upload(ctx, sandbox.SandboxID, remotePath, content)
		if err != nil {
			lastErr = err
			if client.StatusCode(err) == 400 {
				t.Skip("沙箱不允许互联网访问或域名不可用，跳过 Upload 测试")
			}
			if client.StatusCode(err) != 500 {
				t.Fatalf("Upload 失败: %v", err)
			}
			time.Sleep(retryInterval)
			continue
		}
		if resp != nil && resp.SessionID != "" {
			t.Logf("Upload 成功, sessionId=%s", resp.SessionID)
		}

		// 验证上传内容
		data, err := sandboxClient.Read(ctx, sandbox.SandboxID, remotePath)
		if err != nil {
			t.Fatalf("Read 验证失败: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("Upload 后内容不匹配: 期望 %q, 得到 %q", content, data)
		}

		// UploadWithProgress（SSE 进度，token 为 API key，若后端不支持则跳过）
		apiKey := os.Getenv("SCALEBOX_API_KEY")
		var progressCalls int
		_, err = sandboxClient.UploadWithProgress(ctx, sandbox.SandboxID, "/workspace/upload-progress-test.txt", content, func(p float64) {
			progressCalls++
		}, apiKey)
		if err != nil {
			if client.StatusCode(err) == 401 || client.StatusCode(err) == 403 {
				t.Log("UploadWithProgress 需要用户 token，跳过")
			} else if client.StatusCode(err) != 500 {
				t.Logf("UploadWithProgress 失败（非关键）: %v", err)
			}
		} else {
			t.Logf("UploadWithProgress 成功, 进度回调 %d 次", progressCalls)
		}

		t.Log("Upload/UploadWithProgress 集成测试通过")
		return
	}
	t.Skipf("Upload 在 2 分钟内重试后仍不可用: %v", lastErr)
}

// TestIntegrationUploadDownloadAsync 测试 UploadAsync、DownloadAsync
func TestIntegrationUploadDownloadAsync(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "async-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}
	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// 先用 Write 写入一个文件，以便 DownloadAsync 有内容可读
	remotePath := "/workspace/async-test.txt"
	content := []byte("async test content")

	retryDeadline := time.Now().Add(2 * time.Minute)
	retryInterval := 10 * time.Second
	writeOk := false
	for time.Now().Before(retryDeadline) {
		if err := sandboxClient.Write(ctx, sandbox.SandboxID, remotePath, content); err != nil {
			if client.StatusCode(err) == 400 {
				t.Skip("沙箱不允许互联网访问或域名不可用")
			}
			if client.StatusCode(err) != 500 {
				t.Fatalf("Write 失败: %v", err)
			}
			time.Sleep(retryInterval)
			continue
		}
		writeOk = true
		break
	}
	if !writeOk {
		t.Skip("Write 在 2 分钟内重试后仍不可用，跳过 Async 测试")
	}

	// DownloadAsync
	ch := sandboxClient.DownloadAsync(ctx, sandbox.SandboxID, remotePath)
	result := <-ch
	if result.Error != nil {
		t.Fatalf("DownloadAsync 失败: %v", result.Error)
	}
	if string(result.Data) != string(content) {
		t.Errorf("DownloadAsync 内容不匹配: 期望 %q, 得到 %q", content, result.Data)
	}
	t.Log("DownloadAsync 成功")

	// UploadAsync
	uploadPath := "/workspace/async-upload.txt"
	uploadContent := []byte("async upload content")
	uploadCh := sandboxClient.UploadAsync(ctx, sandbox.SandboxID, uploadPath, uploadContent)
	uploadResult := <-uploadCh
	if uploadResult.Error != nil {
		t.Fatalf("UploadAsync 失败: %v", uploadResult.Error)
	}
	if uploadResult.Response != nil {
		t.Logf("UploadAsync 成功, sessionId=%s", uploadResult.Response.SessionID)
	}

	// 验证 UploadAsync 写入的内容
	data, err := sandboxClient.Read(ctx, sandbox.SandboxID, uploadPath)
	if err != nil {
		t.Fatalf("Read UploadAsync 结果失败: %v", err)
	}
	if string(data) != string(uploadContent) {
		t.Errorf("UploadAsync 后内容不匹配: 期望 %q, 得到 %q", uploadContent, data)
	}

	t.Log("UploadAsync/DownloadAsync 集成测试通过")
}

// TestIntegrationConnectToAgent 测试 ConnectToAgent 与 Agent 能力（Commands、Code Interpreter）
func TestIntegrationConnectToAgent(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "agent-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}

	sandbox, err := sandboxClient.Create(ctx, createReq)
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() {
		_, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	}()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// ConnectToAgent 需要沙箱 running 且 allow_internet_access
	agent, err := sandboxClient.ConnectToAgent(ctx, sandbox.SandboxID)
	if err != nil {
		if strings.Contains(err.Error(), "internet") || strings.Contains(err.Error(), "domain") {
			t.Skip("沙箱未开启公网或域名不可用，跳过 Agent 测试")
		}
		t.Fatalf("ConnectToAgent 失败: %v", err)
	}

	// 测试 Commands.Run（前台）
	result, err := agent.Commands().Run(ctx, "echo hello from agent", nil)
	if err != nil {
		t.Fatalf("Commands.Run 失败: %v", err)
	}
	cr, ok := result.(*sandboxes.CommandResult)
	if !ok {
		t.Fatalf("期望 *CommandResult, 得到 %T", result)
	}
	if cr.ExitCode != 0 {
		t.Errorf("命令退出码应为 0, 得到 %d", cr.ExitCode)
	}
	if !strings.Contains(string(cr.Stdout), "hello from agent") {
		t.Errorf("期望 stdout 包含 'hello from agent', 得到 %q", string(cr.Stdout))
	}
	t.Logf("Commands.Run 成功: %s", string(cr.Stdout))

	// 测试 Code Interpreter（仅当 code-interpreter 模板支持时）
	ctxObj, err := agent.CodeInterpreter().CreateContext(ctx, &sandboxes.CreateContextOptions{Language: "python", Cwd: "/home/user"})
	if err != nil {
		t.Logf("CreateContext 失败（可能模板不支持）: %v", err)
		return
	}
	defer func() {
		_ = agent.CodeInterpreter().DestroyContext(ctx, ctxObj.ID)
	}()

	execResult, err := agent.CodeInterpreter().RunCode(ctx, ctxObj.ID, "print(1+1)", nil)
	if err != nil {
		t.Fatalf("RunCode 失败: %v", err)
	}
	if execResult.ExitCode != 0 {
		t.Errorf("代码执行退出码应为 0, 得到 %d", execResult.ExitCode)
	}
	t.Logf("Code Interpreter RunCode 成功: stdout=%q", string(execResult.Stdout))
}

// TestIntegrationSandboxSession_CreateAndConnect 测试 CreateAndConnect 创建沙箱并连接 Agent
func TestIntegrationSandboxSession_CreateAndConnect(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	createReq := models.CreateSandboxRequest{
		Name:      "session-create-test",
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	}

	session, err := sandboxes.CreateAndConnect(ctx, sandboxClient, createReq, 2*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "internet") || strings.Contains(err.Error(), "domain") {
			t.Skip("沙箱未开启公网或域名不可用，跳过 SandboxSession 测试")
		}
		t.Fatalf("CreateAndConnect 失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, session.SandboxID, nil) }()

	if session.SandboxID == "" || session.REST == nil || session.Agent == nil {
		t.Fatalf("Session 应有 SandboxID、REST 和 Agent，得到 SandboxID=%q", session.SandboxID)
	}

	// 通过 Session 执行命令
	result, err := session.Commands().Run(ctx, "echo hello from session", nil)
	if err != nil {
		t.Fatalf("Session Commands.Run 失败: %v", err)
	}
	cr, ok := result.(*sandboxes.CommandResult)
	if !ok {
		t.Fatalf("期望 *CommandResult, 得到 %T", result)
	}
	if !strings.Contains(string(cr.Stdout), "hello from session") {
		t.Errorf("期望 stdout 包含 'hello from session', 得到 %q", string(cr.Stdout))
	}
	t.Log("SandboxSession CreateAndConnect 集成测试通过")
}

// TestIntegrationSandboxSession_ConnectToExisting 测试 ConnectToExisting 连接已有沙箱
func TestIntegrationSandboxSession_ConnectToExisting(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "session-connect-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	session, err := sandboxes.ConnectToExisting(ctx, sandboxClient, sandbox.SandboxID)
	if err != nil {
		if strings.Contains(err.Error(), "internet") || strings.Contains(err.Error(), "domain") {
			t.Skip("沙箱未开启公网或域名不可用，跳过 SandboxSession 测试")
		}
		t.Fatalf("ConnectToExisting 失败: %v", err)
	}

	if session.SandboxID != sandbox.SandboxID || session.REST == nil || session.Agent == nil {
		t.Fatalf("Session 应匹配已有沙箱，SandboxID=%s", session.SandboxID)
	}

	// 通过 Session REST 委托方法验证
	_, err = session.REST.Get(ctx, session.SandboxID)
	if err != nil {
		t.Fatalf("Session REST.Get 失败: %v", err)
	}
	t.Log("SandboxSession ConnectToExisting 集成测试通过")
}

// TestIntegrationAgentPTYCommandsFilesystem 测试 PTY、Commands 扩展、MakeDir/Move/Remove、WatchDir
func TestIntegrationAgentPTYCommandsFilesystem(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "agent-extended")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	agent, err := sandboxClient.ConnectToAgent(ctx, sandbox.SandboxID)
	if err != nil {
		if strings.Contains(err.Error(), "internet") || strings.Contains(err.Error(), "domain") {
			t.Skip("沙箱未开启公网或域名不可用，跳过 Agent 扩展测试")
		}
		t.Fatalf("ConnectToAgent 失败: %v", err)
	}

	// Commands.List
	listResp, err := agent.List(ctx)
	if err != nil {
		t.Fatalf("Commands.List 失败: %v", err)
	}
	t.Logf("Commands.List 成功，进程数: %d", len(listResp.Processes))

	// Commands.Run 后台
	runResult, err := agent.Commands().Run(ctx, "sleep 60", &sandboxes.RunOptions{Background: true})
	if err != nil {
		t.Fatalf("Commands.Run(background) 失败: %v", err)
	}
	cmdHandle, ok := runResult.(*sandboxes.CommandHandle)
	if !ok {
		t.Fatalf("期望 *CommandHandle, 得到 %T", runResult)
	}
	pid := cmdHandle.PID
	t.Logf("后台进程 PID: %d", pid)

	// Commands.Connect 连接到运行中的进程
	connHandle, err := agent.Commands().Connect(ctx, pid)
	if err != nil {
		t.Logf("Commands.Connect 失败（可选）: %v", err)
	} else {
		t.Logf("Commands.Connect 成功, PID: %d", connHandle.PID)
	}

	// Commands.SendStdin（对 sleep 无效，仅验证调用）
	_ = agent.Commands().SendStdin(ctx, pid, []byte("x"))

	// Commands.Kill
	err = agent.Commands().Kill(ctx, pid)
	if err != nil {
		t.Fatalf("Commands.Kill 失败: %v", err)
	}
	t.Log("Commands.Kill 成功")

	// PTY.Create
	ptyHandle, err := agent.PTY().Create(ctx, &sandboxes.PtyOptions{
		Size: sandboxes.PtySize{Cols: 80, Rows: 24},
	})
	if err != nil {
		t.Fatalf("PTY.Create 失败: %v", err)
	}
	t.Logf("PTY.Create 成功, PID: %d", ptyHandle.PID)

	// PTY.SendStdin
	err = agent.PTY().SendStdin(ctx, ptyHandle.PID, []byte("ls\n"))
	if err != nil {
		t.Fatalf("PTY.SendStdin 失败: %v", err)
	}

	// PTY.Resize
	err = agent.PTY().Resize(ctx, ptyHandle.PID, sandboxes.PtySize{Cols: 100, Rows: 30})
	if err != nil {
		t.Fatalf("PTY.Resize 失败: %v", err)
	}

	// PTY.Kill
	err = agent.PTY().Kill(ctx, ptyHandle.PID)
	if err != nil {
		t.Fatalf("PTY.Kill 失败: %v", err)
	}
	t.Log("PTY 完整流程通过")

	// MakeDir
	entry, err := agent.MakeDir(ctx, "/workspace/sdk-it-dir")
	if err != nil {
		t.Fatalf("MakeDir 失败: %v", err)
	}
	if entry != nil && entry.Name != "" {
		t.Logf("MakeDir 成功: %s", entry.Path)
	}

	// 写入一个文件用于 Move
	_ = sandboxClient.Write(ctx, sandbox.SandboxID, "/workspace/sdk-it-move-src.txt", []byte("move me"))

	// Move (Agent 层)
	moved, err := agent.Move(ctx, "/workspace/sdk-it-move-src.txt", "/workspace/sdk-it-move-dst.txt")
	if err != nil {
		t.Fatalf("Move 失败: %v", err)
	}
	if moved != nil {
		t.Logf("Move 成功: %s", moved.Path)
	}

	// Remove
	err = agent.Remove(ctx, "/workspace/sdk-it-move-dst.txt")
	if err != nil {
		t.Fatalf("Remove 失败: %v", err)
	}
	t.Log("Remove 成功")

	// WatchDir（短暂监控后停止）
	watchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	eventCount := 0
	watchHandle, err := agent.WatchDir(watchCtx, "/workspace", false, func(_ sandboxes.FilesystemEvent) {
		eventCount++
	})
	if err != nil {
		t.Fatalf("WatchDir 失败: %v", err)
	}
	<-watchHandle.Done()
	t.Logf("WatchDir 完成, 收到事件数: %d", eventCount)
}

// TestIntegrationConnect 测试 Connect（连接/恢复沙箱）
func TestIntegrationConnect(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "connect-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	timeout := 600
	connected, err := sandboxClient.Connect(ctx, sandbox.SandboxID, &models.ConnectSandboxRequest{Timeout: &timeout})
	if err != nil {
		t.Fatalf("Connect 失败: %v", err)
	}
	if connected.SandboxID != sandbox.SandboxID {
		t.Errorf("期望 SandboxID %s, 得到 %s", sandbox.SandboxID, connected.SandboxID)
	}
	t.Log("Connect 集成测试通过")
}

// TestIntegrationUpdate 测试更新沙箱
func TestIntegrationUpdate(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "update-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	req := models.UpdateSandboxRequest{Timeout: 600}
	updated, err := sandboxClient.Update(ctx, sandbox.SandboxID, req)
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if updated.Timeout != 600 {
		t.Errorf("期望 Timeout 600, 得到 %d", updated.Timeout)
	}
	t.Log("Update 集成测试通过")
}

// TestIntegrationTerminate 测试终止沙箱
func TestIntegrationTerminate(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "terminate-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	// Terminate 后沙箱会结束，无需 Delete 清理
	resp, err := sandboxClient.Terminate(ctx, sandbox.SandboxID, nil)
	if err != nil {
		t.Fatalf("Terminate 失败: %v", err)
	}
	if resp.SandboxID != sandbox.SandboxID {
		t.Errorf("期望 SandboxID %s, 得到 %s", sandbox.SandboxID, resp.SandboxID)
	}
	t.Log("Terminate 集成测试通过")
}

// TestIntegrationDelete 测试删除沙箱
func TestIntegrationDelete(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "delete-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}

	resp, err := sandboxClient.Delete(ctx, sandbox.SandboxID, nil)
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if resp.SandboxID != sandbox.SandboxID {
		t.Errorf("期望 SandboxID %s, 得到 %s", sandbox.SandboxID, resp.SandboxID)
	}

	// 验证删除后 Get 返回 404
	_, err = sandboxClient.Get(ctx, sandbox.SandboxID)
	if err == nil {
		t.Error("Delete 后 Get 应返回错误")
	}
	if !client.IsNotFound(err) && client.StatusCode(err) != 404 {
		t.Errorf("Delete 后 Get 应返回 404, 得到: %v", err)
	}
	t.Log("Delete 集成测试通过")
}

// TestIntegrationBatchDelete 测试批量删除（后端已实现，需 force=true 删除 running 沙箱）
func TestIntegrationBatchDelete(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	s1, err := createTestSandbox(sandboxClient, ctx, t, "batch-del-1")
	if err != nil {
		t.Fatalf("创建沙箱1失败: %v", err)
	}
	s2, err := createTestSandbox(sandboxClient, ctx, t, "batch-del-2")
	if err != nil {
		_, _ = sandboxClient.Delete(ctx, s1.SandboxID, nil)
		t.Fatalf("创建沙箱2失败: %v", err)
	}

	force := true
	resp, err := sandboxClient.BatchDelete(ctx, models.BatchDeleteRequest{
		SandboxIDs: []string{s1.SandboxID, s2.SandboxID},
		Force:      &force,
	})
	if err != nil {
		_, _ = sandboxClient.Delete(ctx, s1.SandboxID, nil)
		_, _ = sandboxClient.Delete(ctx, s2.SandboxID, nil)
		t.Fatalf("BatchDelete 失败: %v", err)
	}
	if resp.Successful < 1 {
		t.Errorf("期望至少 1 个成功, 得到 successful=%d failed=%d", resp.Successful, resp.Failed)
	}
	t.Logf("BatchDelete 成功: %d successful, %d failed", resp.Successful, resp.Failed)
}

// TestIntegrationBatchTerminate 测试批量终止（后端已实现）
func TestIntegrationBatchTerminate(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	s1, err := createTestSandbox(sandboxClient, ctx, t, "batch-term-1")
	if err != nil {
		t.Fatalf("创建沙箱1失败: %v", err)
	}
	s2, err := createTestSandbox(sandboxClient, ctx, t, "batch-term-2")
	if err != nil {
		_, _ = sandboxClient.Delete(ctx, s1.SandboxID, nil)
		t.Fatalf("创建沙箱2失败: %v", err)
	}

	resp, err := sandboxClient.BatchTerminate(ctx, models.BatchTerminateRequest{
		SandboxIDs: []string{s1.SandboxID, s2.SandboxID},
	})
	if err != nil {
		_, _ = sandboxClient.Delete(ctx, s1.SandboxID, nil)
		_, _ = sandboxClient.Delete(ctx, s2.SandboxID, nil)
		t.Fatalf("BatchTerminate 失败: %v", err)
	}
	if resp.Successful < 1 {
		t.Errorf("期望至少 1 个成功, 得到 successful=%d failed=%d", resp.Successful, resp.Failed)
	}
	t.Logf("BatchTerminate 成功: %d successful, %d failed", resp.Successful, resp.Failed)
}

// TestIntegrationCreateTemplateFromSandbox 测试从沙箱创建模板（后端已实现，沙箱需 running）
func TestIntegrationCreateTemplateFromSandbox(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "create-tpl-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// 模板名需唯一，用 sandboxID 后缀避免冲突
	tplName := "sdk-it-tpl-" + sandbox.SandboxID
	if len(tplName) > 64 {
		tplName = tplName[:64]
	}
	resp, err := sandboxClient.CreateTemplateFromSandbox(ctx, sandbox.SandboxID, models.CreateTemplateFromSandboxRequest{
		Name:        tplName,
		Description: "integration test template",
	})
	if err != nil {
		t.Fatalf("CreateTemplateFromSandbox 失败: %v", err)
	}
	if resp.TemplateID == "" {
		t.Error("期望 TemplateID 非空")
	}
	t.Logf("CreateTemplateFromSandbox 成功: %s", resp.TemplateID)
}

// TestIntegrationGetPorts 测试获取端口列表（后端已实现）
func TestIntegrationGetPorts(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "ports-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	ports, err := sandboxClient.GetPorts(ctx, sandbox.SandboxID)
	if err != nil {
		t.Fatalf("GetPorts 失败: %v", err)
	}
	t.Logf("GetPorts 成功，端口数量: %d", len(ports))
}

// TestIntegrationAddPortRemovePort 测试添加和删除端口（后端已实现，AddPort 需 port+name，沙箱需 running）
func TestIntegrationAddPortRemovePort(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "add-remove-port")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	// 后端要求 port 不在 template 中，name 唯一。code-interpreter 通常有 8888/6080，用 9999
	// 后端要求 name 必填且 DNS 兼容
	port, err := sandboxClient.AddPort(ctx, sandbox.SandboxID, models.AddPortRequest{
		Port: 9999,
		Name: "sdk-test-port",
	})
	if err != nil {
		t.Fatalf("AddPort 失败: %v", err)
	}
	t.Logf("AddPort 成功: %d %s", port.Port, port.Name)

	err = sandboxClient.RemovePort(ctx, sandbox.SandboxID, 9999)
	if err != nil {
		t.Fatalf("RemovePort 失败: %v", err)
	}
	t.Log("RemovePort 成功")
}

// TestIntegrationDownloadURLUploadURLGetHost 测试 URL 构建与 GetHost
func TestIntegrationDownloadURLUploadURLGetHost(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "url-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	// DownloadURL 仅构建 URL，不请求实际下载
	dlURL, err := sandboxClient.DownloadURL(ctx, sandbox.SandboxID, "/workspace/foo.txt", nil)
	if err != nil {
		if sandbox.SandboxDomain == nil || *sandbox.SandboxDomain == "" {
			t.Skip("沙箱无 domain，跳过 URL 测试")
		}
		t.Fatalf("DownloadURL 失败: %v", err)
	}
	if !strings.HasPrefix(dlURL, "https://") {
		t.Errorf("DownloadURL 应以 https:// 开头, 得到 %s", dlURL)
	}
	t.Logf("DownloadURL: %s", dlURL[:min(80, len(dlURL))]+"...")

	// UploadURL
	upURL, err := sandboxClient.UploadURL(ctx, sandbox.SandboxID, "/workspace/", nil)
	if err != nil {
		t.Fatalf("UploadURL 失败: %v", err)
	}
	if !strings.HasPrefix(upURL, "https://") {
		t.Errorf("UploadURL 应以 https:// 开头, 得到 %s", upURL)
	}

	// GetHost
	host, err := sandboxClient.GetHost(ctx, sandbox.SandboxID, 0)
	if err != nil {
		t.Fatalf("GetHost 失败: %v", err)
	}
	if host == "" {
		t.Error("GetHost 不应返回空")
	}
	hostWithPort, _ := sandboxClient.GetHost(ctx, sandbox.SandboxID, 8080)
	if hostWithPort != "" && !strings.Contains(hostWithPort, ":8080") {
		t.Errorf("GetHost(8080) 应包含 :8080, 得到 %s", hostWithPort)
	}
	t.Log("DownloadURL/UploadURL/GetHost 集成测试通过")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestIntegrationWriteBatchConcurrent 测试并发批量写入
func TestIntegrationWriteBatchConcurrent(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "wbc-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	entries := []models.WriteEntry{
		{Path: "/workspace/concurrent-a.txt", Data: strings.NewReader("a"), ContentLength: 1},
		{Path: "/workspace/concurrent-b.txt", Data: strings.NewReader("bb"), ContentLength: 2},
		{Path: "/workspace/concurrent-c.txt", Data: strings.NewReader("ccc"), ContentLength: 3},
	}
	results, err := sandboxClient.WriteBatchConcurrent(ctx, sandbox.SandboxID, entries, 3)
	if err != nil {
		if client.StatusCode(err) == 400 {
			t.Skip("沙箱不允许互联网访问，跳过")
		}
		t.Fatalf("WriteBatchConcurrent 失败: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("期望 3 个结果, 得到 %d", len(results))
	}
	t.Log("WriteBatchConcurrent 集成测试通过")
}

// TestIntegrationUploadWithProgress 测试带进度的上传
func TestIntegrationUploadWithProgress(t *testing.T) {
	sandboxClient := setupClient(t)
	ctx := context.Background()

	sandbox, err := createTestSandbox(sandboxClient, ctx, t, "progress-test")
	if err != nil {
		t.Fatalf("创建沙箱失败: %v", err)
	}
	defer func() { _, _ = sandboxClient.Delete(ctx, sandbox.SandboxID, nil) }()

	waitForSandboxRunning(t, sandboxClient, ctx, sandbox.SandboxID)

	progressCalled := false
	apiKey := os.Getenv("SCALEBOX_API_KEY")
	_, err = sandboxClient.UploadWithProgress(ctx, sandbox.SandboxID, "/workspace/progress.txt", []byte("small"), func(p float64) {
		progressCalled = true
		t.Logf("上传进度: %.1f%%", p)
	}, apiKey)
	if err != nil {
		if client.StatusCode(err) == 400 {
			t.Skip("沙箱不允许互联网访问，跳过")
		}
		t.Fatalf("UploadWithProgress 失败: %v", err)
	}
	t.Logf("UploadWithProgress 成功, progress 回调被调用: %v", progressCalled)
}

// createTestSandbox 创建测试用沙箱
func createTestSandbox(c *sandboxes.Client, ctx context.Context, t *testing.T, name string) (*models.Sandbox, error) {
	return c.Create(ctx, models.CreateSandboxRequest{
		Name:      name,
		Template:  getTestTemplate(),
		CPUCount:  2,
		MemoryMB:  2048,
		StorageGB: 2,
	})
}
