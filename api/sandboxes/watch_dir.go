package sandboxes

import (
	"context"
	"fmt"
	"sync"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/pb"
)

// FilesystemEvent represents a file system event from watch_dir.
type FilesystemEvent struct {
	Name string
	Type pb.EventType
}

// WatchHandle represents an active directory watch.
type WatchHandle struct {
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
	mu     sync.Mutex
}

// Stop stops the directory watch.
func (h *WatchHandle) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
}

// Done returns a channel that is closed when the watch stops.
func (h *WatchHandle) Done() <-chan struct{} {
	return h.done
}

// WatchDir monitors directory changes and invokes onEvent for each event.
// The watch runs until ctx is cancelled or Stop() is called.
// Returns a WatchHandle to stop the watch.
func (c *AgentClient) WatchDir(ctx context.Context, path string, recursive bool, onEvent func(FilesystemEvent)) (*WatchHandle, error) {
	stream, err := c.filesystemClient.WatchDir(ctx, connect.NewRequest(&pb.WatchDirRequest{
		Path:      path,
		Recursive: recursive,
	}))
	if err != nil {
		return nil, fmt.Errorf("watch dir: %w", err)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	h := &WatchHandle{
		ctx:    watchCtx,
		cancel: cancel,
		done:   done,
	}

	go func() {
		defer close(done)
		for stream.Receive() {
			select {
			case <-watchCtx.Done():
				return
			default:
			}
			msg := stream.Msg()
			if msg == nil {
				continue
			}
			switch e := msg.Event.(type) {
			case *pb.WatchDirResponse_Filesystem:
				if e.Filesystem != nil && onEvent != nil {
					onEvent(FilesystemEvent{
						Name: e.Filesystem.Name,
						Type: e.Filesystem.Type,
					})
				}
			}
		}
	}()

	return h, nil
}
