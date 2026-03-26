package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerInfo represents information about an MCP server
type ServerInfo struct {
	Name      string   `json:"name"`
	Transport string   `json:"transport"`
	URL       string   `json:"url,omitempty"`
	Command   string   `json:"command,omitempty"`
	Tools     []string `json:"tools,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// ToolInfo represents information about a tool
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

// ToolMatch represents a matched tool with its server
type ToolMatch struct {
	ServerName string
	Tool       *ToolInfo
}

// ParamInfo represents structured parameter information
type ParamInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// Client represents an MCP server client
type Client struct {
	name      string
	config    ServerConfig
	transport string
	session   *mcp.ClientSession
	toolCache []*mcp.Tool
}

// NewClient creates a new MCP client for a server
func NewClient(name string, config ServerConfig) *Client {
	return &Client{
		name:      name,
		config:    config,
		transport: InferTransportType(config),
	}
}

// Connect establishes connection to the MCP server
func (c *Client) Connect(ctx context.Context) error {
	timeout := time.Duration(c.config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	transport, err := c.buildTransport()
	if err != nil {
		return err
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp2cli",
		Version: "v0.2.8",
	}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return ConnectErrors(c.name, err.Error())
	}

	c.session = session
	return nil
}

// buildTransport creates the appropriate transport based on config
func (c *Client) buildTransport() (mcp.Transport, error) {
	switch c.transport {
	case TransportStdio:
		if c.config.Command == "" {
			return nil, fmt.Errorf("stdio transport requires command")
		}
		// Build exec.Cmd
		cmdParts := []string{c.config.Command}
		cmdParts = append(cmdParts, c.config.Args...)
		cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
		if c.config.Env != nil {
			cmd.Env = buildEnvList(c.config.Env)
		}
		return &mcp.CommandTransport{Command: cmd}, nil

	case TransportSSE:
		if c.config.URL == "" {
			return nil, fmt.Errorf("sse transport requires url")
		}
		transport := &mcp.SSEClientTransport{Endpoint: c.config.URL}
		if headers := c.resolveHeaders(); len(headers) > 0 {
			transport.HTTPClient = c.newHTTPClientWithHeaders(headers)
		}
		return transport, nil

	case TransportStreamable:
		if c.config.URL == "" {
			return nil, fmt.Errorf("streamable transport requires url")
		}
		transport := &mcp.StreamableClientTransport{
			Endpoint: c.config.URL,
		}
		// Get headers from config
		headers := c.resolveHeaders()

		// Configure OAuth - this handles accessToken and full OAuth config
		oauthToken, err := configureOAuth(c.config.Auth.OAuth)
		if err != nil {
			return nil, err
		}

		// Add OAuth token to headers if present
		if oauthToken != "" {
			if headers == nil {
				headers = make(map[string]string)
			}
			if _, hasAuth := headers["Authorization"]; !hasAuth {
				headers["Authorization"] = "Bearer " + oauthToken
			}
		}

		if len(headers) > 0 {
			transport.HTTPClient = c.newHTTPClientWithHeaders(headers)
		}
		return transport, nil

	default:
		return nil, fmt.Errorf("unknown transport: %s", c.transport)
	}
}

// resolveHeaders resolves environment variables in header values
// Supports ${VAR} and $VAR syntax
func (c *Client) resolveHeaders() map[string]string {
	if c.config.Headers == nil {
		return nil
	}

	headers := make(map[string]string)
	envVarPattern := regexp.MustCompile(`\$\{([^}]+)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)

	for key, value := range c.config.Headers {
		resolved := envVarPattern.ReplaceAllStringFunc(value, func(match string) string {
			var varName string
			if len(match) > 2 && match[0] == '$' && match[1] == '{' {
				// ${VAR} syntax
				varName = match[2 : len(match)-1]
			} else if len(match) > 1 && match[0] == '$' {
				// $VAR syntax
				varName = match[1:]
			} else {
				return match
			}
			if envValue, exists := os.LookupEnv(varName); exists {
				return envValue
			}
			return match
		})
		headers[key] = resolved
	}

	return headers
}

// newHTTPClientWithHeaders creates an HTTP client with custom headers
func (c *Client) newHTTPClientWithHeaders(headers map[string]string) *http.Client {
	return &http.Client{
		Transport: &headerTransport{
			headers: headers,
			base:    http.DefaultTransport,
		},
	}
}

// headerTransport is an http.RoundTripper that adds custom headers
type headerTransport struct {
	headers map[string]string
	base    http.RoundTripper
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req = req.Clone(req.Context())
	for key, value := range t.headers {
		req.Header.Set(key, value)
	}
	return t.base.RoundTrip(req)
}

// buildEnvList converts env map to slice for exec.Cmd
func buildEnvList(env map[string]string) []string {
	var result []string
	for k, v := range env {
		result = append(result, k+"="+v)
	}
	return result
}

// ListTools returns all available tools from the server
func (c *Client) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	if c.session == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	timeout := time.Duration(c.config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result, err := c.session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return nil, ConnectErrors(c.name, err.Error())
	}

	c.toolCache = result.Tools
	return result.Tools, nil
}

// CallTool calls a tool with the given name and parameters
func (c *Client) CallTool(ctx context.Context, name string, params map[string]any) (*mcp.CallToolResult, error) {
	if c.session == nil {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	timeout := time.Duration(c.config.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Convert params to proper format
	var args any = params
	// If params is empty, set to nil
	if len(params) == 0 {
		args = nil
	}

	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: args,
	})
	if err != nil {
		return nil, CallErrors(name, c.name, err)
	}

	return result, nil
}

// Close closes the client connection
func (c *Client) Close() {
	if c.session != nil {
		c.session.Close()
	}
}

// GetTransport returns the transport type
func (c *Client) GetTransport() string {
	return c.transport
}

// GetConfig returns the server config
func (c *Client) GetConfig() ServerConfig {
	return c.config
}

// FormatToolForOutput formats a tool for JSON output
func FormatToolForOutput(tool *mcp.Tool) *ToolInfo {
	info := &ToolInfo{
		Name:        tool.Name,
		Description: tool.Description,
	}

	// Convert input schema
	if tool.InputSchema != nil {
		if schemaJSON, err := json.Marshal(tool.InputSchema); err == nil {
			var schema any
			if err := json.Unmarshal(schemaJSON, &schema); err == nil {
				info.InputSchema = schema
			}
		}
	}

	return info
}
