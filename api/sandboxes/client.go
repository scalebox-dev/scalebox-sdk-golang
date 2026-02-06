package sandboxes

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/client"
	"github.com/scalebox/scalebox-sdk-golang/models"
)

// Client provides methods for interacting with the Sandboxes API
type Client struct {
	baseClient *client.Client
}

// NewClient creates a new Sandboxes API client
func NewClient(baseClient *client.Client) *Client {
	return &Client{
		baseClient: baseClient,
	}
}

// Create creates a new sandbox
func (c *Client) Create(ctx context.Context, req models.CreateSandboxRequest) (*models.Sandbox, error) {
	resp, err := c.baseClient.DoRequest(ctx, "POST", "/v1/sandboxes", req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// List lists sandboxes with optional filters
func (c *Client) List(ctx context.Context, opts *models.ListSandboxesOptions) (*models.SandboxListResponse, error) {
	queryParams := make(map[string]string)
	if opts != nil {
		if opts.ProjectID != "" {
			queryParams["project_id"] = opts.ProjectID
		}
		if opts.Status != "" {
			queryParams["status"] = opts.Status
		}
		if opts.OwnerUserID != "" {
			queryParams["owner_user_id"] = opts.OwnerUserID
		}
		if opts.Search != "" {
			queryParams["search"] = opts.Search
		}
		if opts.SortBy != "" {
			queryParams["sort_by"] = opts.SortBy
		}
		if opts.SortOrder != "" {
			queryParams["sort_order"] = opts.SortOrder
		}
		if opts.Limit > 0 {
			queryParams["limit"] = strconv.Itoa(opts.Limit)
		}
		if opts.Offset > 0 {
			queryParams["offset"] = strconv.Itoa(opts.Offset)
		}
	}

	resp, err := c.baseClient.DoRequest(ctx, "GET", "/v1/sandboxes", nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.SandboxListResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Get retrieves a sandbox by ID
func (c *Client) Get(ctx context.Context, sandboxID string) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// GetStatus retrieves lightweight sandbox status
func (c *Client) GetStatus(ctx context.Context, sandboxID string) (*models.SandboxStatus, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/status", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var status models.SandboxStatus
	if err := c.baseClient.ParseResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Update updates a sandbox
func (c *Client) Update(ctx context.Context, sandboxID string, req models.UpdateSandboxRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "PUT", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Delete deletes a sandbox
func (c *Client) Delete(ctx context.Context, sandboxID string, force *bool) (*models.DeletionResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s", sandboxID)
	queryParams := make(map[string]string)
	if force != nil && !*force {
		queryParams["force"] = "false"
	}

	resp, err := c.baseClient.DoRequest(ctx, "DELETE", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.DeletionResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Terminate terminates a sandbox
func (c *Client) Terminate(ctx context.Context, sandboxID string, force *bool) (*models.TerminationResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/terminate", sandboxID)
	queryParams := make(map[string]string)
	if force != nil && *force {
		queryParams["force"] = "true"
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.TerminationResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Pause pauses a sandbox
func (c *Client) Pause(ctx context.Context, sandboxID string) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/pause", sandboxID)
	req := models.PauseSandboxRequest{}
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Resume resumes a sandbox
func (c *Client) Resume(ctx context.Context, sandboxID string) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/resume", sandboxID)
	req := models.ResumeSandboxRequest{}
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// Connect connects to a sandbox (resumes if paused)
func (c *Client) Connect(ctx context.Context, sandboxID string, req *models.ConnectSandboxRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/connect", sandboxID)
	if req == nil {
		req = &models.ConnectSandboxRequest{}
	}

	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// SetTimeout sets the timeout for a sandbox
func (c *Client) SetTimeout(ctx context.Context, sandboxID string, req models.SandboxTimeoutRequest) (*models.Sandbox, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/timeout", sandboxID)
	resp, err := c.baseClient.DoRequest(ctx, "POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var sandbox models.Sandbox
	if err := c.baseClient.ParseResponse(resp, &sandbox); err != nil {
		return nil, err
	}

	return &sandbox, nil
}

// GetMetrics retrieves metrics for a sandbox
func (c *Client) GetMetrics(ctx context.Context, sandboxID string, opts *models.GetSandboxMetricsOptions) (*models.SandboxMetricsResponse, error) {
	path := fmt.Sprintf("/v1/sandboxes/%s/metrics", sandboxID)
	queryParams := make(map[string]string)

	if opts != nil {
		if opts.Start != nil {
			queryParams["start"] = formatTime(*opts.Start)
		}
		if opts.End != nil {
			queryParams["end"] = formatTime(*opts.End)
		}
		if opts.Step != nil {
			queryParams["step"] = strconv.Itoa(*opts.Step)
		}
	}

	resp, err := c.baseClient.DoRequest(ctx, "GET", path, nil, queryParams)
	if err != nil {
		return nil, err
	}

	var result models.SandboxMetricsResponse
	if err := c.baseClient.ParseResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// formatTime formats time for API query parameters
// Supports Unix timestamp, RFC3339, and simple datetime formats
func formatTime(t time.Time) string {
	// Try RFC3339 first (most common)
	return t.Format(time.RFC3339)
}
