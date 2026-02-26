package sandboxes

import (
	"strings"
	"testing"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

func TestBuildDownloadURL_NoSignature(t *testing.T) {
	domain := "sbx-abc.example.com"
	sandbox := &models.Sandbox{
		SandboxDomain: &domain,
	}
	url, err := BuildDownloadURL(sandbox, "/home/user/file.txt", nil)
	if err != nil {
		t.Fatalf("BuildDownloadURL: %v", err)
	}
	if !strings.HasPrefix(url, "https://sbx-abc.example.com/download/") {
		t.Errorf("expected https://sbx-abc.example.com/download/ prefix, got %s", url)
	}
	if !strings.Contains(url, "username=root") {
		t.Errorf("expected username=root in query, got %s", url)
	}
}

func TestBuildDownloadURL_WithSignature(t *testing.T) {
	domain := "sbx-abc.example.com"
	token := "my-token"
	sandbox := &models.Sandbox{
		SandboxDomain:   &domain,
		EnvdAccessToken: &token,
	}
	exp := 360
	opts := &FileURLOptions{User: "root", UseSignatureExpiration: &exp}
	url, err := BuildDownloadURL(sandbox, "/home/user/file.txt", opts)
	if err != nil {
		t.Fatalf("BuildDownloadURL: %v", err)
	}
	if !strings.Contains(url, "signature=v1_") {
		t.Errorf("expected signature=v1_ in url, got %s", url)
	}
	if !strings.Contains(url, "signature_expiration=") {
		t.Errorf("expected signature_expiration in url, got %s", url)
	}
}

func TestBuildUploadURL_NoSignature(t *testing.T) {
	domain := "sbx-abc.example.com"
	sandbox := &models.Sandbox{
		SandboxDomain: &domain,
	}
	url, err := BuildUploadURL(sandbox, "/home/user/", nil)
	if err != nil {
		t.Fatalf("BuildUploadURL: %v", err)
	}
	if !strings.HasPrefix(url, "https://sbx-abc.example.com/upload") {
		t.Errorf("expected https://sbx-abc.example.com/upload prefix, got %s", url)
	}
	if !strings.Contains(url, "path=") {
		t.Errorf("expected path in query, got %s", url)
	}
}

func TestBuildDownloadURL_NoDomain(t *testing.T) {
	sandbox := &models.Sandbox{}
	_, err := BuildDownloadURL(sandbox, "/x", nil)
	if err == nil {
		t.Fatal("expected error when sandbox domain is empty")
	}
}

func TestGetHostFromSandbox(t *testing.T) {
	domain := "sbx-abc.example.com"
	sandbox := &models.Sandbox{SandboxDomain: &domain}

	host, err := GetHostFromSandbox(sandbox, 0)
	if err != nil {
		t.Fatalf("GetHostFromSandbox: %v", err)
	}
	if host != "sbx-abc.example.com" {
		t.Errorf("expected sbx-abc.example.com, got %s", host)
	}

	host, err = GetHostFromSandbox(sandbox, 8080)
	if err != nil {
		t.Fatalf("GetHostFromSandbox: %v", err)
	}
	if host != "sbx-abc.example.com:8080" {
		t.Errorf("expected sbx-abc.example.com:8080, got %s", host)
	}
}

func TestGetHostFromSandbox_NoDomain(t *testing.T) {
	_, err := GetHostFromSandbox(&models.Sandbox{}, 0)
	if err == nil {
		t.Fatal("expected error when sandbox domain is empty")
	}
}
