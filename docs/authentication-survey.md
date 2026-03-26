# MCP 服务器认证方式调研

## 1. 概述

MCP (Model Context Protocol) 服务器的认证方式主要分为以下几类：

| 认证方式 | 描述 | 适用场景 |
|---------|------|---------|
| **Bearer Token (API Key)** | 通过 HTTP `Authorization` 头传递静态令牌 | 快速原型、简单认证 |
| **OAuth 2.1 + PKCE** | 完整的 OAuth 流程，支持动态客户端注册 | 企业级应用、需要第三方登录 |
| **HTTP Headers** | 自定义 HTTP 头传递认证信息 | 内部服务、定制化需求 |

---

## 2. Bearer Token / API Key 认证

### 2.1 原理

最常见的认证方式。MCP 客户端在请求中携带 `Authorization: Bearer <token>` 头：

```
GET /mcp HTTP/1.1
Host: your-server.com
Authorization: Bearer eyJhbGciOiJSUzI1NiIs...
```

### 2.2 配置文件格式

**Atlassian 官方格式** (使用 `headers` 字段):

```json
{
  "mcpServers": {
    "atlassian-rovo-mcp": {
      "url": "https://mcp.atlassian.com/v1/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_API_KEY_HERE"
      }
    }
  }
}
```

**Cloudflare 示例** (使用 env 变量):

```json
{
  "mcpServers": {
    "my-server": {
      "command": "node",
      "args": ["./server.js"],
      "env": {
        "AUTH_TOKEN": "Bearer ${AUTH_TOKEN}"
      }
    }
  }
}
```

### 2.3 各客户端支持情况

| 客户端 | headers 字段 | env 字段 | 备注 |
|-------|-------------|---------|------|
| Claude Desktop | ✅ | ✅ | headers 需要配置在 manifest.json |
| Cursor | ✅ | ✅ | 支持 `Authorization: Bearer xxx` |
| VS Code | ✅ | ✅ | MCP 扩展支持 |
| mcp2cli (本项目) | ❌ | ✅ | **待支持** |
| Python SDK | ✅ | ✅ | 通过 transport 参数传递 |
| Go SDK | ✅ (httpHeaders) | ✅ | 已支持但需暴露配置 |

---

## 3. OAuth 2.1 + PKCE 认证

### 3.1 原理

MCP 规范要求 MCP 服务器实现 OAuth 2.1 Protected Resource Metadata (RFC9728)，客户端需要支持：

- **Authorization Code + PKCE** (必须)
- **Dynamic Client Registration (DCR)** (必须)
- **Resource Indicators** (防止令牌误用)

### 3.2 流程

```
1. MCP Client → MCP Server: 发送请求
2. MCP Server ← 401 Unauthorized + authorization endpoint
3. MCP Client → Authorization Server: 使用 DCR 注册客户端
4. MCP Client → Authorization Server: Authorization Code Flow + PKCE
5. MCP Client → MCP Server: 使用 access_token 发起请求
6. MCP Server 验证 token 并返回资源
```

### 3.3 适用场景

- 需要用户登录（如 Google OAuth）
- 企业级应用，需要细粒度权限控制
- 需要 token 刷新机制

---

## 4. HTTP Headers 自定义认证

### 4.1 常见 Header 名称

| Header 名称 | 用途 |
|------------|------|
| `Authorization: Bearer <token>` | 标准 Bearer 令牌 |
| `X-API-Key: <key>` | API Key 认证 |
| `Api-Key: <key>` | 备选 API Key 格式 |
| `X-Auth-Token: <token>` | 自定义认证 Token |

### 4.2 多 Header 场景

```json
{
  "mcpServers": {
    "secure-server": {
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer secret-token-123",
        "X-Custom-Header": "custom-value"
      }
    }
  }
}
```

---

## 5. Go SDK 认证实现

### 5.1 当前代码分析

查看 `client.go:107-117`：

```go
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
```

**问题**: 第二个参数 `nil` 表示没有传递任何 HTTP headers，认证信息无法传递。

### 5.2 需要的修改

**ServerConfig 需要新增字段**:

```go
type ServerConfig struct {
    Transport string            `json:"transport,omitempty"`
    Type      string            `json:"type,omitempty"`
    URL       string            `json:"url,omitempty"`
    Command   string            `json:"command,omitempty"`
    Args      []string          `json:"args,omitempty"`
    Env       map[string]string `json:"env,omitempty"`
    Timeout   int               `json:"timeout,omitempty"`

    // 新增认证相关字段
    Headers   map[string]string `json:"headers,omitempty"`  // HTTP Headers
    Auth      AuthConfig         `json:"auth,omitempty"`     // 认证配置
}
```

### 5.3 go-sdk 接口

查看 [pkg.go.dev](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)：

```go
// SSE 传输
func NewSSEClientTransport(url string, httpHeaders map[string]string) *SSEClientTransport

// Streamable HTTP 传输
func NewStreamableClientTransport(url string, httpHeaders map[string]string) *StreamableClientTransport
```

**认证支持**:

- `httpHeaders`: 传递自定义 HTTP 头，支持 `Authorization: Bearer xxx`
- SDK 内部处理 OAuth 流程时也会使用这些 headers

---

## 6. 配置兼容性矩阵

### 6.1 主流 MCP 客户端/服务器配置格式

| 客户端 | `headers` | `auth` | `env` (认证用) | 备注 |
|-------|----------|--------|---------------|------|
| Claude Desktop | ✅ | ❌ | ✅ | 需 manifest.json |
| Cursor | ✅ | ❌ | ✅ | mcp.json |
| VS Code | ✅ | ❌ | ✅ | MCP 扩展 |
| n8n | ✅ | ❌ | ✅ | MCP Client Node |
| GitHub Copilot | ❌ | ❌ | ✅ | 仅 env |
| **本项目 mcp2cli** | **待支持** | **待设计** | ✅ | 需扩展 |

### 6.2 建议兼容的配置字段

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

---

## 7. 环境变量替换

### 7.1 常见模式

```json
{
  "headers": {
    "Authorization": "Bearer ${AUTH_TOKEN}",
    "X-API-Key": "${API_KEY}"
  },
  "env": {
    "AUTH_TOKEN": "actual-token-value",
    "API_KEY": "actual-api-key"
  }
}
```

### 7.2 注意事项

- **敏感信息不要写在配置文件中**，应使用环境变量引用
- `env` 字段既可以传递命令执行时的环境变量，也可以用于存储敏感信息
- Bearer Token 格式通常是 `Bearer <token>` 或直接是 `<token>`

---

## 8. 实现建议

### 8.1 优先级

| 优先级 | 功能 | 说明 |
|-------|------|------|
| P0 | `headers` 字段支持 | 最广泛使用，最简单 |
| P1 | Bearer Token 解析 | 识别 `Bearer xxx` 格式 |
| P2 | OAuth 2.1 支持 | 企业级需求 |
| P3 | 环境变量替换 | `${VAR}` 语法 |

### 8.2 变更点

1. **config.go**: 扩展 `ServerConfig` 结构体，添加 `Headers map[string]string`
2. **client.go**: 修改 `buildTransport()` 方法，将 headers 传递给 SDK
3. **测试**: 添加认证场景的集成测试

### 8.3 安全建议

- 不在日志中打印 Authorization 头
- 支持从环境变量读取敏感信息
- 配置文件中使用 `${VAR}` 引用环境变量

---

## 9. 参考资料

- [MCP Authorization Specification](https://modelcontextprotocol.io/specification/draft/basic/authorization)
- [MCP Security Tutorial](https://modelcontextprotocol.io/docs/tutorials/security/authorization)
- [Atlassian MCP Authentication](https://support.atlassian.com/atlassian-rovo-mcp-server/docs/configuring-authentication-via-api-token/)
- [Cloudflare MCP Bearer Auth Example](https://github.com/jmorrell-cloudflare/mcp-bearer-auth-example)
- [TrueFoundry MCP Authentication Guide](https://www.truefoundry.com/blog/mcp-authentication-in-claude-code)
- [go-sdk GitHub](https://github.com/modelcontextprotocol/go-sdk)

---

## 10. 结论

当前 `mcp2cli` 项目需要添加认证支持，建议按以下顺序实现：

1. **首先支持 `headers` 字段**，这是最广泛使用的认证配置方式
2. **支持环境变量替换** (`${VAR}` 语法)，确保敏感信息安全
3. **后续考虑 OAuth 2.1 支持**，满足企业级需求

主要变更点在 `ServerConfig` 结构体和 `buildTransport()` 方法，SDK 层面已支持通过 `httpHeaders` 参数传递认证信息。
