package models

import (
	"encoding/json"
	"io"
	"strconv"
	"time"
)

// FileType represents the type of a filesystem entry
type FileType int

const (
	FileTypeUnspecified FileType = 0
	FileTypeFile        FileType = 1
	FileTypeDirectory   FileType = 2
)

func (f *FileType) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		switch x {
		case "FILE_TYPE_FILE":
			*f = FileTypeFile
		case "FILE_TYPE_DIRECTORY":
			*f = FileTypeDirectory
		default:
			*f = FileTypeUnspecified
		}
	case float64:
		*f = FileType(int(x))
	default:
		*f = FileTypeUnspecified
	}
	return nil
}

// FlexibleInt64 支持 JSON 中 size 等字段为 string 或 number
type FlexibleInt64 int64

func (f *FlexibleInt64) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch x := v.(type) {
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return err
		}
		*f = FlexibleInt64(n)
	case float64:
		*f = FlexibleInt64(x)
	case nil:
		*f = 0
	default:
		return nil
	}
	return nil
}

func (f FlexibleInt64) Int64() int64 { return int64(f) }

// EntryInfo represents a file or directory entry
type EntryInfo struct {
	Name          string        `json:"name"`
	Type          FileType      `json:"type"`
	Path          string        `json:"path"`
	Size          FlexibleInt64 `json:"size"` // 后端可能返回 string 或 number
	Mode          uint32        `json:"mode"`
	Permissions   string        `json:"permissions,omitempty"`
	Owner         string        `json:"owner,omitempty"`
	Group         string        `json:"group,omitempty"`
	ModifiedTime  *time.Time    `json:"modified_time,omitempty"`
	SymlinkTarget *string       `json:"symlink_target,omitempty"`
}

// FileListRequest is the request for listing directory contents
type FileListRequest struct {
	Path  string `json:"path"`
	Depth uint32 `json:"depth,omitempty"`
	User  string `json:"user,omitempty"` // optional: execute as user (e.g. "user", "root")
}

// FileListResponse is the response from listing directory contents
type FileListResponse struct {
	Entries []EntryInfo `json:"entries"`
}

// FileStatRequest is the request for getting file/directory info
type FileStatRequest struct {
	Path string `json:"path"`
	User string `json:"user,omitempty"` // optional: execute as user
}

// ListFilesOptions holds optional parameters for ListFiles
type ListFilesOptions struct {
	User string // optional: execute as user (e.g. "user", "root")
}

// FileOperationOptions holds optional parameters for file operations (GetInfo, Download, Write, Upload)
type FileOperationOptions struct {
	User string // optional: execute as user
}

// FileStatResponse wraps the agent Stat response (Connect RPC returns entry field)
type FileStatResponse struct {
	Entry EntryInfo `json:"entry"`
}

// WriteEntry contains path and data for file upload
type WriteEntry struct {
	Path          string
	Data          io.Reader
	ContentLength int64 // -1 if unknown
}

// WriteInfo represents the result of a file write (matches Python SDK WriteInfo)
type WriteInfo struct {
	Name string   `json:"name"`
	Type FileType `json:"type"`
	Path string   `json:"path"`
}

// UploadResponse is the response from upload API
type UploadResponse struct {
	Message   string `json:"message"`
	SessionID string `json:"sessionId"`
}

// UploadProgress represents SSE upload progress event
type UploadProgress struct {
	BytesRead  int64   `json:"bytesRead"`
	TotalBytes int64   `json:"totalBytes"`
	Progress   float64 `json:"progress"` // 0-100
	Status     string  `json:"status"`
}
