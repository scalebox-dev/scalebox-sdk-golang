package sandboxes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
	"github.com/scalebox/scalebox-sdk-golang/pb"
	"github.com/scalebox/scalebox-sdk-golang/pb/pbconnect"
)

// mockProcessHandler implements ProcessHandler for unit tests.
type mockProcessHandler struct {
	pbconnect.UnimplementedProcessHandler
}

func (m *mockProcessHandler) List(ctx context.Context, req *connect.Request[pb.ListRequest]) (*connect.Response[pb.ListResponse], error) {
	return connect.NewResponse(&pb.ListResponse{
		Processes: []*pb.ProcessInfo{
			{Pid: 42, Config: &pb.ProcessConfig{Cmd: "/bin/sh", Args: []string{"-c", "sleep 1"}}},
		},
	}), nil
}

func (m *mockProcessHandler) Start(ctx context.Context, req *connect.Request[pb.StartRequest], stream *connect.ServerStream[pb.StartResponse]) error {
	// Send Start event with PID
	if err := stream.Send(&pb.StartResponse{
		Event: &pb.ProcessEvent{Event: &pb.ProcessEvent_Start{Start: &pb.ProcessEvent_StartEvent{Pid: 123}}},
	}); err != nil {
		return err
	}
	// Send Data event (stdout)
	if err := stream.Send(&pb.StartResponse{
		Event: &pb.ProcessEvent{Event: &pb.ProcessEvent_Data{
			Data: &pb.ProcessEvent_DataEvent{Output: &pb.ProcessEvent_DataEvent_Stdout{Stdout: []byte("hello\n")}},
		}},
	}); err != nil {
		return err
	}
	// Send End event
	return stream.Send(&pb.StartResponse{
		Event: &pb.ProcessEvent{Event: &pb.ProcessEvent_End{End: &pb.ProcessEvent_EndEvent{ExitCode: 0}}},
	})
}

func (m *mockProcessHandler) SendInput(ctx context.Context, req *connect.Request[pb.SendInputRequest]) (*connect.Response[pb.SendInputResponse], error) {
	return connect.NewResponse(&pb.SendInputResponse{}), nil
}

func (m *mockProcessHandler) SendSignal(ctx context.Context, req *connect.Request[pb.SendSignalRequest]) (*connect.Response[pb.SendSignalResponse], error) {
	return connect.NewResponse(&pb.SendSignalResponse{}), nil
}

func (m *mockProcessHandler) Update(ctx context.Context, req *connect.Request[pb.UpdateRequest]) (*connect.Response[pb.UpdateResponse], error) {
	return connect.NewResponse(&pb.UpdateResponse{}), nil
}

func (m *mockProcessHandler) Connect(ctx context.Context, req *connect.Request[pb.ConnectRequest], stream *connect.ServerStream[pb.ConnectResponse]) error {
	if err := stream.Send(&pb.ConnectResponse{
		Event: &pb.ProcessEvent{Event: &pb.ProcessEvent_Data{
			Data: &pb.ProcessEvent_DataEvent{Output: &pb.ProcessEvent_DataEvent_Stdout{Stdout: []byte("connected\n")}},
		}},
	}); err != nil {
		return err
	}
	return stream.Send(&pb.ConnectResponse{
		Event: &pb.ProcessEvent{Event: &pb.ProcessEvent_End{End: &pb.ProcessEvent_EndEvent{ExitCode: 0}}},
	})
}

// mockFilesystemHandler implements FilesystemHandler for WatchDir, MakeDir, Move, Remove.
type mockFilesystemHandler struct {
	pbconnect.UnimplementedFilesystemHandler
}

func (m *mockFilesystemHandler) MakeDir(ctx context.Context, req *connect.Request[pb.MakeDirRequest]) (*connect.Response[pb.MakeDirResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		path = "/"
	}
	return connect.NewResponse(&pb.MakeDirResponse{
		Entry: &pb.EntryInfo{
			Name: "mydir",
			Type: pb.FileType_FILE_TYPE_DIRECTORY,
			Path: path,
		},
	}), nil
}

func (m *mockFilesystemHandler) Move(ctx context.Context, req *connect.Request[pb.MoveRequest]) (*connect.Response[pb.MoveResponse], error) {
	return connect.NewResponse(&pb.MoveResponse{
		Entry: &pb.EntryInfo{
			Name: "moved",
			Type: pb.FileType_FILE_TYPE_FILE,
			Path: req.Msg.GetDestination(),
		},
	}), nil
}

func (m *mockFilesystemHandler) Remove(ctx context.Context, req *connect.Request[pb.RemoveRequest]) (*connect.Response[pb.RemoveResponse], error) {
	return connect.NewResponse(&pb.RemoveResponse{}), nil
}

func (m *mockFilesystemHandler) WatchDir(ctx context.Context, req *connect.Request[pb.WatchDirRequest], stream *connect.ServerStream[pb.WatchDirResponse]) error {
	if err := stream.Send(&pb.WatchDirResponse{
		Event: &pb.WatchDirResponse_Start{Start: &pb.WatchDirResponse_StartEvent{}},
	}); err != nil {
		return err
	}
	return stream.Send(&pb.WatchDirResponse{
		Event: &pb.WatchDirResponse_Filesystem{Filesystem: &pb.FilesystemEvent{Name: "test.txt", Type: pb.EventType_EVENT_TYPE_CREATE}},
	})
}

// mockContextHandler implements ContextServiceHandler.
type mockContextHandler struct {
	pbconnect.UnimplementedContextServiceHandler
}

func (m *mockContextHandler) CreateContext(ctx context.Context, req *connect.Request[pb.CreateContextRequest]) (*connect.Response[pb.Context], error) {
	return connect.NewResponse(&pb.Context{
		Id:       "ctx-123",
		Language: "python",
		Cwd:      "/home/user",
	}), nil
}

func (m *mockContextHandler) DestroyContext(ctx context.Context, req *connect.Request[pb.DestroyContextRequest]) (*connect.Response[pb.DestroyContextResponse], error) {
	return connect.NewResponse(&pb.DestroyContextResponse{Success: true}), nil
}

// mockExecutionHandler implements ExecutionServiceHandler.
type mockExecutionHandler struct {
	pbconnect.UnimplementedExecutionServiceHandler
}

func (m *mockExecutionHandler) Execute(ctx context.Context, req *connect.Request[pb.ExecuteRequest], stream *connect.ServerStream[pb.ExecuteResponse]) error {
	if err := stream.Send(&pb.ExecuteResponse{
		Event: &pb.ExecuteResponse_Stdout{Stdout: &pb.Output{Content: "2"}},
	}); err != nil {
		return err
	}
	return stream.Send(&pb.ExecuteResponse{
		Event: &pb.ExecuteResponse_Result{Result: &pb.Result{ExitCode: 0, Text: "2"}},
	})
}

// startMockAgentServer starts an httptest server with mock Connect handlers.
func startMockAgentServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	_, processHandler := pbconnect.NewProcessHandler(&mockProcessHandler{})
	mux.Handle("/sandboxagent.Process/", processHandler)

	_, filesystemHandler := pbconnect.NewFilesystemHandler(&mockFilesystemHandler{})
	mux.Handle("/sandboxagent.Filesystem/", filesystemHandler)

	_, contextHandler := pbconnect.NewContextServiceHandler(&mockContextHandler{})
	mux.Handle("/sandboxagent.ContextService/", contextHandler)

	_, executionHandler := pbconnect.NewExecutionServiceHandler(&mockExecutionHandler{})
	mux.Handle("/sandboxagent.ExecutionService/", executionHandler)

	return httptest.NewServer(mux)
}

func TestConnectToAgent_Success(t *testing.T) {
	// REST server for Get sandbox
	domain := "test.example.com"
	token := "test-token-123"
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sandboxes/sbx-123" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		sandbox := models.Sandbox{
			SandboxID:           "sbx-123",
			Name:                "Test",
			Status:              "running",
			AllowInternetAccess: true,
			SandboxDomain:       &domain,
			EnvdAccessToken:     &token,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	agent, err := sandboxClient.ConnectToAgent(context.Background(), "sbx-123")
	if err != nil {
		t.Fatalf("ConnectToAgent failed: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil AgentClient")
	}
}

func TestConnectToAgent_SandboxNotRunning(t *testing.T) {
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sandbox := models.Sandbox{
			SandboxID: "sbx-123",
			Status:    "paused",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.ConnectToAgent(context.Background(), "sbx-123")
	if err == nil {
		t.Fatal("expected error when sandbox not running")
	}
	if !strings.Contains(err.Error(), "running") {
		t.Errorf("expected error about running status, got: %v", err)
	}
}

func TestConnectToAgent_NoDomain(t *testing.T) {
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sandbox := models.Sandbox{
			SandboxID:           "sbx-123",
			Status:              "running",
			AllowInternetAccess: true,
			SandboxDomain:       nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.ConnectToAgent(context.Background(), "sbx-123")
	if err == nil {
		t.Fatal("expected error when no domain")
	}
	if !strings.Contains(err.Error(), "domain") {
		t.Errorf("expected error about domain, got: %v", err)
	}
}

func TestConnectToAgent_NoInternetAccess(t *testing.T) {
	domain := "x.com"
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sandbox := models.Sandbox{
			SandboxID:           "sbx-123",
			Status:              "running",
			AllowInternetAccess: false,
			SandboxDomain:       &domain,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sandbox)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	sandboxClient := NewClient(baseClient)

	_, err := sandboxClient.ConnectToAgent(context.Background(), "sbx-123")
	if err == nil {
		t.Fatal("expected error when no internet access")
	}
	if !strings.Contains(err.Error(), "internet") {
		t.Errorf("expected error about internet, got: %v", err)
	}
}

func TestAgentClient_Commands_List(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	procs, err := agent.Commands().List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(procs) != 1 {
		t.Fatalf("expected 1 process, got %d", len(procs))
	}
	if procs[0].Pid != 42 {
		t.Errorf("expected PID 42, got %d", procs[0].Pid)
	}
}

func TestAgentClient_Commands_Run(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	result, err := agent.Commands().Run(context.Background(), "echo hello", nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	cr, ok := result.(*CommandResult)
	if !ok {
		t.Fatalf("expected *CommandResult, got %T", result)
	}
	if cr.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", cr.ExitCode)
	}
	if string(cr.Stdout) != "hello\n" {
		t.Errorf("expected stdout 'hello\\n', got %q", string(cr.Stdout))
	}
}

func TestAgentClient_Commands_Kill(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.Commands().Kill(context.Background(), 42)
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
}

func TestAgentClient_Commands_SendStdin(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.Commands().SendStdin(context.Background(), 42, []byte("data"))
	if err != nil {
		t.Fatalf("SendStdin failed: %v", err)
	}
}

func TestAgentClient_CodeInterpreter_CreateContext(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	ctx, err := agent.CodeInterpreter().CreateContext(context.Background(), nil)
	if err != nil {
		t.Fatalf("CreateContext failed: %v", err)
	}
	if ctx.ID != "ctx-123" {
		t.Errorf("expected ctx ID ctx-123, got %s", ctx.ID)
	}
	if ctx.Language != "python" {
		t.Errorf("expected language python, got %s", ctx.Language)
	}
}

func TestAgentClient_CodeInterpreter_DestroyContext(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.CodeInterpreter().DestroyContext(context.Background(), "ctx-123")
	if err != nil {
		t.Fatalf("DestroyContext failed: %v", err)
	}
}

func TestAgentClient_CodeInterpreter_RunCode(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	result, err := agent.CodeInterpreter().RunCode(context.Background(), "ctx-123", "1+1", nil)
	if err != nil {
		t.Fatalf("RunCode failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if string(result.Stdout) != "2" {
		t.Errorf("expected stdout '2', got %q", string(result.Stdout))
	}
	if result.Result == nil || result.Result.Text != "2" {
		t.Errorf("expected result text '2', got %v", result.Result)
	}
}

func TestAgentClient_WatchDir(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	eventCh := make(chan FilesystemEvent, 4)
	handle, err := agent.WatchDir(context.Background(), "/tmp", false, func(ev FilesystemEvent) {
		eventCh <- ev
	})
	if err != nil {
		t.Fatalf("WatchDir failed: %v", err)
	}
	// Wait for mock to send Filesystem event (or timeout)
	select {
	case ev := <-eventCh:
		if ev.Name != "test.txt" {
			t.Errorf("expected name test.txt, got %s", ev.Name)
		}
	case <-handle.Done():
		t.Fatal("stream closed before receiving event")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for event")
	}
	handle.Stop()
	<-handle.Done()
}

func TestAgentClient_PTY_Create(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	handle, err := agent.PTY().Create(context.Background(), &PtyOptions{Size: PtySize{Cols: 80, Rows: 24}})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if handle.PID != 123 {
		t.Errorf("expected PID 123, got %d", handle.PID)
	}
}

func TestAgentClient_PTY_Resize(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.PTY().Resize(context.Background(), 123, PtySize{Cols: 100, Rows: 30})
	if err != nil {
		t.Fatalf("Resize failed: %v", err)
	}
}

func TestAgentClient_PTY_Kill(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.PTY().Kill(context.Background(), 123)
	if err != nil {
		t.Fatalf("Kill failed: %v", err)
	}
}

func TestAgentClient_MakeDir(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	entry, err := agent.MakeDir(context.Background(), "/home/user/mydir")
	if err != nil {
		t.Fatalf("MakeDir failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Name != "mydir" {
		t.Errorf("expected name mydir, got %s", entry.Name)
	}
	if entry.Type != models.FileTypeDirectory {
		t.Errorf("expected type directory, got %v", entry.Type)
	}
}

func TestAgentClient_Move(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	entry, err := agent.Move(context.Background(), "/home/user/old.txt", "/home/user/new.txt")
	if err != nil {
		t.Fatalf("Move failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
	if entry.Path != "/home/user/new.txt" {
		t.Errorf("expected path /home/user/new.txt, got %s", entry.Path)
	}
}

func TestAgentClient_Remove(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.Remove(context.Background(), "/home/user/file.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestAgentClient_List(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	resp, err := agent.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(resp.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(resp.Processes))
	}
	if resp.Processes[0].Pid != 42 {
		t.Errorf("expected PID 42, got %d", resp.Processes[0].Pid)
	}
}

func TestAgentClient_Getters(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	if agent.Process() == nil {
		t.Error("Process() should not be nil")
	}
	if agent.Filesystem() == nil {
		t.Error("Filesystem() should not be nil")
	}
	if agent.Execution() == nil {
		t.Error("Execution() should not be nil")
	}
	if agent.Context() == nil {
		t.Error("Context() should not be nil")
	}
}

func TestAgentClient_PTY_SendStdin(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	err := agent.PTY().SendStdin(context.Background(), 123, []byte("ls\n"))
	if err != nil {
		t.Fatalf("SendStdin failed: %v", err)
	}
}

func TestAgentClient_Commands_Connect(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	handle, err := agent.Commands().Connect(context.Background(), 42)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if handle == nil {
		t.Fatal("expected non-nil CommandHandle")
	}
	if handle.PID != 42 {
		t.Errorf("expected PID 42, got %d", handle.PID)
	}
}

func TestAgentClient_WithHTTPClient(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	customClient := &http.Client{Timeout: 5 * time.Second}
	agent := NewAgentClient(agentServer.URL, "token", WithHTTPClient(customClient))

	if agent == nil {
		t.Fatal("expected non-nil AgentClient")
	}
	// Verify List still works with custom client
	_, err := agent.List(context.Background())
	if err != nil {
		t.Fatalf("List with custom client failed: %v", err)
	}
}

func TestCommandHandle_Wait_FromConnect(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	handle, err := agent.Commands().Connect(context.Background(), 42)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	result, err := handle.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if string(result.Stdout) != "connected\n" {
		t.Errorf("expected stdout 'connected\\n', got %q", string(result.Stdout))
	}
}

func TestCommandHandle_Wait_FromBackground(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	agent := NewAgentClient(agentServer.URL, "token")

	// Run with Background=true returns handle with no stream
	runResult, err := agent.Commands().Run(context.Background(), "sleep 1", &RunOptions{Background: true})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	handle, ok := runResult.(*CommandHandle)
	if !ok {
		t.Fatalf("expected *CommandHandle, got %T", runResult)
	}
	if handle.stream != nil {
		t.Fatal("background handle should have nil stream")
	}

	// Wait calls Connect internally to get stream
	result, err := handle.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if string(result.Stdout) != "connected\n" {
		t.Errorf("expected stdout 'connected\\n', got %q", string(result.Stdout))
	}
}

