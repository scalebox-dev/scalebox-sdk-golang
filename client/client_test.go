package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "key")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.BaseURL != "https://api.example.com" || c.APIKey != "key" {
		t.Errorf("BaseURL or APIKey incorrect: %s %s", c.BaseURL, c.APIKey)
	}
	if c.HTTPClient.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", c.HTTPClient.Timeout)
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	c := NewClientWithHTTPClient("https://api.example.com", "key", custom)
	if c == nil {
		t.Fatal("NewClientWithHTTPClient returned nil")
	}
	if c.HTTPClient != custom {
		t.Error("expected custom HTTP client to be used")
	}
}

func TestNewClientWithTimeout(t *testing.T) {
	c := NewClientWithTimeout("https://api.example.com", "key", 60*time.Second)
	if c == nil {
		t.Fatal("NewClientWithTimeout returned nil")
	}
	if c.HTTPClient.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", c.HTTPClient.Timeout)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	cfg := DefaultTransportConfig()
	c := NewClientWithConfig("https://api.example.com", "key", 10*time.Second, cfg)
	if c == nil {
		t.Fatal("NewClientWithConfig returned nil")
	}
	if c.HTTPClient.Timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", c.HTTPClient.Timeout)
	}
	if transport, ok := c.HTTPClient.Transport.(*http.Transport); ok {
		if transport.MaxIdleConns != cfg.MaxIdleConns {
			t.Errorf("expected MaxIdleConns %d, got %d", cfg.MaxIdleConns, transport.MaxIdleConns)
		}
		if transport.MaxIdleConnsPerHost != cfg.MaxIdleConnsPerHost {
			t.Errorf("expected MaxIdleConnsPerHost %d, got %d", cfg.MaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)
		}
	} else {
		t.Error("expected *http.Transport")
	}
}

func TestNewClientWithProxy(t *testing.T) {
	proxyURL, err := url.Parse("http://proxy.example.com:8080")
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	c := NewClientWithProxy("https://api.example.com", "key", proxyURL)
	if c == nil {
		t.Fatal("NewClientWithProxy returned nil")
	}
	transport, ok := c.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if transport.Proxy == nil {
		t.Error("expected Proxy to be set")
	}
}

func TestIsRateLimited(t *testing.T) {
	if !IsRateLimited(&APIError{StatusCode: 429, Message: "rate limited"}) {
		t.Error("IsRateLimited should return true for 429")
	}
	if IsRateLimited(&APIError{StatusCode: 404, Message: "not found"}) {
		t.Error("IsRateLimited should return false for 404")
	}
	if IsRateLimited(nil) {
		t.Error("IsRateLimited should return false for nil")
	}
}

func TestIsTimeout(t *testing.T) {
	if !IsTimeout(&APIError{StatusCode: 408, Message: "timeout"}) {
		t.Error("IsTimeout should return true for 408")
	}
	if IsTimeout(&APIError{StatusCode: 500, Message: "error"}) {
		t.Error("IsTimeout should return false for 500")
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(&APIError{StatusCode: 404, Message: "not found"}) {
		t.Error("IsNotFound should return true for 404")
	}
	if IsNotFound(&APIError{StatusCode: 500, Message: "error"}) {
		t.Error("IsNotFound should return false for 500")
	}
	if IsNotFound(nil) {
		t.Error("IsNotFound should return false for nil")
	}
}

func TestIsUnauthorized(t *testing.T) {
	if !IsUnauthorized(&APIError{StatusCode: 401, Message: "unauthorized"}) {
		t.Error("IsUnauthorized should return true for 401")
	}
	if IsUnauthorized(&APIError{StatusCode: 404, Message: "not found"}) {
		t.Error("IsUnauthorized should return false for 404")
	}
}

func TestIsForbidden(t *testing.T) {
	if !IsForbidden(&APIError{StatusCode: 403, Message: "forbidden"}) {
		t.Error("IsForbidden should return true for 403")
	}
	if IsForbidden(&APIError{StatusCode: 404, Message: "not found"}) {
		t.Error("IsForbidden should return false for 404")
	}
}

func TestStatusCode(t *testing.T) {
	if StatusCode(&APIError{StatusCode: 404, Message: "not found"}) != 404 {
		t.Error("StatusCode should return 404")
	}
	if StatusCode(nil) != 0 {
		t.Error("StatusCode should return 0 for nil")
	}
}

func TestDoRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/test" {
			t.Errorf("expected /v1/test, got %s", r.URL.Path)
		}
		if r.Header.Get("X-API-KEY") != "test-key" {
			t.Error("expected X-API-KEY header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type application/json")
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["foo"] != "bar" {
			t.Errorf("expected foo=bar, got %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer server.Close()

	c := NewClientWithHTTPClient(server.URL, "test-key", server.Client())
	resp, err := c.DoRequest(context.Background(), "POST", "/v1/test", map[string]string{"foo": "bar"}, nil)
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDoRequestWithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("status") != "running" {
			t.Errorf("expected query params, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewClientWithHTTPClient(server.URL, "key", server.Client())
	resp, err := c.DoRequest(context.Background(), "GET", "/v1/test", nil, map[string]string{"limit": "10", "status": "running"})
	if err != nil {
		t.Fatalf("DoRequest: %v", err)
	}
	defer resp.Body.Close()
}

func TestDoRequestInvalidBaseURL(t *testing.T) {
	c := NewClient("://invalid", "key")
	_, err := c.DoRequest(context.Background(), "GET", "/path", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid base URL")
	}
}

func TestDoRequestRaw(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("raw body"))
	}))
	defer server.Close()

	c := NewClientWithHTTPClient(server.URL, "key", server.Client())
	resp, err := c.DoRequestRaw(context.Background(), "GET", "/v1/raw", strings.NewReader("body"), nil, nil)
	if err != nil {
		t.Fatalf("DoRequestRaw: %v", err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if string(data) != "raw body" {
		t.Errorf("expected raw body, got %s", string(data))
	}
}

func TestDoRequestMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("expected multipart Content-Type, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := NewClientWithHTTPClient(server.URL, "key", server.Client())
	fields := map[string]string{"path": "/tmp/file"}
	fileFields := []MultipartField{{Name: "test.txt", Reader: strings.NewReader("content"), Size: 7}}
	resp, err := c.DoRequestMultipart(context.Background(), "POST", "/upload", fields, fileFields, nil)
	if err != nil {
		t.Fatalf("DoRequestMultipart: %v", err)
	}
	defer resp.Body.Close()
}

func TestParseResponseSuccess(t *testing.T) {
	body := `{"success":true,"data":{"id":"123"}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	c := NewClient("https://api.test", "key")
	var target struct {
		ID string `json:"id"`
	}
	if err := c.ParseResponse(resp, &target); err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if target.ID != "123" {
		t.Errorf("expected id=123, got %s", target.ID)
	}
}

func TestParseResponseDirectJSON(t *testing.T) {
	body := `{"id":"456"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	c := NewClient("https://api.test", "key")
	var target struct {
		ID string `json:"id"`
	}
	if err := c.ParseResponse(resp, &target); err != nil {
		t.Fatalf("ParseResponse: %v", err)
	}
	if target.ID != "456" {
		t.Errorf("expected id=456, got %s", target.ID)
	}
}

func TestParseResponseError(t *testing.T) {
	body := `{"error":"not found"}`
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	c := NewClient("https://api.test", "key")
	err := c.ParseResponse(resp, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if apiErr, ok := err.(*APIError); !ok || apiErr.StatusCode != 404 {
		t.Errorf("expected APIError 404, got %v", err)
	}
}

func TestParseResponseErrorWithMessage(t *testing.T) {
	body := `{"message":"Resource not found"}`
	resp := &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	c := NewClient("https://api.test", "key")
	err := c.ParseResponse(resp, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if apiErr, ok := err.(*APIError); !ok || apiErr.Message != "Resource not found" {
		t.Errorf("expected APIError with message, got %v", err)
	}
}

func TestCheckResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
	}
	c := NewClient("https://api.test", "key")
	err := c.CheckResponse(resp)
	if err == nil {
		t.Fatal("CheckResponse should return error for 400")
	}
}

func TestAPIError(t *testing.T) {
	e := &APIError{StatusCode: 404, Message: "not found"}
	s := e.Error()
	if !strings.Contains(s, "404") || !strings.Contains(s, "not found") {
		t.Errorf("APIError.Error() = %s", s)
	}
}
