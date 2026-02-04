package models

import "time"

// Sandbox represents a sandbox instance
type Sandbox struct {
	SandboxID                 string                 `json:"sandbox_id"`
	Name                      string                 `json:"name"`
	Description               *string                `json:"description,omitempty"`
	TemplateID                string                 `json:"template_id"`
	TemplateName              *string                `json:"template_name,omitempty"`
	TemplateExists            *bool                  `json:"template_exists,omitempty"`
	OwnerUserID               string                 `json:"owner_user_id"`
	ProjectID                 string                 `json:"project_id"`
	ProjectName               *string                `json:"project_name,omitempty"`
	CPUCount                  int                    `json:"cpu_count"`
	MemoryMB                  int                    `json:"memory_mb"`
	StorageGB                 int                    `json:"storage_gb"`
	Timeout                   int                    `json:"timeout"`
	AutoPause                 bool                   `json:"auto_pause"`
	Secure                    bool                   `json:"secure"`
	AllowInternetAccess       bool                   `json:"allow_internet_access"`
	Metadata                  map[string]string      `json:"metadata,omitempty"`
	EnvVars                   map[string]string      `json:"env_vars,omitempty"`
	ObjectStorage             map[string]string      `json:"object_storage,omitempty"`
	Ports                     []PortConfig           `json:"ports,omitempty"`
	TemplatePorts             []PortConfig           `json:"template_ports,omitempty"`
	CustomPorts               []PortConfig           `json:"custom_ports,omitempty"`
	Status                    string                 `json:"status"`
	Substatus                 *string                `json:"substatus,omitempty"`
	Reason                    *string                `json:"reason,omitempty"`
	SandboxDomain             *string                `json:"sandbox_domain,omitempty"`
	SandboxDomainInternal     *string                `json:"sandbox_domain_internal,omitempty"`
	WebTerminalAvailable      bool                   `json:"web_terminal_available"`
	WebFilesAvailable         bool                   `json:"web_files_available"`
	EnvdAccessToken           *string                `json:"envd_access_token,omitempty"`
	NetworkProxy              map[string]interface{} `json:"network_proxy,omitempty"`
	CreatedAt                 time.Time              `json:"created_at"`
	UpdatedAt                 time.Time              `json:"updated_at"`
	StartedAt                 *time.Time             `json:"started_at,omitempty"`
	StoppedAt                 *time.Time             `json:"stopped_at,omitempty"`
	EndedAt                   *time.Time             `json:"ended_at,omitempty"`
	TimeoutAt                 *time.Time             `json:"timeout_at,omitempty"`
	PausedAt                  *time.Time             `json:"paused_at,omitempty"`
	PausingAt                 *time.Time             `json:"pausing_at,omitempty"`
	ResumedAt                 *time.Time             `json:"resumed_at,omitempty"`
	TotalPausedSeconds        int                    `json:"total_paused_seconds"`
	TotalRunningSeconds       int                    `json:"total_running_seconds"`
	ActualTotalPausedSeconds  *int                   `json:"actual_total_paused_seconds,omitempty"`
	ActualTotalRunningSeconds *int64                 `json:"actual_total_running_seconds,omitempty"`
	Uptime                    int64                  `json:"uptime,omitempty"`
	PersistenceDays           *int                   `json:"persistence_days,omitempty"`
	PersistenceExpiresAt      *time.Time             `json:"persistence_expires_at,omitempty"`
	PersistenceDaysRemaining  *int                   `json:"persistence_days_remaining,omitempty"`
	Owner                     *Owner                 `json:"owner,omitempty"`
	AccountOwner              *AccountOwner          `json:"account_owner,omitempty"`
	Resources                 *Resources             `json:"resources,omitempty"`
}

// Owner represents the sandbox owner
type Owner struct {
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
	Email       string  `json:"email"`
}

// AccountOwner represents the account owner
type AccountOwner struct {
	AccountID          string  `json:"account_id"`
	AccountDisplayName *string `json:"account_display_name,omitempty"`
	AccountEmail       *string `json:"account_email,omitempty"`
}

// Resources represents sandbox resources
type Resources struct {
	CPU       int `json:"cpu"`
	Memory    int `json:"memory"`
	Storage   int `json:"storage"`
	Bandwidth int `json:"bandwidth"`
}

// PortConfig represents a port configuration
type PortConfig struct {
	Port        int32  `json:"port"`
	ServicePort int32  `json:"service_port,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Name        string `json:"name,omitempty"`
	IsProtected bool   `json:"is_protected"`
}

// SandboxStatus represents lightweight sandbox status
type SandboxStatus struct {
	SandboxID string    `json:"sandbox_id"`
	Status    string    `json:"status"`
	Substatus *string   `json:"substatus,omitempty"`
	Reason    *string   `json:"reason,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SandboxListResponse represents the response from listing sandboxes
type SandboxListResponse struct {
	Sandboxes []Sandbox `json:"sandboxes"`
}

// DeletionResponse represents the response from deleting a sandbox
type DeletionResponse struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
	Note      string `json:"note"`
}

// TerminationResponse represents the response from terminating a sandbox
type TerminationResponse struct {
	SandboxID string `json:"sandbox_id"`
	Status    string `json:"status"`
}
