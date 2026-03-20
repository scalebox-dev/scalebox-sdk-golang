package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

// TransportConfig configures HTTP transport for connection reuse.
type TransportConfig struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	ProxyURL            *url.URL // Optional HTTP proxy for all requests
}

// DefaultTransportConfig returns recommended transport settings for API clients.
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
}

// NewClient creates a new Scalebox API client
func NewClient(baseURL, apiKey string) *Client {
	return NewClientWithConfig(baseURL, apiKey, 30*time.Second, DefaultTransportConfig())
}

// NewClientWithConfig creates a client with custom timeout and transport config.
func NewClientWithConfig(baseURL, apiKey string, timeout time.Duration, transportConfig TransportConfig) *Client {
	transport := &http.Transport{
		MaxIdleConns:        transportConfig.MaxIdleConns,
		MaxIdleConnsPerHost: transportConfig.MaxIdleConnsPerHost,
		IdleConnTimeout:     transportConfig.IdleConnTimeout,
	}
	if transportConfig.ProxyURL != nil {
		transport.Proxy = http.ProxyURL(transportConfig.ProxyURL)
	}
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
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

// NewClientWithTimeout creates a new client with the given request timeout.
// Uses DefaultTransportConfig for connection reuse.
func NewClientWithTimeout(baseURL, apiKey string, timeout time.Duration) *Client {
	return NewClientWithConfig(baseURL, apiKey, timeout, DefaultTransportConfig())
}

// NewClientWithProxy creates a new client with HTTP proxy. All API requests will go through the proxy.
func NewClientWithProxy(baseURL, apiKey string, proxyURL *url.URL) *Client {
	cfg := DefaultTransportConfig()
	cfg.ProxyURL = proxyURL
	return NewClientWithConfig(baseURL, apiKey, 30*time.Second, cfg)
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

// DoRequestRaw performs an HTTP request and returns the raw response without parsing.
// Caller is responsible for closing resp.Body. Use for streaming downloads.
func (c *Client) DoRequestRaw(ctx context.Context, method, path string, body io.Reader, headers map[string]string, queryParams map[string]string) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path

	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-KEY", c.APIKey)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// MultipartField represents a form field for multipart upload
type MultipartField struct {
	Name   string
	Reader io.Reader
	Size   int64 // -1 if unknown
}

// DoRequestMultipart performs a multipart/form-data upload.
// Fields are added in order. For file uploads, use MultipartField with Reader and optional Size.
func (c *Client) DoRequestMultipart(ctx context.Context, method, path string, fields map[string]string, fileFields []MultipartField, extraHeaders map[string]string) (*http.Response, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		for k, v := range fields {
			if err := writer.WriteField(k, v); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		for _, f := range fileFields {
			part, err := writer.CreateFormFile("file", f.Name)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if _, err := io.Copy(part, f.Reader); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	u, err := url.Parse(c.BaseURL)
	if err != nil {
		pr.Close()
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	u.Path = path

	req, err := http.NewRequestWithContext(ctx, method, u.String(), pr)
	if err != nil {
		pr.Close()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-KEY", c.APIKey)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		pr.Close()
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// CheckResponse reads the response body and returns an error if status is not 2xx.
// Caller should use this for raw responses (e.g., upload) when not using ParseResponse.
func (c *Client) CheckResponse(resp *http.Response) error {
	return c.ParseResponse(resp, nil)
}

// StandardResponse represents the backend's standard API response wrapper
type StandardResponse struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data,omitempty"`
	Message   string          `json:"message,omitempty"`
	Error     string          `json:"error,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
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
		var apiErr StandardResponse
		if err := json.Unmarshal(body, &apiErr); err == nil {
			if apiErr.Error != "" {
				return &APIError{
					StatusCode: resp.StatusCode,
					Message:    apiErr.Error,
				}
			}
			if apiErr.Message != "" {
				return &APIError{
					StatusCode: resp.StatusCode,
					Message:    apiErr.Message,
				}
			}
		}
		// Fallback: try old error format
		var oldErr Error
		if err := json.Unmarshal(body, &oldErr); err == nil && oldErr.Message != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    oldErr.Message,
			}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(body)),
		}
	}

	// Parse JSON response
	if target != nil {
		// Try to parse as wrapped response first
		var wrapped StandardResponse
		if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Success && len(wrapped.Data) > 0 {
			// Response is wrapped, extract data field
			if err := json.Unmarshal(wrapped.Data, target); err != nil {
				return fmt.Errorf("failed to parse response data: %w", err)
			}
			return nil
		}

		// Not wrapped, parse directly
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

// Health checks if the backend is healthy by calling GET /health
func (c *Client) Health(ctx context.Context) error {
	resp, err := c.DoRequest(ctx, "GET", "/health", nil, nil)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("health check failed with status %d", resp.StatusCode),
		}
	}
	return nil
}
