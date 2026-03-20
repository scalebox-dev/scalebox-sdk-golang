package models

import "time"

// CreateSandboxRequest represents a request to create a sandbox
type CreateSandboxRequest struct {
	Name                string               `json:"name"`
	Description         string               `json:"description"`
	Template            string               `json:"template"`             // Template name or ID, defaults to "base"
	ProjectID           string               `json:"project_id,omitempty"` // Optional: defaults to user's default project
	CPUCount            int                  `json:"cpu_count"`
	MemoryMB            int                  `json:"memory_mb"`
	StorageGB           int                  `json:"storage_gb"`
	Metadata            map[string]string    `json:"metadata,omitempty"`
	Timeout             int                  `json:"timeout,omitempty"`    // Timeout in seconds, defaults to 300 (5 minutes)
	AutoPause           *bool                `json:"auto_pause,omitempty"` // If true, timeout causes auto-pause; if false/null, timeout causes termination
	EnvVars             map[string]string    `json:"env_vars,omitempty"`
	Secure              *bool                `json:"secure,omitempty"`                // Security setting, defaults to true if not provided
	AllowInternetAccess *bool                `json:"allow_internet_access,omitempty"` // Internet access, defaults to true if not provided
	ObjectStorage       *ObjectStorageConfig `json:"object_storage,omitempty"`
	CustomPorts         []PortConfig         `json:"custom_ports,omitempty"`
	NetProxyCountry     string               `json:"net_proxy_country,omitempty"` // Optional: Preferred proxy country (ISO code)
	Locality            *LocalityRequest     `json:"locality,omitempty"`
}

// ObjectStorageConfig represents object storage configuration
type ObjectStorageConfig struct {
	URI        string `json:"uri"`                // Format: s3://bucket/object-path (required)
	MountPoint string `json:"mount_point"`        // Absolute mount path in container (required)
	AccessKey  string `json:"access_key"`         // S3 access key (required, NOT stored in DB)
	SecretKey  string `json:"secret_key"`         // S3 secret key (required, NOT stored in DB)
	Endpoint   string `json:"endpoint,omitempty"` // Custom endpoint URL (optional)
	Region     string `json:"region,omitempty"`   // Region name (optional)
}

// LocalityRequest represents locality-based scheduling preferences
type LocalityRequest struct {
	AutoDetect bool   `json:"auto_detect"` // Whether to auto-detect region from source IP
	Region     string `json:"region"`      // Specified Sandbox Region (takes precedence over auto_detect)
	Force      bool   `json:"force"`       // Hard constraint: fail if region not available (default: false, best-effort)
}

// UpdateSandboxRequest represents a request to update a sandbox
type UpdateSandboxRequest struct {
	Timeout int `json:"timeout"` // New timeout in seconds (extends lifetime from started_at) - must be greater than current
}

// SandboxTimeoutRequest represents a request to set sandbox timeout
type SandboxTimeoutRequest struct {
	Timeout int `json:"timeout"` // New timeout in seconds - must be greater than or equal to already used lifetime
}

// ConnectSandboxRequest represents a request to connect to a sandbox
type ConnectSandboxRequest struct {
	Timeout *int `json:"timeout,omitempty"` // Optional timeout in seconds - if provided, must be valid and will update sandbox timeout
}

// PauseSandboxRequest represents a request to pause a sandbox
type PauseSandboxRequest struct {
	IsAsync *bool `json:"is_async,omitempty"` // if true, return immediately without waiting for pause completion
}

// PauseOptions holds optional parameters for Pause
type PauseOptions struct {
	IsAsync bool
}

// ResumeSandboxRequest represents a request to resume a sandbox
type ResumeSandboxRequest struct {
	IsAsync *bool `json:"is_async,omitempty"` // if true, return immediately without waiting for resume completion
}

// ResumeOptions holds optional parameters for Resume
type ResumeOptions struct {
	IsAsync bool
}

// ListSandboxesOptions represents options for listing sandboxes
type ListSandboxesOptions struct {
	ProjectID   string
	Status      string
	OwnerUserID string
	Search      string
	SortBy      string
	SortOrder   string
	Limit       int
	Offset      int
	Page        int // 1-based page (takes precedence over Offset when > 0)
}

// BatchDeleteRequest represents a request to batch delete sandboxes
type BatchDeleteRequest struct {
	SandboxIDs []string `json:"sandbox_ids"`
	Force      *bool    `json:"force,omitempty"`
}

// BatchTerminateRequest represents a request to batch terminate sandboxes
type BatchTerminateRequest struct {
	SandboxIDs []string `json:"sandbox_ids"`
	Force      *bool    `json:"force,omitempty"`
}

// BatchPauseRequest represents a request to batch pause sandboxes
type BatchPauseRequest struct {
	SandboxIDs []string `json:"sandbox_ids"`
}

// BatchResumeRequest represents a request to batch resume sandboxes
type BatchResumeRequest struct {
	SandboxIDs []string `json:"sandbox_ids"`
}

// CreateTemplateFromSandboxRequest represents a request to create a template from a sandbox
type CreateTemplateFromSandboxRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// AddPortRequest represents a request to add a port to a sandbox
type AddPortRequest struct {
	Port        int32  `json:"port"`
	ServicePort int32  `json:"service_port,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Name        string `json:"name,omitempty"`
	IsProtected bool   `json:"is_protected"`
}

// GetSandboxMetricsOptions represents options for getting sandbox metrics
type GetSandboxMetricsOptions struct {
	Start *time.Time
	End   *time.Time
	Step  *int
}
