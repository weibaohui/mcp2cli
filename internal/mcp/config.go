package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ServerConfig represents a server configuration
type ServerConfig struct {
	Transport string            `json:"transport,omitempty"`
	Type      string            `json:"type,omitempty"`
	URL       string            `json:"url,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Timeout   int               `json:"timeout,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Auth      AuthConfig        `json:"auth,omitempty"`
}

// AuthConfig represents authentication configuration for MCP servers
type AuthConfig struct {
	// OAuth 2.1 + PKCE configuration
	OAuth *OAuthConfig `json:"oauth,omitempty"`
}

// OAuthConfig holds OAuth 2.1 authentication parameters
type OAuthConfig struct {
	// AccessToken is a static access token to use for authentication.
	// If provided, it will be used directly as a Bearer token.
	AccessToken string `json:"accessToken,omitempty"`
	// ClientID is the OAuth client ID (for full OAuth flow).
	ClientID string `json:"clientId,omitempty"`
	// ClientSecret is the OAuth client secret (for confidential clients).
	ClientSecret string `json:"clientSecret,omitempty"`
	// AuthorizationURL is the authorization server's authorization endpoint.
	AuthorizationURL string `json:"authorizationURL,omitempty"`
	// TokenURL is the authorization server's token endpoint.
	TokenURL string `json:"tokenURL,omitempty"`
	// Scopes are the OAuth scopes to request.
	Scopes string `json:"scopes,omitempty"`
	// RedirectURL is the callback URL for OAuth (defaults to http://localhost:7777/callback).
	RedirectURL string `json:"redirectURL,omitempty"`
	// ClientIDMetadataURL is the URL for Client ID Metadata Document (MCP spec).
	ClientIDMetadataURL string `json:"clientIdMetadataUrl,omitempty"`
}

// MCPConfig represents the full MCP configuration
type MCPConfig struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// LoadConfig loads and merges all config files from search paths
func LoadConfig() (*MCPConfig, []string, error) {
	paths := GetConfigSearchPaths()
	return LoadConfigFromPaths(paths)
}

// LoadConfigFromPaths loads and merges config files from specified paths
func LoadConfigFromPaths(paths []string) (*MCPConfig, []string, error) {
	result := &MCPConfig{
		MCPServers: make(map[string]ServerConfig),
	}

	var loadedPaths []string

	for _, path := range paths {
		// Expand ~ in path
		path = ExpandHome(path)

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		// Read file
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Parse JSON
		var cfg MCPConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}

		// Merge servers (higher priority paths first, so they overwrite)
		for name, serverCfg := range cfg.MCPServers {
			result.MCPServers[name] = serverCfg
		}

		loadedPaths = append(loadedPaths, path)
	}

	if len(loadedPaths) == 0 {
		return nil, loadedPaths, ConfigErrors("no config files found")
	}

	return result, loadedPaths, nil
}

// GetServerConfig returns config for a specific server
func (c *MCPConfig) GetServerConfig(name string) (ServerConfig, bool) {
	cfg, ok := c.MCPServers[name]
	return cfg, ok
}

// Transport type constants
const (
	TransportSSE        = "sse"
	TransportStreamable = "streamable"
	TransportStdio      = "stdio"
)

// InferTransportType infers transport type from server config
func InferTransportType(cfg ServerConfig) string {
	// 1. Explicit transport field
	if cfg.Transport != "" {
		return normalizeTransportType(cfg.Transport)
	}

	// 2. type field alias
	if cfg.Type != "" {
		return normalizeTransportType(cfg.Type)
	}

	// 3. Has command → stdio
	if cfg.Command != "" {
		return TransportStdio
	}

	// 4. URL contains "sse" (case insensitive)
	if strings.Contains(strings.ToLower(cfg.URL), "sse") {
		return TransportSSE
	}

	// 5. URL contains "stream" → streamable
	if strings.Contains(strings.ToLower(cfg.URL), "stream") {
		return TransportStreamable
	}

	// 6. Default streamable
	return TransportStreamable
}

// normalizeTransportType normalizes transport type aliases
func normalizeTransportType(t string) string {
	lower := strings.ToLower(t)

	if strings.Contains(lower, "stream") {
		return TransportStreamable
	}
	if strings.Contains(lower, "sse") {
		return TransportSSE
	}
	if strings.Contains(lower, "command") || strings.Contains(lower, "stdio") {
		return TransportStdio
	}

	return t
}

// SaveConfig saves the config to a specific path
func SaveConfig(config *MCPConfig, path string) error {
	path = ExpandHome(path)

	// Ensure directory exists
	dir := strings.TrimSuffix(path, "/config.json")
	dir = strings.TrimSuffix(dir, "/mcp.json")
	if dir != path {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
