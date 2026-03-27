# mcp2cli

> A powerful CLI tool for interacting with MCP (Model Context Protocol) servers

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

## Quick Usage

```bash
# Install via npm (recommended)
npm install -g @weibaohui/mcp2cli

# Or install via Go
go install github.com/weibaohui/mcp2cli@latest

# Rename to mcp for convenience
mv $(go env GOPATH)/bin/mcp2cli $(go env GOPATH)/bin/mcp

# List configured servers (no server connection)
mcp

# Inspect a server's available tools
mcp openDeepWiki

# View tool details with parameter examples
mcp openDeepWiki list_repositories

# Call a tool
mcp openDeepWiki list_repositories limit=3

# Call with typed arguments (recommended)
mcp openDeepWiki list_repositories limit:number=3 enabled:bool=true
```

## Argument Format

Arguments can be in two formats:

```bash
# Simple key=value (string type by default)
mcp server tool name=John age=30

# Typed key:type=value (recommended for precision)
mcp server tool name:string=John age:number=30 enabled:bool=true
```

**Supported types:** `string`, `number`, `int`, `float`, `bool`

---

## Features

- **🔍 Discover Servers** - List all configured MCP servers without connecting
- **📋 Explore Tools** - View detailed tool information with formatted parameters
- **🚀 Invoke Tools** - Call tools directly with type-safe arguments
- **🔌 Multiple Transports** - Support for SSE, Streamable HTTP, and Stdio
- **📁 Smart Config** - Auto-detects and merges configs from standard locations
- **📤 Unified JSON Output** - Machine-readable output for scripting

## Installation

### npm (multi-platform)

```bash
npm install -g @weibaohui/mcp2cli
```

Supports Linux, macOS, Windows on amd64/arm64.

### Binary (latest release)

Download from [GitHub Releases](https://github.com/weibaohui/mcp2cli/releases/latest)

### From source

```bash
go install github.com/weibaohui/mcp2cli@latest

# Rename to mcp for convenience
mv $(go env GOPATH)/bin/mcp2cli $(go env GOPATH)/bin/mcp
```

### Build from source

```bash
git clone https://github.com/weibaohui/mcp2cli.git
cd mcp2cli
make build
```

## Configuration

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

### Config File Search Paths

| Platform | Priority Order |
|----------|---------------|
| macOS/Linux | `~/.config/modelcontextprotocol/mcp.json` → `~/.config/mcp/config.json` → `./mcp.json` → `./.mcp/config.json` → `/etc/mcp/config.json` |
| Windows | `%APPDATA%\modelcontextprotocol\mcp.json` → `%APPDATA%\mcp\config.json` → `%USERPROFILE%\.mcp\config.json` → `.\mcp.json` → `.\.mcp\config.json` |

### Config File Format

```json
{
  "mcpServers": {
    "serverName": {
      "transport": "streamable",
      "url": "https://example.com/mcp",
      "command": "npx",
      "args": ["-y", "@server/mcp"],
      "env": {"KEY": "value"},
      "timeout": 30000,
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}",
        "X-API-Key": "${API_KEY}"
      }
    }
  }
}
```

### Authentication

**HTTP Headers (API Key / Bearer Token):**
```json
{
  "mcpServers": {
    "secure-server": {
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}",
        "X-API-Key": "${API_KEY}"
      }
    }
  }
}
```

Supports `${VAR}` and `$VAR` environment variable substitution.

**OAuth 2.1 with Static Access Token:**
```json
{
  "mcpServers": {
    "oauth-server": {
      "url": "https://api.example.com/mcp",
      "auth": {
        "oauth": {
          "accessToken": "${OAUTH_ACCESS_TOKEN}"
        }
      }
    }
  }
}
```

**Transport Types:**
| Type | Description |
|------|-------------|
| `streamable` | Modern streaming HTTP (default) |
| `sse` | Server-Sent Events over HTTP |
| `stdio` | Local subprocess communication |

## Command Reference

```bash
# List all configured servers
mcp

# Get server info with tools list
mcp <server_name>

# Get tool details with parameter examples
mcp <server_name> <tool_name>

# Call a tool with arguments
mcp <server_name> <tool_name> <key=value> [key2=value2]...
```

## Output Format

All commands return unified JSON:

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
main.go                   # CLI entry point, argument routing
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
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Lint code
make lint
```

## License

MIT License - see [LICENSE](LICENSE) for details.
