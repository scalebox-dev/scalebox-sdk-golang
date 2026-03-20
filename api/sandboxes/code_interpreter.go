package sandboxes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/pb"
)

// CodeInterpreterClient provides code execution (run_code, create/destroy context).
type CodeInterpreterClient struct {
	agent *AgentClient
}

// CodeContext represents an execution context for code runs.
type CodeContext struct {
	ID       string
	Language string
	Cwd      string
}

// ExecutionResult holds the result of run_code.
type ExecutionResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Result   *pb.Result
	Error    *pb.Error
}

// CreateContext creates a new code execution context.
func (cc *CodeInterpreterClient) CreateContext(ctx context.Context, opts *CreateContextOptions) (*CodeContext, error) {
	if opts == nil {
		opts = &CreateContextOptions{}
	}
	language := opts.Language
	if language == "" {
		language = "python"
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd = "/home/user"
	}

	resp, err := cc.agent.contextClient.CreateContext(ctx, connect.NewRequest(&pb.CreateContextRequest{
		Language: language,
		Cwd:      cwd,
	}))
	if err != nil {
		return nil, fmt.Errorf("create context: %w", err)
	}
	m := resp.Msg
	return &CodeContext{
		ID:       m.Id,
		Language: m.Language,
		Cwd:      m.Cwd,
	}, nil
}

// CreateContextOptions configures context creation.
type CreateContextOptions struct {
	Language string
	Cwd      string
}

// DestroyContext destroys an execution context.
func (cc *CodeInterpreterClient) DestroyContext(ctx context.Context, contextID string) error {
	_, err := cc.agent.contextClient.DestroyContext(ctx, connect.NewRequest(&pb.DestroyContextRequest{
		ContextId: contextID,
	}))
	if err != nil {
		return fmt.Errorf("destroy context: %w", err)
	}
	return nil
}

// RunCodeOptions configures code execution.
type RunCodeOptions struct {
	Language string
	Envs     map[string]string
	Timeout  int // seconds
	OnStdout func([]byte)
	OnStderr func([]byte)
	OnResult func(*pb.Result)
	OnError  func(*pb.Error)
}

// RunCode executes code in the given context.
func (cc *CodeInterpreterClient) RunCode(ctx context.Context, contextID string, code string, opts *RunCodeOptions) (*ExecutionResult, error) {
	if opts == nil {
		opts = &RunCodeOptions{}
	}
	language := opts.Language
	if language == "" {
		language = "python"
	}
	envs := opts.Envs
	if envs == nil {
		envs = make(map[string]string)
	}

	stream, err := cc.agent.executionClient.Execute(ctx, connect.NewRequest(&pb.ExecuteRequest{
		ContextId: contextID,
		Code:      code,
		Language:  language,
		EnvVars:   envs,
	}))
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	var stdout, stderr []byte
	var result *pb.Result
	var execErr *pb.Error
	exitCode := 0

	for stream.Receive() {
		msg := stream.Msg()
		if msg == nil {
			continue
		}
		switch e := msg.Event.(type) {
		case *pb.ExecuteResponse_Stdout:
			if e.Stdout != nil && e.Stdout.Content != "" {
				b := []byte(e.Stdout.Content)
				stdout = append(stdout, b...)
				if opts.OnStdout != nil {
					opts.OnStdout(b)
				}
			}
		case *pb.ExecuteResponse_Stderr:
			if e.Stderr != nil && e.Stderr.Content != "" {
				b := []byte(e.Stderr.Content)
				stderr = append(stderr, b...)
				if opts.OnStderr != nil {
					opts.OnStderr(b)
				}
			}
		case *pb.ExecuteResponse_Result:
			if e.Result != nil {
				result = e.Result
				exitCode = int(e.Result.ExitCode)
				if opts.OnResult != nil {
					opts.OnResult(e.Result)
				}
			}
		case *pb.ExecuteResponse_Error:
			if e.Error != nil {
				execErr = e.Error
				if opts.OnError != nil {
					opts.OnError(e.Error)
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("execute stream: %w", err)
	}

	return &ExecutionResult{
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
		Result:   result,
		Error:    execErr,
	}, nil
}
