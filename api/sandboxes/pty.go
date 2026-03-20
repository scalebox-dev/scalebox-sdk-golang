package sandboxes

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/pb"
)

// PtySize represents terminal dimensions.
type PtySize struct {
	Cols uint32
	Rows uint32
}

// PtyOptions configures PTY creation.
type PtyOptions struct {
	Size    PtySize
	User    string
	Cwd     string
	Envs    map[string]string
	Timeout int // seconds
}

// PTYClient provides PTY (pseudo-terminal) operations.
type PTYClient struct {
	agent *AgentClient
}

// Create creates a new PTY session (typically a shell).
// Returns CommandHandle for sending input and receiving output.
func (pc *PTYClient) Create(ctx context.Context, opts *PtyOptions) (*CommandHandle, error) {
	if opts == nil {
		opts = &PtyOptions{}
	}
	user := opts.User
	if user == "" {
		user = "user"
	}
	cwd := opts.Cwd
	if cwd == "" {
		cwd = "/"
	}
	envs := opts.Envs
	if envs == nil {
		envs = make(map[string]string)
	}
	if _, ok := envs["TERM"]; !ok {
		envs["TERM"] = "xterm-256color"
	}
	cols := opts.Size.Cols
	if cols == 0 {
		cols = 80
	}
	rows := opts.Size.Rows
	if rows == 0 {
		rows = 24
	}

	req := &pb.StartRequest{
		Process: &pb.ProcessConfig{
			Cmd:  "/bin/bash",
			Args: []string{"-l"},
			Cwd:  &cwd,
			Envs: envs,
		},
		Pty: &pb.PTY{
			Size: &pb.PTY_Size{Cols: cols, Rows: rows},
		},
	}

	stream, err := pc.agent.processClient.Start(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("start PTY: %w", err)
	}

	// Read Start event to get PID
	if !stream.Receive() {
		if err := stream.Err(); err != nil {
			return nil, fmt.Errorf("start PTY stream: %w", err)
		}
		return nil, fmt.Errorf("start PTY: no start event")
	}
	ev := stream.Msg().GetEvent()
	if ev == nil {
		return nil, fmt.Errorf("start PTY: nil event")
	}
	start, ok := ev.Event.(*pb.ProcessEvent_Start)
	if !ok {
		return nil, fmt.Errorf("start PTY: expected start event")
	}
	pid := start.Start.Pid

	// Drain the stream in background so the agent does not block
	go func() {
		for stream.Receive() {
			_ = stream.Msg()
		}
		_ = stream.Err()
	}()

	return &CommandHandle{PID: pid, agent: pc.agent}, nil
}

// SendStdin sends input to a PTY process.
func (pc *PTYClient) SendStdin(ctx context.Context, pid uint32, data []byte) error {
	_, err := pc.agent.processClient.SendInput(ctx, connect.NewRequest(&pb.SendInputRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
		Input:   &pb.ProcessInput{Input: &pb.ProcessInput_Pty{Pty: data}},
	}))
	if err != nil {
		return fmt.Errorf("send stdin: %w", err)
	}
	return nil
}

// Resize resizes the PTY terminal.
func (pc *PTYClient) Resize(ctx context.Context, pid uint32, size PtySize) error {
	_, err := pc.agent.processClient.Update(ctx, connect.NewRequest(&pb.UpdateRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
		Pty:     &pb.PTY{Size: &pb.PTY_Size{Cols: size.Cols, Rows: size.Rows}},
	}))
	if err != nil {
		return fmt.Errorf("resize PTY: %w", err)
	}
	return nil
}

// Kill terminates a PTY process by PID.
func (pc *PTYClient) Kill(ctx context.Context, pid uint32) error {
	_, err := pc.agent.processClient.SendSignal(ctx, connect.NewRequest(&pb.SendSignalRequest{
		Process: &pb.ProcessSelector{Selector: &pb.ProcessSelector_Pid{Pid: pid}},
		Signal:  pb.Signal_SIGNAL_SIGKILL,
	}))
	if err != nil {
		return fmt.Errorf("kill PTY: %w", err)
	}
	return nil
}
