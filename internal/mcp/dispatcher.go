package mcp

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Dispatcher manages multiple MCP server clients
type Dispatcher struct {
	config      *MCPConfig
	loadedPaths []string
	clients     map[string]*Client
	mu          sync.RWMutex
}

// NewDispatcher creates a new Dispatcher
func NewDispatcher(config *MCPConfig, loadedPaths []string) *Dispatcher {
	return &Dispatcher{
		config:      config,
		loadedPaths: loadedPaths,
		clients:     make(map[string]*Client),
	}
}

// GetClient returns a client for the given server name, creating one if needed
func (d *Dispatcher) GetClient(serverName string) (*Client, error) {
	d.mu.RLock()
	client, exists := d.clients[serverName]
	d.mu.RUnlock()

	if exists {
		return client, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists = d.clients[serverName]; exists {
		return client, nil
	}

	cfg, ok := d.config.MCPServers[serverName]
	if !ok {
		return nil, ServerNotFoundErrors(serverName)
	}

	client = NewClient(serverName, cfg)
	d.clients[serverName] = client
	return client, nil
}

// Close closes all client connections
func (d *Dispatcher) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for name, client := range d.clients {
		client.Close()
		delete(d.clients, name)
	}
}

// ConfigPaths returns the loaded config file paths
func (d *Dispatcher) ConfigPaths() []string {
	return d.loadedPaths
}

// ListServersConfig returns server info from config without connecting
func (d *Dispatcher) ListServersConfig() []ServerInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var servers []ServerInfo
	for name, cfg := range d.config.MCPServers {
		transport := InferTransportType(cfg)
		info := ServerInfo{
			Name:      name,
			Transport: transport,
		}

		if transport == TransportStdio {
			info.Command = buildCommandString(cfg.Command, cfg.Args)
		} else {
			info.URL = cfg.URL
		}

		servers = append(servers, info)
	}

	return servers
}

// buildCommandString builds a command string from command and args
func buildCommandString(command string, args []string) string {
	if command == "" {
		return ""
	}
	if len(args) == 0 {
		return command
	}
	var parts []string
	parts = append(parts, command)
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}

// ListAllServers returns server info with tools from all servers
func (d *Dispatcher) ListAllServers(ctx context.Context) []ServerInfo {
	d.mu.RLock()
	serversConfig := make(map[string]ServerConfig)
	for name, cfg := range d.config.MCPServers {
		serversConfig[name] = cfg
	}
	d.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	servers := make([]ServerInfo, 0, len(serversConfig))

	for name, cfg := range serversConfig {
		wg.Add(1)
		go func(name string, cfg ServerConfig) {
			defer wg.Done()

			client := NewClient(name, cfg)
			transport := InferTransportType(cfg)

			info := ServerInfo{
				Name:      name,
				Transport: transport,
			}

			if transport == TransportStdio {
				info.Command = buildCommandString(cfg.Command, cfg.Args)
			} else {
				info.URL = cfg.URL
			}

			// Try to get tools
			if tools, err := client.ListTools(ctx); err != nil {
				info.Error = err.Error()
			} else {
				toolNames := make([]string, len(tools))
				for i, t := range tools {
					toolNames[i] = t.Name
				}
				info.Tools = toolNames
			}

			mu.Lock()
			servers = append(servers, info)
			mu.Unlock()
		}(name, cfg)
	}

	wg.Wait()
	return servers
}

// GetServerInfo returns info for a specific server
func (d *Dispatcher) GetServerInfo(ctx context.Context, serverName string) (*ServerInfo, error) {
	cfg, ok := d.config.MCPServers[serverName]
	if !ok {
		return nil, ServerNotFoundErrors(serverName)
	}

	client, err := d.GetClient(serverName)
	if err != nil {
		return nil, err
	}
	transport := InferTransportType(cfg)

	info := &ServerInfo{
		Name:      serverName,
		Transport: transport,
	}

	if transport == TransportStdio {
		info.Command = buildCommandString(cfg.Command, cfg.Args)
	} else {
		info.URL = cfg.URL
	}

	// Get tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	toolNames := make([]string, len(tools))
	for i, t := range tools {
		toolNames[i] = t.Name
	}
	info.Tools = toolNames

	return info, nil
}

// GetToolInfo returns info for a specific tool on a server
func (d *Dispatcher) GetToolInfo(ctx context.Context, serverName, toolName string) (*ToolMatch, error) {
	_, ok := d.config.MCPServers[serverName]
	if !ok {
		return nil, ServerNotFoundErrors(serverName)
	}

	client, err := d.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	// Find the tool
	for _, t := range tools {
		if t.Name == toolName {
			return &ToolMatch{
				ServerName: serverName,
				Tool: &ToolInfo{
					Name:        t.Name,
					Description: t.Description,
					InputSchema: t.InputSchema,
				},
			}, nil
		}
	}

	return nil, MethodNotFoundErrors(toolName, serverName)
}

// FindTool searches for a tool across all servers
func (d *Dispatcher) FindTool(ctx context.Context, toolName string) ([]ToolMatch, error) {
	d.mu.RLock()
	serversConfig := make(map[string]ServerConfig)
	for name, cfg := range d.config.MCPServers {
		serversConfig[name] = cfg
	}
	d.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var matches []ToolMatch
	var firstErr error

	for name, cfg := range serversConfig {
		wg.Add(1)
		go func(name string, cfg ServerConfig) {
			defer wg.Done()

			client := NewClient(name, cfg)
			tools, err := client.ListTools(ctx)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			for _, t := range tools {
				if t.Name == toolName {
					matches = append(matches, ToolMatch{
						ServerName: name,
						Tool: &ToolInfo{
							Name:        t.Name,
							Description: t.Description,
							InputSchema: t.InputSchema,
						},
					})
				}
			}
			mu.Unlock()
		}(name, cfg)
	}

	wg.Wait()

	if firstErr != nil && len(matches) == 0 {
		return nil, firstErr
	}

	return matches, nil
}

// CallTool calls a tool on a server with retry on network failures
func (d *Dispatcher) CallTool(ctx context.Context, toolName, serverName string, params map[string]any) (string, any, error) {
	_, ok := d.config.MCPServers[serverName]
	if !ok {
		return "", nil, ServerNotFoundErrors(serverName)
	}

	client, err := d.GetClient(serverName)
	if err != nil {
		return "", nil, err
	}

	// Retry on network errors
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := client.CallTool(ctx, toolName, params)
		if err == nil {
			return serverName, result, nil
		}

		lastErr = err

		// Check if it's a network error that might be retryable
		if !isRetryableError(err) {
			// Non-retryable error, return immediately with enhanced message
			return serverName, nil, enhanceCallError(toolName, serverName, err)
		}

		// Don't retry on last attempt
		if attempt < maxRetries {
			// Brief backoff before retry
			select {
			case <-ctx.Done():
				return serverName, nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}

	return serverName, nil, enhanceCallError(toolName, serverName, lastErr)
}

// enhanceCallError enhances error message with tool information
func enhanceCallError(toolName, serverName string, err error) *MCPError {
	mcpErr := CallErrors(toolName, serverName, err)

	// Try to get tool info to suggest parameters
	// Note: This is best-effort and may fail if the server is unreachable
	return mcpErr
}

// isRetryableError checks if an error is a network error that should be retried
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network-related errors that are often transient
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"timeout",
		"temporary failure",
		"i/o timeout",
		"network unreachable",
		"no such host",
		"use of closed network connection",
	}

	errLower := strings.ToLower(errStr)
	for _, pattern := range retryablePatterns {
		if strings.Contains(errLower, pattern) {
			return true
		}
	}

	return false
}

// GetConfig returns the underlying config
func (d *Dispatcher) GetConfig() *MCPConfig {
	return d.config
}

// SuggestMatch represents a suggested tool with reasoning
type SuggestMatch struct {
	ServerName string
	Tool       *ToolInfo
	Reason     string
}

// SearchTools searches for tools across all servers by name or description
func (d *Dispatcher) SearchTools(ctx context.Context, query string) []ToolMatch {
	d.mu.RLock()
	serversConfig := make(map[string]ServerConfig)
	for name, cfg := range d.config.MCPServers {
		serversConfig[name] = cfg
	}
	d.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var matches []ToolMatch

	for name, cfg := range serversConfig {
		wg.Add(1)
		go func(name string, cfg ServerConfig) {
			defer wg.Done()

			client := NewClient(name, cfg)
			tools, err := client.ListTools(ctx)
			if err != nil {
				return
			}

			mu.Lock()
			for _, t := range tools {
				toolNameLower := strings.ToLower(t.Name)
				descLower := strings.ToLower(t.Description)

				// Match if query appears in name or description
				if strings.Contains(toolNameLower, query) || strings.Contains(descLower, query) {
					matches = append(matches, ToolMatch{
						ServerName: name,
						Tool: &ToolInfo{
							Name:        t.Name,
							Description: t.Description,
							InputSchema: t.InputSchema,
						},
					})
				}
			}
			mu.Unlock()
		}(name, cfg)
	}

	wg.Wait()
	return matches
}

// SuggestTools suggests tools based on a task description using keyword matching
func (d *Dispatcher) SuggestTools(ctx context.Context, task string) []SuggestMatch {
	d.mu.RLock()
	serversConfig := make(map[string]ServerConfig)
	for name, cfg := range d.config.MCPServers {
		serversConfig[name] = cfg
	}
	d.mu.RUnlock()

	// Extract keywords from task
	taskLower := strings.ToLower(task)
	taskWords := strings.Fields(taskLower)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var suggestions []SuggestMatch

	for name, cfg := range serversConfig {
		wg.Add(1)
		go func(name string, cfg ServerConfig) {
			defer wg.Done()

			client := NewClient(name, cfg)
			tools, err := client.ListTools(ctx)
			if err != nil {
				return
			}

			mu.Lock()
			for _, t := range tools {
				toolNameLower := strings.ToLower(t.Name)
				descLower := strings.ToLower(t.Description)

				// Check for keyword matches
				var matchedWords []string
				for _, word := range taskWords {
					if len(word) < 3 {
						continue // Skip short words
					}
					if strings.Contains(toolNameLower, word) || strings.Contains(descLower, word) {
						matchedWords = append(matchedWords, word)
					}
				}

				if len(matchedWords) > 0 {
					suggestions = append(suggestions, SuggestMatch{
						ServerName: name,
						Tool: &ToolInfo{
							Name:        t.Name,
							Description: t.Description,
							InputSchema: t.InputSchema,
						},
						Reason: fmt.Sprintf("Matched keywords: %s", strings.Join(matchedWords, ", ")),
					})
				}
			}
			mu.Unlock()
		}(name, cfg)
	}

	wg.Wait()
	return suggestions
}
