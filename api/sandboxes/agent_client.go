package sandboxes

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/scalebox/scalebox-sdk-golang/pb"
	"github.com/scalebox/scalebox-sdk-golang/pb/pbconnect"
)

// authTransport adds Authorization and X-Access-Token headers for Sandbox Agent.
// Pattern matches Backend sandbox_files.go and sandbox_terminal.go.
type authTransport struct {
	transport http.RoundTripper
	authToken string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	reqCopy.Header.Set("Authorization", "Bearer root")
	if t.authToken != "" {
		reqCopy.Header.Set("X-Access-Token", t.authToken)
	}
	return t.transport.RoundTrip(reqCopy)
}

// AgentClientOption configures AgentClient.
type AgentClientOption func(*AgentClient)

// WithHTTPClient sets a custom HTTP client for the Agent connection.
func WithHTTPClient(client *http.Client) AgentClientOption {
	return func(c *AgentClient) {
		c.httpClient = client
	}
}

// AgentClient connects directly to Sandbox Agent via gRPC/Connect.
// Use for Commands, PTY, Code Interpreter, watch_dir.
type AgentClient struct {
	baseURL    string // https://{sandbox.Domain}
	authToken  string
	httpClient *http.Client

	processClient    pbconnect.ProcessClient
	filesystemClient pbconnect.FilesystemClient
	executionClient  pbconnect.ExecutionServiceClient
	contextClient    pbconnect.ContextServiceClient
}

// NewAgentClient creates an Agent client for the given base URL and access token.
// baseURL: https://{sandbox.SandboxDomain} (e.g. https://sbx-xxx.cluster.example.com)
// authToken: sandbox.EnvdAccessToken
func NewAgentClient(baseURL string, authToken string, opts ...AgentClientOption) *AgentClient {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http") {
		baseURL = "https://" + baseURL
	}

	transport := &authTransport{
		transport: http.DefaultTransport,
		authToken: authToken,
	}
	httpClient := &http.Client{
		Transport: transport,
	}

	c := &AgentClient{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: httpClient,
	}
	for _, opt := range opts {
		opt(c)
	}

	// Ensure transport has auth (if custom client was set, wrap its transport)
	if c.httpClient.Transport == nil {
		c.httpClient.Transport = transport
	} else if _, ok := c.httpClient.Transport.(*authTransport); !ok {
		c.httpClient.Transport = &authTransport{
			transport: c.httpClient.Transport,
			authToken: authToken,
		}
	}

	c.processClient = pbconnect.NewProcessClient(c.httpClient, baseURL)
	c.filesystemClient = pbconnect.NewFilesystemClient(c.httpClient, baseURL)
	c.executionClient = pbconnect.NewExecutionServiceClient(c.httpClient, baseURL)
	c.contextClient = pbconnect.NewContextServiceClient(c.httpClient, baseURL)

	return c
}

// Process returns the Process client for Commands and PTY.
func (c *AgentClient) Process() pbconnect.ProcessClient {
	return c.processClient
}

// Filesystem returns the Filesystem client for watch_dir and filesystem RPCs.
func (c *AgentClient) Filesystem() pbconnect.FilesystemClient {
	return c.filesystemClient
}

// Execution returns the ExecutionService client for run_code.
func (c *AgentClient) Execution() pbconnect.ExecutionServiceClient {
	return c.executionClient
}

// Context returns the ContextService client for create/destroy_code_context.
func (c *AgentClient) Context() pbconnect.ContextServiceClient {
	return c.contextClient
}

// Commands returns a CommandsClient for command execution.
func (c *AgentClient) Commands() *CommandsClient {
	return &CommandsClient{agent: c}
}

// PTY returns a PTYClient for PTY operations.
func (c *AgentClient) PTY() *PTYClient {
	return &PTYClient{agent: c}
}

// CodeInterpreter returns a CodeInterpreterClient for code execution.
func (c *AgentClient) CodeInterpreter() *CodeInterpreterClient {
	return &CodeInterpreterClient{agent: c}
}

// List lists running processes (Commands and PTY sessions).
func (c *AgentClient) List(ctx context.Context) (*pb.ListResponse, error) {
	resp, err := c.processClient.List(ctx, connect.NewRequest(&pb.ListRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}
	return resp.Msg, nil
}

// ConnectToAgent creates an AgentClient for the given sandbox.
// It fetches sandbox details via REST to get domain and access token.
// The sandbox must be running with internet access.
func (c *Client) ConnectToAgent(ctx context.Context, sandboxID string) (*AgentClient, error) {
	sandbox, err := c.Get(ctx, sandboxID)
	if err != nil {
		return nil, fmt.Errorf("get sandbox: %w", err)
	}
	if sandbox.Status != "running" {
		return nil, fmt.Errorf("sandbox must be running (current: %s)", sandbox.Status)
	}
	if !sandbox.AllowInternetAccess {
		return nil, fmt.Errorf("sandbox must allow internet access for agent connection")
	}
	if sandbox.SandboxDomain == nil || *sandbox.SandboxDomain == "" {
		return nil, fmt.Errorf("sandbox domain not available")
	}
	authToken := ""
	if sandbox.EnvdAccessToken != nil && *sandbox.EnvdAccessToken != "" {
		authToken = *sandbox.EnvdAccessToken
	}
	baseURL := *sandbox.SandboxDomain
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return NewAgentClient(baseURL, authToken), nil
}
