package sandboxes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/pb"
)

// CommandsClient provides command execution methods.
type CommandsClient struct {
	agent *AgentClient
}

// RunOptions configures command execution.
type RunOptions struct {
	Background bool
	User       string // "user" or "root"
	Cwd        string
	Envs       map[string]string
	Timeout    int // seconds
	OnStdout   func([]byte)
	OnStderr   func([]byte)
}

// CommandResult holds the result of a foreground command.
type CommandResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

// CommandHandle represents a running or connected command for interaction.
type CommandHandle struct {
	PID    uint32
	agent  *AgentClient
	stream *connect.ServerStreamForClient[pb.ConnectResponse] // set when Connect() is used for output
}

// Run runs a command in the sandbox.
// If opts.Background is true, returns CommandHandle immediately.
// Otherwise waits for completion and returns CommandResult.
func (cc *CommandsClient) Run(ctx context.Context, cmd string, opts *RunOptions) (interface{}, error) {
	if opts == nil {
		opts = &RunOptions{}
	}
	user := opts.User
	if user == "" {
		user = "user"
	}
	envs := opts.Envs
	if envs == nil {
		envs = make(map[string]string)
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd = "/"
	}

	req := &pb.StartRequest{
		Process: &pb.ProcessConfig{
			Cmd:  "/bin/sh",
			Args: []string{"-c", cmd},
			Cwd:  &cwd,
			Envs: envs,
		},
	}
	if !opts.Background {
		req.Pty = &pb.PTY{Size: &pb.PTY_Size{Cols: 80, Rows: 24}}
	}

	stream, err := cc.agent.processClient.Start(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("start process: %w", err)
	}

	var pid uint32
	var stdout, stderr []byte
	var exitCode int32

	for stream.Receive() {
		msg := stream.Msg()
		ev := msg.GetEvent()
		if ev == nil {
			continue
		}
		switch e := ev.Event.(type) {
		case *pb.ProcessEvent_Start:
			pid = e.Start.Pid
			if opts.Background {
				return &CommandHandle{PID: pid, agent: cc.agent}, nil
			}
		case *pb.ProcessEvent_Data:
			if e.Data != nil && e.Data.Output != nil {
				switch o := e.Data.Output.(type) {
				case *pb.ProcessEvent_DataEvent_Stdout:
					stdout = append(stdout, o.Stdout...)
					if opts.OnStdout != nil {
						opts.OnStdout(o.Stdout)
					}
				case *pb.ProcessEvent_DataEvent_Stderr:
					stderr = append(stderr, o.Stderr...)
					if opts.OnStderr != nil {
						opts.OnStderr(o.Stderr)
					}
				case *pb.ProcessEvent_DataEvent_Pty:
					stdout = append(stdout, o.Pty...)
					if opts.OnStdout != nil {
						opts.OnStdout(o.Pty)
					}
				}
			}
		case *pb.ProcessEvent_End:
			exitCode = e.End.ExitCode
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}

	return &CommandResult{ExitCode: int(exitCode), Stdout: stdout, Stderr: stderr}, nil
}

// List lists running processes.
func (cc *CommandsClient) List(ctx context.Context) ([]*pb.ProcessInfo, error) {
	resp, err := cc.agent.processClient.List(ctx, connect.NewRequest(&pb.ListRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	return resp.Msg.Processes, nil
}

// Kill terminates a process by PID using SIGKILL.
func (cc *CommandsClient) Kill(ctx context.Context, pid uint32) error {
	_, err := cc.agent.processClient.SendSignal(ctx, connect.NewRequest(&pb.SendSignalRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
		Signal:  pb.Signal_SIGNAL_SIGKILL,
	}))
	if err != nil {
		return fmt.Errorf("kill: %w", err)
	}
	return nil
}

// SendStdin sends input to a running process.
func (cc *CommandsClient) SendStdin(ctx context.Context, pid uint32, data []byte) error {
	_, err := cc.agent.processClient.SendInput(ctx, connect.NewRequest(&pb.SendInputRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
		Input:   &pb.ProcessInput{Input: &pb.ProcessInput_Stdin{Stdin: data}},
	}))
	if err != nil {
		return fmt.Errorf("send stdin: %w", err)
	}
	return nil
}

// Connect connects to an existing process by PID and returns a handle for streaming output.
func (cc *CommandsClient) Connect(ctx context.Context, pid uint32) (*CommandHandle, error) {
	stream, err := cc.agent.processClient.Connect(ctx, connect.NewRequest(&pb.ConnectRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
	}))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return &CommandHandle{PID: pid, agent: cc.agent, stream: stream}, nil
}

// Wait waits for the command to finish and returns the result.
// If the handle was from Connect(), consumes the existing stream.
// If the handle was from Run(background=true), calls Connect to get the stream first.
func (h *CommandHandle) Wait(ctx context.Context) (*CommandResult, error) {
	var stream *connect.ServerStreamForClient[pb.ConnectResponse]
	if h.stream != nil {
		stream = h.stream
	} else {
		s, err := h.agent.processClient.Connect(ctx, connect.NewRequest(&pb.ConnectRequest{
			Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: h.PID}},
		}))
		if err != nil {
			return nil, fmt.Errorf("connect for wait: %w", err)
		}
		stream = s
	}

	var stdout, stderr []byte
	var exitCode int32

	for stream.Receive() {
		msg := stream.Msg()
		ev := msg.GetEvent()
		if ev == nil {
			continue
		}
		switch e := ev.Event.(type) {
		case *pb.ProcessEvent_Data:
			if e.Data != nil && e.Data.Output != nil {
				switch o := e.Data.Output.(type) {
				case *pb.ProcessEvent_DataEvent_Stdout:
					stdout = append(stdout, o.Stdout...)
				case *pb.ProcessEvent_DataEvent_Stderr:
					stderr = append(stderr, o.Stderr...)
				case *pb.ProcessEvent_DataEvent_Pty:
					stdout = append(stdout, o.Pty...)
				}
			}
		case *pb.ProcessEvent_End:
			exitCode = e.End.ExitCode
		}
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("stream: %w", err)
	}
	return &CommandResult{ExitCode: int(exitCode), Stdout: stdout, Stderr: stderr}, nil
}
