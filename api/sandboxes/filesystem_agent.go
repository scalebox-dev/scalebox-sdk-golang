package sandboxes

import (
	"context"
	"fmt"
	"strings"

	"github.com/scalebox/scalebox-sdk-golang/models"
	"github.com/scalebox/scalebox-sdk-golang/pb"
	"connectrpc.com/connect"
)

// MakeDir creates a directory in the sandbox via Agent gRPC.
// Path should be absolute (e.g. /home/user/mydir).
func (c *AgentClient) MakeDir(ctx context.Context, path string) (*models.EntryInfo, error) {
	path = normalizePath(path)
	if path == "" {
		return nil, fmt.Errorf("path cannot be empty")
	}
	resp, err := c.filesystemClient.MakeDir(ctx, connect.NewRequest(&pb.MakeDirRequest{
		Path: path,
	}))
	if err != nil {
		return nil, fmt.Errorf("make dir: %w", err)
	}
	if resp.Msg == nil || resp.Msg.Entry == nil {
		return nil, nil
	}
	return pbEntryToModels(resp.Msg.Entry), nil
}

// Move renames or moves a file/directory (Agent gRPC).
func (c *AgentClient) Move(ctx context.Context, source, destination string) (*models.EntryInfo, error) {
	src := normalizePath(source)
	dst := normalizePath(destination)
	if src == "" || dst == "" {
		return nil, fmt.Errorf("source and destination cannot be empty")
	}
	resp, err := c.filesystemClient.Move(ctx, connect.NewRequest(&pb.MoveRequest{
		Source:      src,
		Destination: dst,
	}))
	if err != nil {
		return nil, fmt.Errorf("move: %w", err)
	}
	if resp.Msg == nil || resp.Msg.Entry == nil {
		return nil, nil
	}
	return pbEntryToModels(resp.Msg.Entry), nil
}

// Remove deletes a file or directory recursively (Agent gRPC).
func (c *AgentClient) Remove(ctx context.Context, path string) error {
	path = normalizePath(path)
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	_, err := c.filesystemClient.Remove(ctx, connect.NewRequest(&pb.RemoveRequest{
		Path: path,
	}))
	if err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return nil
}

func normalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "/"
	}
	return "/" + p
}

func pbEntryToModels(e *pb.EntryInfo) *models.EntryInfo {
	if e == nil {
		return nil
	}
	out := &models.EntryInfo{
		Name:        e.GetName(),
		Path:        e.GetPath(),
		Size:        models.FlexibleInt64(e.GetSize()),
		Mode:        e.GetMode(),
		Permissions: e.GetPermissions(),
		Owner:       e.GetOwner(),
		Group:       e.GetGroup(),
	}
	switch e.GetType() {
	case pb.FileType_FILE_TYPE_FILE:
		out.Type = models.FileTypeFile
	case pb.FileType_FILE_TYPE_DIRECTORY:
		out.Type = models.FileTypeDirectory
	default:
		out.Type = models.FileTypeUnspecified
	}
	if e.ModifiedTime != nil {
		t := e.ModifiedTime.AsTime()
		out.ModifiedTime = &t
	}
	if e.SymlinkTarget != nil && *e.SymlinkTarget != "" {
		t := *e.SymlinkTarget
		out.SymlinkTarget = &t
	}
	return out
}
