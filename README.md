# mcp2cli

> A powerful CLI tool for interacting with MCP (Model Context Protocol) servers

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

## Overview

mcp2cli is a command-line interface for the Model Context Protocol, enabling developers to discover, explore, and invoke tools from any MCP server directly from the terminal.

```
┌─────────────────────────────────────────────────────────────┐
│                     mcp2cli                                │
│                                                             │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐                │
│  │  List   │───▶│ Inspect │───▶│  Call   │                │
│  │ Servers │    │  Tools  │    │  Tools  │                │
│  └─────────┘    └─────────┘    └─────────┘                │
│                                                             │
│         SSE · Streamable HTTP · Stdio                       │
└─────────────────────────────────────────────────────────────┘
```

## Features

- **🔍 Discover Servers** - List all configured MCP servers without connecting
- **📋 Explore Tools** - View detailed tool information with formatted parameters
- **🚀 Invoke Tools** - Call tools directly with type-safe arguments
- **🔌 Multiple Transports** - Support for SSE, Streamable HTTP, and Stdio
- **📁 Smart Config** - Auto-detects and merges configs from standard locations
- **📤 Unified JSON Output** - Machine-readable output for scripting

## Quick Start

### Installation

```bash
go install github.com/weibaohui/mcp2cli@latest
```

### Configuration

Create `~/.config/mcp/config.json`:

```json
{
  "mcpServers": {
    "openDeepWiki": {
      "url": "https://opendeepwiki.k8m.site/mcp/streamable",
      "timeout": 30000
    }
  }
}
```

### Usage

```bash
# List all configured servers
mcp

# List tools from a specific server
mcp openDeepWiki

# View tool details
mcp openDeepWiki list_repositories

# Call a tool with arguments
mcp openDeepWiki list_repositories limit=3
```

## Config File Search Paths

mcp2cli searches for configuration in the following locations (priority order):

| Platform | Paths |
|----------|-------|
| macOS/Linux | `~/.config/modelcontextprotocol/mcp.json` |
|  | `~/.config/mcp/config.json` |
|  | `./mcp.json` |
|  | `./.mcp/config.json` |
|  | `/etc/mcp/config.json` |
| Windows | `%APPDATA%\modelcontextprotocol\mcp.json` |
|  | `%APPDATA%\mcp\config.json` |
|  | `%USERPROFILE%\.mcp\config.json` |

## Config File Format

```json
{
  "mcpServers": {
    "serverName": {
      "transport": "streamable",
      "url": "https://example.com/mcp",
      "command": "npx",
      "args": ["-y", "@server/mcp"],
      "env": {"KEY": "value"},
      "timeout": 30000
    }
  }
}
```

### Transport Types

| Type | Description |
|------|-------------|
| `streamable` | Modern streaming HTTP (default) |
| `sse` | Server-Sent Events over HTTP |
| `stdio` | Local subprocess communication |

### Argument Format

Arguments can be specified in two formats:

```bash
# String (default)
mcp server tool name=John

# With type annotation
mcp server tool name:string=John age:number=30 enabled:bool=true
```

## Output Format

All commands return unified JSON output:

```json
{
  "success": true,
  "data": {
    "configFiles": ["/home/user/.config/mcp/config.json"],
    "servers": [
      {
        "name": "openDeepWiki",
        "transport": "streamable",
        "url": "https://opendeepwiki.k8m.site/mcp/streamable"
      }
    ]
  },
  "meta": {
    "timestamp": "2026-03-24T10:00:00Z",
    "version": "v0.2.8"
  }
}
```

## Architecture

```
cmd/mcp/main.go          # CLI entry point, argument routing
internal/mcp/
  ├── types.go           # Error codes, shared types
  ├── config.go          # Config loading & merging
  ├── config_paths.go    # Platform-specific paths
  ├── client.go          # MCP server client
  ├── dispatcher.go      # Multi-server coordination
  └── formatter.go       # Schema formatting, arg parsing
```

## Development

```bash
# Clone the repository
git clone https://github.com/weibaohui/mcp2cli.git
cd mcp2cli

# Build
go build -o mcp ./cmd/mcp/

# Run tests
go test ./...

# Run with custom config
mcp
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related

- [Model Context Protocol Specification](https://modelcontextprotocol.io)
- [Official Go SDK](https://github.com/modelcontextprotocol/go-sdk)
