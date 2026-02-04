package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client represents the Scalebox API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Scalebox API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTPClient creates a new client with a custom HTTP client
func NewClientWithHTTPClient(baseURL, apiKey string, httpClient *http.Client) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: httpClient,
	}
}

// DoRequest performs an HTTP request
func (c *Client) DoRequest(ctx context.Context, method, path string, body interface{}, queryParams map[string]string) (*http.Response, error) {
	// Build URL
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path

	// Add query parameters
	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Create request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", c.APIKey)

	// Perform request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// ParseResponse parses the HTTP response into the target struct
func (c *Client) ParseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr Error
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    apiErr.Message,
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	// Parse JSON response
	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Error represents an API error response
type Error struct {
	Message string `json:"message"`
}

// APIError represents an API error
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}
