package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
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
		return mcp.NewSSEClientTransport(c.config.URL, nil), nil

	case TransportStreamable:
		if c.config.URL == "" {
			return nil, fmt.Errorf("streamable transport requires url")
		}
		return mcp.NewStreamableClientTransport(c.config.URL, nil), nil

	default:
		return nil, fmt.Errorf("unknown transport: %s", c.transport)
	}
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
