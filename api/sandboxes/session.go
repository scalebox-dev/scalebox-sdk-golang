package sandboxes

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

// SandboxSession aggregates REST and Agent clients for a sandbox, similar to Python's Sandbox.
// Use CreateAndConnect or ConnectToExisting to obtain a session.
type SandboxSession struct {
	SandboxID string
	REST      *Client
	Agent     *AgentClient
}

// CreateAndConnect creates a sandbox, polls until it is running, then connects to the Agent.
// Returns a SandboxSession with both REST and Agent clients.
// pollInterval is the delay between status checks; use 2*time.Second for typical cases.
func CreateAndConnect(ctx context.Context, rest *Client, req models.CreateSandboxRequest, pollInterval time.Duration) (*SandboxSession, error) {
	sandbox, err := rest.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create sandbox: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		status, err := rest.GetStatus(ctx, sandbox.SandboxID)
		if err != nil {
			return nil, fmt.Errorf("get status: %w", err)
		}

		switch status.Status {
		case "running":
			agent, err := rest.ConnectToAgent(ctx, sandbox.SandboxID)
			if err != nil {
				return nil, fmt.Errorf("connect to agent: %w", err)
			}
			return &SandboxSession{
				SandboxID: sandbox.SandboxID,
				REST:      rest,
				Agent:     agent,
			}, nil
		case "failed", "terminated":
			return nil, fmt.Errorf("sandbox reached terminal state: %s", status.Status)
		}

		time.Sleep(pollInterval)
	}
}

// ConnectToExisting connects to an existing sandbox by ID. The sandbox must be running
// with internet access for Agent features.
func ConnectToExisting(ctx context.Context, rest *Client, sandboxID string) (*SandboxSession, error) {
	_, err := rest.Get(ctx, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}

	agent, err := rest.ConnectToAgent(ctx, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("connect to agent: %w", err)
	}

	return &SandboxSession{
		SandboxID: sandboxID,
		REST:      rest,
		Agent:     agent,
	}, nil
}

func (s *SandboxSession) requireAgent() (*AgentClient, error) {
	if s.Agent == nil {
		return nil, fmt.Errorf("agent not connected: use CreateAndConnect or ConnectToExisting")
	}
	return s.Agent, nil
}

// Read reads file content. Delegates to REST client.
func (s *SandboxSession) Read(ctx context.Context, path string, opts ...*models.FileOperationOptions) ([]byte, error) {
	return s.REST.Read(ctx, s.SandboxID, path, opts...)
}

// Write writes data to a file. Delegates to REST client.
func (s *SandboxSession) Write(ctx context.Context, path string, data []byte, opts ...*models.FileOperationOptions) error {
	return s.REST.Write(ctx, s.SandboxID, path, data, opts...)
}

// ListFiles lists directory contents. Delegates to REST client.
func (s *SandboxSession) ListFiles(ctx context.Context, path string, depth uint32, opts ...*models.ListFilesOptions) (*models.FileListResponse, error) {
	return s.REST.ListFiles(ctx, s.SandboxID, path, depth, opts...)
}

// Stat returns file or directory info. Delegates to REST client.
func (s *SandboxSession) Stat(ctx context.Context, path string, opts ...*models.FileOperationOptions) (*models.EntryInfo, error) {
	return s.REST.Stat(ctx, s.SandboxID, path, opts...)
}

// Exists checks if a file or directory exists. Delegates to REST client.
func (s *SandboxSession) Exists(ctx context.Context, path string, opts ...*models.FileOperationOptions) (bool, error) {
	return s.REST.Exists(ctx, s.SandboxID, path, opts...)
}

// Download returns a stream of file content. Delegates to REST client.
func (s *SandboxSession) Download(ctx context.Context, path string, opts ...*models.FileOperationOptions) (io.ReadCloser, error) {
	return s.REST.Download(ctx, s.SandboxID, path, opts...)
}

// ReadStream returns a stream of file content. Delegates to REST client.
func (s *SandboxSession) ReadStream(ctx context.Context, path string, opts ...*models.FileOperationOptions) (io.ReadCloser, error) {
	return s.REST.ReadStream(ctx, s.SandboxID, path, opts...)
}

// Commands returns the Commands client for command execution. Requires Agent.
func (s *SandboxSession) Commands() *CommandsClient {
	agent, _ := s.requireAgent()
	if agent == nil {
		return nil
	}
	return agent.Commands()
}

// PTY returns the PTY client. Requires Agent.
func (s *SandboxSession) PTY() *PTYClient {
	agent, _ := s.requireAgent()
	if agent == nil {
		return nil
	}
	return agent.PTY()
}

// CodeInterpreter returns the Code Interpreter client. Requires Agent.
func (s *SandboxSession) CodeInterpreter() *CodeInterpreterClient {
	agent, _ := s.requireAgent()
	if agent == nil {
		return nil
	}
	return agent.CodeInterpreter()
}

// MakeDir creates a directory. Requires Agent.
func (s *SandboxSession) MakeDir(ctx context.Context, path string) (*models.EntryInfo, error) {
	agent, err := s.requireAgent()
	if err != nil {
		return nil, err
	}
	return agent.MakeDir(ctx, path)
}

// Move renames or moves a file. Requires Agent.
func (s *SandboxSession) Move(ctx context.Context, source, destination string) (*models.EntryInfo, error) {
	agent, err := s.requireAgent()
	if err != nil {
		return nil, err
	}
	return agent.Move(ctx, source, destination)
}

// Remove deletes a file or directory. Requires Agent.
func (s *SandboxSession) Remove(ctx context.Context, path string) error {
	agent, err := s.requireAgent()
	if err != nil {
		return err
	}
	return agent.Remove(ctx, path)
}

// WatchDir monitors directory changes. Requires Agent.
func (s *SandboxSession) WatchDir(ctx context.Context, path string, recursive bool, onEvent func(FilesystemEvent)) (*WatchHandle, error) {
	agent, err := s.requireAgent()
	if err != nil {
		return nil, err
	}
	return agent.WatchDir(ctx, path, recursive, onEvent)
}
