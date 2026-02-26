package sandboxes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

func TestCreateAndConnect(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	callCount := 0
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		agentURL := agentServer.URL

		switch {
		case r.URL.Path == "/v1/sandboxes" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(models.Sandbox{
				SandboxID:           "sbx-created",
				Name:                "Test",
				Status:              "starting",
				AllowInternetAccess: true,
				SandboxDomain:       &agentURL,
				EnvdAccessToken:     ptr("test-token"),
			})

		case r.URL.Path == "/v1/sandboxes/sbx-created/status" && r.Method == "GET":
			callCount++
			status := "starting"
			if callCount >= 2 {
				status = "running"
			}
			json.NewEncoder(w).Encode(models.SandboxStatus{
				SandboxID: "sbx-created",
				Status:    status,
			})

		case r.URL.Path == "/v1/sandboxes/sbx-created" && r.Method == "GET":
			json.NewEncoder(w).Encode(models.Sandbox{
				SandboxID:           "sbx-created",
				Status:              "running",
				AllowInternetAccess: true,
				SandboxDomain:       &agentURL,
				EnvdAccessToken:     ptr("test-token"),
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	rest := NewClient(baseClient)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := CreateAndConnect(ctx, rest, models.CreateSandboxRequest{
		Name:      "Test",
		Template:  "base",
		CPUCount:  2,
		MemoryMB:  512,
		StorageGB: 2,
	}, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("CreateAndConnect failed: %v", err)
	}

	if session.SandboxID != "sbx-created" {
		t.Errorf("expected SandboxID sbx-created, got %s", session.SandboxID)
	}
	if session.REST != rest {
		t.Error("expected REST client to match")
	}
	if session.Agent == nil {
		t.Error("expected non-nil Agent")
	}
}

func TestConnectToExisting(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()
	agentURL := agentServer.URL

	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/v1/sandboxes/sbx-existing" || r.Method != "GET" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(models.Sandbox{
			SandboxID:           "sbx-existing",
			Status:              "running",
			AllowInternetAccess: true,
			SandboxDomain:       &agentURL,
			EnvdAccessToken:     ptr("test-token"),
		})
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	rest := NewClient(baseClient)

	session, err := ConnectToExisting(context.Background(), rest, "sbx-existing")
	if err != nil {
		t.Fatalf("ConnectToExisting failed: %v", err)
	}

	if session.SandboxID != "sbx-existing" {
		t.Errorf("expected SandboxID sbx-existing, got %s", session.SandboxID)
	}
	if session.Agent == nil {
		t.Error("expected non-nil Agent")
	}
}

func TestSandboxSession_Read(t *testing.T) {
	agentServer := startMockAgentServer(t)
	defer agentServer.Close()

	content := []byte("file content")
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sandboxes/sbx-1/files/download/workspace/foo.txt" {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(content)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	rest := NewClient(baseClient)
	agent := NewAgentClient(agentServer.URL, "token")

	session := &SandboxSession{
		SandboxID: "sbx-1",
		REST:      rest,
		Agent:     agent,
	}

	data, err := session.Read(context.Background(), "/workspace/foo.txt")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("expected %q, got %q", content, data)
	}
}

func TestSandboxSession_Commands_RequiresAgent(t *testing.T) {
	restServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer restServer.Close()

	baseClient := client.NewClient(restServer.URL, "test-api-key")
	rest := NewClient(baseClient)

	session := &SandboxSession{SandboxID: "sbx-1", REST: rest, Agent: nil}

	if session.Commands() != nil {
		t.Error("Commands() should return nil when Agent is nil")
	}
}

func ptr(s string) *string {
	return &s
}
