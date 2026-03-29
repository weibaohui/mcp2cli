# MCP (Model Context Protocol) CLI Implementation

## 1. Overview

This is an implementation of `mcp` command (mcp2cli) for interacting with MCP servers via CLI.

## 2. Architecture

### 2.1 Layered Structure

```
CLI Layer (cmd/mcp/main.go)
    - Argument parsing (0/1/2/3+ param modes)
    - Output formatting (JSON unified output)
         │
Dispatcher Layer (internal/mcp/dispatcher.go)
    - ListServersConfig()    │ Read-only config, no server connection
    - ListAllServers()       │ Concurrent connection to get tools
    - GetServerInfo()        │ Single server info
    - GetToolInfo()          │ Tool details
    - FindTool()             │ Cross-server tool search
    - CallTool()             │ Tool invocation
         │
Client Layer (internal/mcp/client.go)
    - buildTransport()       │ Create transport based on config
    - listTools()            │ Get tools list
    - callTool()             │ Call specified tool
         │
Transport Layer (go-sdk)
    - SSEClientTransport     │ Traditional HTTP/SSE
    - StreamableClientTransport │ Modern streaming HTTP
    - CommandTransport       │ Local subprocess (stdio)
```

### 2.2 File Responsibilities

| File | Responsibility |
|------|----------------|
| `cmd/mcp/main.go` | cobra command definition, argument routing, output formatting |
| `internal/mcp/types.go` | Shared structs, error codes, utility functions |
| `internal/mcp/config.go` | Config file search and loading |
| `internal/mcp/config_paths.go` | Platform-specific path construction |
| `internal/mcp/client.go` | Single MCP server client |
| `internal/mcp/dispatcher.go` | Multi-server dispatch coordination |

## 3. Configuration System

### 3.1 Config File Search Paths

**Linux/macOS (priority high to low):**
1. `~/.config/modelcontextprotocol/mcp.json`
2. `~/.config/mcp/config.json`
3. `./mcp.json` (current directory)
4. `./.mcp/config.json` (current directory)
5. `/etc/mcp/config.json` (system-level)

**Windows:**
1. `%APPDATA%\modelcontextprotocol\mcp.json`
2. `%APPDATA%\mcp\config.json`
3. `%USERPROFILE%\.mcp\config.json`
4. `.\mcp.json`
5. `.\.mcp\config.json`
6. `%ProgramData%\mcp\config.json`

### 3.2 Config File Structure

```json
{
  "mcpServers": {
    "serverName": {
      "transport": "sse|streamable|stdio",
      "type": "streamable-http",
      "url": "https://example.com/mcp",
      "command": "npx",
      "args": ["-y", "@server/mcp"],
      "env": {"KEY": "value"},
      "timeout": 30000,
      "headers": {
        "Authorization": "Bearer ${API_TOKEN}",
        "X-API-Key": "${API_KEY}"
      },
      "auth": {
        "oauth": {
          "clientId": "my-client-id",
          "clientSecret": "${OAUTH_CLIENT_SECRET}",
          "authorizationURL": "https://auth.example.com/authorize",
          "tokenURL": "https://auth.example.com/token",
          "scopes": "openid profile"
        }
      }
    }
  }
}
```

**示例配置：**

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

### 3.3 Authentication

#### 3.3.1 HTTP Headers Authentication

 Supports custom HTTP headers for API key / Bearer token authentication:

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

**环境变量替换**：支持 `${VAR}` 和 `$VAR` 语法。

#### 3.3.2 OAuth 2.1 + PKCE

OAuth 配置支持以下两种模式：

**1. 静态访问令牌（推荐用于 CLI）**
```json
{
  "mcpServers": {
    "server-with-token": {
      "url": "https://api.example.com/mcp",
      "auth": {
        "oauth": {
          "accessToken": "${MY_ACCESS_TOKEN}"
        }
      }
    }
  }
}
```

**2. 完整 OAuth 授权码流程**（需要用户交互）
```json
{
  "mcpServers": {
    "oauth-server": {
      "url": "https://api.example.com/mcp",
      "auth": {
        "oauth": {
          "clientId": "my-client-id",
          "clientSecret": "${OAUTH_CLIENT_SECRET}",
          "authorizationURL": "https://auth.example.com/authorize",
          "tokenURL": "https://auth.example.com/token",
          "scopes": "openid profile"
        }
      }
    }
  }
}
```

#### 3.3.3 Stdio Transport 认证

对于本地 subprocess 传输，使用 `env` 字段传递环境变量：

```json
{
  "mcpServers": {
    "local-server": {
      "command": "npx",
      "args": ["-y", "@server/mcp"],
      "env": {
        "API_TOKEN": "${API_TOKEN}"
      }
    }
  }
}
```

### 3.4 Transport Type Inference

### 3.3 Transport Type Inference

```go
func InferTransportType(cfg ServerConfig) string {
    // 1. Explicit transport field
    if cfg.Transport != "" {
        return cfg.Transport
    }
    // 2. type field alias mapping
    if cfg.Type != "" {
        return normalizeTransportType(cfg.Type)
    }
    // 3. Has command → stdio
    if cfg.Command != "" {
        return "stdio"
    }
    // 4. URL contains "sse" (case insensitive)
    if strings.Contains(strings.ToLower(cfg.URL), "sse") {
        return "sse"
    }
    // 5. URL contains "stream" → streamable
    if strings.Contains(strings.ToLower(cfg.URL), "stream") {
        return "streamable"
    }
    // 6. Default streamable
    return "streamable"
}
```

### 3.4 Config Merge Strategy

Higher priority paths overwrite lower priority for conflicting server names.

## 4. Core Data Structures

### 4.1 MCPConfig

```go
type MCPConfig struct {
    MCPServers map[string]ServerConfig `json:"mcpServers"`
}
```

### 4.2 ServerConfig (for config parsing)

```go
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

type AuthConfig struct {
    OAuth *OAuthConfig `json:"oauth,omitempty"`
}

type OAuthConfig struct {
    AccessToken      string `json:"accessToken,omitempty"`
    ClientID         string `json:"clientId,omitempty"`
    ClientSecret     string `json:"clientSecret,omitempty"`
    AuthorizationURL string `json:"authorizationURL,omitempty"`
    TokenURL         string `json:"tokenURL,omitempty"`
    Scopes           string `json:"scopes,omitempty"`
}
```

### 4.3 ServerInfo (for output)

```go
type ServerInfo struct {
    Name      string   `json:"name"`
    Transport string   `json:"transport"`
    URL       string   `json:"url,omitempty"`
    Command   string   `json:"command,omitempty"`
    Tools     []string `json:"tools,omitempty"`
    Error     string   `json:"error,omitempty"`
}
```

### 4.4 ToolInfo

```go
type ToolInfo struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
    InputSchema any    `json:"inputSchema,omitempty"`
}
```

### 4.5 ToolMatch

```go
type ToolMatch struct {
    ServerName string
    Tool       *ToolInfo
}
```

### 4.6 ParamInfo

```go
type ParamInfo struct {
    Name        string `json:"name"`
    Type        string `json:"type"`
    Required    bool   `json:"required"`
    Description string `json:"description,omitempty"`
}
```

### 4.7 MCPError

```go
type MCPError struct {
    Code    string
    Message string
    Details map[string]any
}

const (
    ErrCodeConfigNotFound  = "MCP_CONFIG_NOT_FOUND"
    ErrCodeConnectFailed   = "MCP_CONNECT_FAILED"
    ErrCodeServerNotFound  = "MCP_SERVER_NOT_FOUND"
    ErrCodeMethodNotFound  = "MCP_METHOD_NOT_FOUND"
    ErrCodeMethodAmbiguous = "MCP_METHOD_AMBIGUOUS"
    ErrCodeCallFailed      = "MCP_CALL_FAILED"
    ErrCodeParamInvalid    = "MCP_PARAM_INVALID"
)
```

## 5. CLI Interface

### 5.1 Four Operation Modes

| Args Count | Mode | Handler |
|------------|------|---------|
| 0 | list | runMCPList() |
| 1 | server-info | runMCPServerInfo(serverName) |
| 2 | tool-info | runMCPInfo(serverName, toolName) |
| 3+ | call | runMCPCall(serverName, toolName, args...) |

### 5.2 Help Text

```
Interact with MCP (Model Context Protocol) Servers

Config file search paths (by priority):
  1. ~/.config/modelcontextprotocol/mcp.json
  2. ~/.config/mcp/config.json
  3. ./mcp.json
  4. ./.mcp/config.json
  5. /etc/mcp/config.json

Usage examples:

  # List all configured servers (config only, no tool fetching)
  mcp

  # List tools for a specific server
  mcp openDeepWiki

  # View details of a specific tool
  mcp openDeepWiki list_repositories

  # Call a tool (args format: key=value or key:type=value)
  mcp openDeepWiki list_repositories limit=3

  # Call with YAML parameters (inline)
  mcp openDeepWiki list_repositories --yaml 'limit: 3 repoOwner: github'

  # Call with YAML from file (like kubectl apply -f)
  mcp openDeepWiki create_issue -f issue.yaml

  # Pipe YAML to stdin
  cat issue.yaml | mcp openDeepWiki create_issue
```

## 6. Operation Details

### 6.1 0-Args Mode (list)

Uses `ListServersConfig()` which only reads config, no server connection.

### 6.2 1-Arg Mode (server-info)

Connects to specified server and fetches tool list.

### 6.3 2-Args Mode (tool-info)

Gets detailed tool info with human-readable formatting:
- `param_format`: Format description
- `param_example`: Formatted params like `key:type={value} // description`
- `call_example`: Full command example

### 6.4 3+ Args Mode (call)

Parses `key=value` or `key:type=value` arguments and calls the tool.

**Parameter Input Methods:**
1. `key=value` format (default, backward compatible)
2. `-f <file>` / `--file=<file>`: Read YAML from file
3. `--yaml <yaml>` / `-y <yaml>`: Inline YAML string
4. stdin pipe (auto-detected if no positional args and stdin has data)

## 7. Unified Output Format

### 7.1 Success Output

```json
{
  "success": true,
  "data": {...},
  "meta": {
    "timestamp": "2026-03-23T10:00:00Z",
    "version": "v0.2.8"
  }
}
```

### 7.2 Error Output

```json
{
  "success": false,
  "error": {
    "code": "MCP_SERVER_NOT_FOUND",
    "message": "Server 'xxx' not found in config",
    "details": {...}
  },
  "meta": {
    "timestamp": "2026-03-23T10:00:00Z",
    "version": "v0.2.8"
  }
}
```

## 8. Transport Types

- `"sse"` - Traditional HTTP/SSE transport
- `"streamable"` - Modern streaming HTTP transport (default)
- `"stdio"` - Local subprocess transport

## 9. Key Implementation Details

### 9.1 ParseKVArgs

Supports formats:
- `key=value` (string type)
- `key:string=value`
- `key:number=123`
- `key:bool=true`

### 9.2 YAML Parameter Input

Supports three input modes for complex parameters (objects, arrays, multi-line text):

#### 9.2.1 Inline YAML (`--yaml` or `-y`)

```bash
mcp server tool --yaml 'name: John details: {age: 30, city: NYC}'
mcp server tool -y 'tags: [dev, ops] enabled: true'
```

#### 9.2.2 File Input (`-f` or `--file`)

```bash
mcp server tool -f params.yaml
mcp server tool --file=/path/to/params.yaml
```

#### 9.2.3 Pipe Input (auto-detect stdin)

```bash
cat params.yaml | mcp server tool
echo 'name: test' | mcp server tool
```

#### 9.2.4 Priority

When multiple input methods are present:
1. `-f` / `--file` takes highest priority
2. `--yaml` / `-y` second priority
3. stdin pipe (if detected and no positional args) third priority
4. Fall back to `key=value` parsing

#### 9.2.5 YAML to JSON Conversion

YAML is automatically converted to JSON for MCP protocol:
- YAML booleans → JSON booleans
- YAML numbers → JSON numbers
- YAML strings → JSON strings
- YAML arrays → JSON arrays
- YAML objects → JSON objects

### 9.3 FormatInputSchema

Transforms JSON Schema to human-readable format:
```
["repo_id:number={value} // Repository ID", ...]
```

### 9.3 Type Hints

- `string` → `string`
- `number` → `number`
- `integer` → `int`
- `boolean` → `bool`
- `array` → `array`
- `object` → `object`
