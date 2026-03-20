package sandboxes

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/scalebox/scalebox-sdk-golang/models"
)

const (
	envdDownloadRoute = "/download"
	envdUploadRoute   = "/upload"
)

// FileURLOptions configures download/upload URL generation.
type FileURLOptions struct {
	User                  string // "root" or "user", default "root"
	UseSignatureExpiration *int   // seconds from now for URL expiry
}

// computeSignature generates a v1 signature for sandbox file URLs.
// Format: path:operation:user:token[:expiration] -> SHA256 -> base64url (no padding) -> "v1_" + encoded
func computeSignature(path, operation, user, token string, expiration *int) (sig string, exp *int) {
	var raw string
	if expiration != nil {
		raw = fmt.Sprintf("%s:%s:%s:%s:%d", path, operation, user, token, *expiration)
	} else {
		raw = fmt.Sprintf("%s:%s:%s:%s", path, operation, user, token)
	}
	digest := sha256.Sum256([]byte(raw))
	encoded := base64.RawStdEncoding.EncodeToString(digest[:])
	return "v1_" + encoded, expiration
}

// buildEnvdBaseURL returns https://{domain} from sandbox.
func buildEnvdBaseURL(sandbox *models.Sandbox) (string, error) {
	if sandbox == nil || sandbox.SandboxDomain == nil || *sandbox.SandboxDomain == "" {
		return "", fmt.Errorf("sandbox domain not available")
	}
	domain := strings.TrimPrefix(*sandbox.SandboxDomain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	return "https://" + domain, nil
}

// BuildDownloadURL builds a download URL for a file in the sandbox.
// Caller provides sandbox (e.g. from Get). If EnvdAccessToken is set, uses signature auth.
func BuildDownloadURL(sandbox *models.Sandbox, path string, opts *FileURLOptions) (string, error) {
	base, err := buildEnvdBaseURL(sandbox)
	if err != nil {
		return "", err
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		path = "/"
	}
	user := "root"
	if opts != nil && opts.User != "" {
		user = opts.User
	}

	u := base + envdDownloadRoute + "/" + path
	q := url.Values{}
	q.Set("username", user)

	token := ""
	if sandbox.EnvdAccessToken != nil && *sandbox.EnvdAccessToken != "" {
		token = *sandbox.EnvdAccessToken
	}
	if token != "" {
		var exp *int
		if opts != nil && opts.UseSignatureExpiration != nil {
			sec := int(time.Now().Unix()) + *opts.UseSignatureExpiration
			exp = &sec
		}
		sig, _ := computeSignature(path, "read", user, token, exp)
		q.Set("signature", sig)
		if exp != nil {
			q.Set("signature_expiration", strconv.Itoa(*exp))
		}
	}

	return u + "?" + q.Encode(), nil
}

// BuildUploadURL builds an upload URL for a file in the sandbox.
// path may be empty; if empty, path is provided when uploading.
func BuildUploadURL(sandbox *models.Sandbox, path string, opts *FileURLOptions) (string, error) {
	base, err := buildEnvdBaseURL(sandbox)
	if err != nil {
		return "", err
	}
	user := "root"
	if opts != nil && opts.User != "" {
		user = opts.User
	}

	u := base + envdUploadRoute
	q := url.Values{}
	q.Set("username", user)
	if path != "" {
		q.Set("path", strings.TrimPrefix(path, "/"))
	}

	token := ""
	if sandbox.EnvdAccessToken != nil && *sandbox.EnvdAccessToken != "" {
		token = *sandbox.EnvdAccessToken
	}
	if token != "" {
		var exp *int
		if opts != nil && opts.UseSignatureExpiration != nil {
			sec := int(time.Now().Unix()) + *opts.UseSignatureExpiration
			exp = &sec
		}
		pathForSig := path
		if pathForSig == "" {
			pathForSig = "/"
		}
		sig, _ := computeSignature(strings.TrimPrefix(pathForSig, "/"), "write", user, token, exp)
		q.Set("signature", sig)
		if exp != nil {
			q.Set("signature_expiration", strconv.Itoa(*exp))
		}
	}

	return u + "?" + q.Encode(), nil
}

// GetHostFromSandbox returns the host address for connecting to the sandbox.
// port is optional; if > 0, returns "domain:port", otherwise "domain".
func GetHostFromSandbox(sandbox *models.Sandbox, port int) (string, error) {
	if sandbox == nil || sandbox.SandboxDomain == nil || *sandbox.SandboxDomain == "" {
		return "", fmt.Errorf("sandbox domain not available")
	}
	domain := strings.TrimPrefix(*sandbox.SandboxDomain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimSuffix(domain, "/")
	if port > 0 {
		return fmt.Sprintf("%s:%d", domain, port), nil
	}
	return domain, nil
}
