package models

import "time"

// SandboxMetricsResponse represents the response from getting sandbox metrics
type SandboxMetricsResponse struct {
	SandboxID     string             `json:"sandbox_id"`
	Timestamp     time.Time          `json:"timestamp"`
	Status        string             `json:"status"`
	UptimeSeconds int64              `json:"uptime_seconds"`
	Metrics       []MetricsDataPoint `json:"metrics"`
}

// MetricsDataPoint represents a single timestamped metrics group
type MetricsDataPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	CPUCount   int       `json:"cpu_count"`    // requested CPU count (cores as int)
	CPUUsedPct float64   `json:"cpu_used_pct"` // CPU usage percentage (0-100)
	DiskTotal  int64     `json:"disk_total"`   // requested storage (bytes)
	DiskUsed   int64     `json:"disk_used"`    // used storage (bytes)
	MemTotal   int64     `json:"mem_total"`    // requested memory (bytes)
	MemUsed    int64     `json:"mem_used"`     // used memory (bytes)
}
