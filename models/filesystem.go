package models

import "time"

// FileType represents the type of a filesystem entry
type FileType int

const (
	FileTypeUnspecified FileType = 0
	FileTypeFile        FileType = 1
	FileTypeDirectory   FileType = 2
)

// EntryInfo represents a file or directory entry
type EntryInfo struct {
	Name          string     `json:"name"`
	Type          FileType   `json:"type"`
	Path          string     `json:"path"`
	Size          int64      `json:"size"`
	Mode          uint32     `json:"mode"`
	Permissions   string     `json:"permissions,omitempty"`
	Owner         string     `json:"owner,omitempty"`
	Group         string     `json:"group,omitempty"`
	ModifiedTime  *time.Time `json:"modified_time,omitempty"`
	SymlinkTarget *string    `json:"symlink_target,omitempty"`
}

// FileListRequest is the request for listing directory contents
type FileListRequest struct {
	Path  string `json:"path"`
	Depth uint32 `json:"depth,omitempty"`
}

// FileListResponse is the response from listing directory contents
type FileListResponse struct {
	Entries []EntryInfo `json:"entries"`
}
