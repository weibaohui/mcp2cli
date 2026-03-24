package mcp

import (
	"context"
	"strings"
	"sync"
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
	d.mu.RLock()
	cfg, ok := d.config.MCPServers[serverName]
	d.mu.RUnlock()

	if !ok {
		return nil, ServerNotFoundErrors(serverName)
	}

	client := NewClient(serverName, cfg)
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
	d.mu.RLock()
	cfg, ok := d.config.MCPServers[serverName]
	d.mu.RUnlock()

	if !ok {
		return nil, ServerNotFoundErrors(serverName)
	}

	client := NewClient(serverName, cfg)

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

// CallTool calls a tool on a server
func (d *Dispatcher) CallTool(ctx context.Context, toolName, serverName string, params map[string]any) (string, any, error) {
	d.mu.RLock()
	cfg, ok := d.config.MCPServers[serverName]
	d.mu.RUnlock()

	if !ok {
		return "", nil, ServerNotFoundErrors(serverName)
	}

	client := NewClient(serverName, cfg)

	result, err := client.CallTool(ctx, toolName, params)
	if err != nil {
		return serverName, nil, err
	}

	return serverName, result, nil
}

// GetConfig returns the underlying config
func (d *Dispatcher) GetConfig() *MCPConfig {
	return d.config
}
