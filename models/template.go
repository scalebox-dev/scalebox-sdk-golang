package models

import "time"

// Template represents a sandbox template
type Template struct {
	TemplateID          string      `json:"template_id"`
	Name                string      `json:"name"`
	Description         string      `json:"description,omitempty"`
	ExternalImageURL    string      `json:"external_image_url,omitempty"`
	Ports               interface{} `json:"ports,omitempty"` // API may return string or array
	CustomCommand       string      `json:"custom_command,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
	UpdatedAt           time.Time   `json:"updated_at"`
	Status              string      `json:"status,omitempty"`
	IsPublic            bool        `json:"is_public,omitempty"`
	IconURL             string      `json:"icon_url,omitempty"`
	Category            string      `json:"category,omitempty"`
	Tags                []string    `json:"tags,omitempty"`
	SupportedFamilies   []string    `json:"supported_families,omitempty"`
	Architecture        string      `json:"architecture,omitempty"`
	DiskGB              int         `json:"disk_gb,omitempty"`
	RecommendedMemoryMB int         `json:"recommended_memory_mb,omitempty"`
	RecommendedCPUCount int         `json:"recommended_cpu_count,omitempty"`
	RuntimeMinutes      int         `json:"runtime_minutes,omitempty"`
	CostPerMinute       float64     `json:"cost_per_minute,omitempty"`
	EstimatedCost       float64     `json:"estimated_cost,omitempty"`
}

// TemplateListResponse represents the response from listing templates
type TemplateListResponse struct {
	Templates  []Template `json:"templates"`
	Page       int        `json:"page,omitempty"`
	Total      int        `json:"total,omitempty"`
	TotalPages int        `json:"total_pages,omitempty"`
}

// ImportTemplateRequest represents a request to import a template from an external image
type ImportTemplateRequest struct {
	Name             string      `json:"name"`
	Description      string      `json:"description,omitempty"`
	ExternalImageURL string      `json:"external_image_url,omitempty"`
	Ports           []PortConfig `json:"ports,omitempty"`
	CustomCommand   string      `json:"custom_command,omitempty"`
}

// ImportTemplateResponse represents the response from importing a template
type ImportTemplateResponse struct {
	JobID string `json:"job_id"`
}

// GetImportJobResponse represents the response from getting an import job status
type GetImportJobResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	TemplateID string   `json:"template_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}
