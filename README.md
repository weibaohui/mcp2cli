# mcp2cli

> Turn MCP servers into CLI commands — one bash call replaces dozens of tool rounds.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![npm](https://img.shields.io/npm/v/@weibaohui/mcp2cli?style=flat-square)](https://www.npmjs.com/package/@weibaohui/mcp2cli)

## Why mcp2cli?

When an LLM uses MCP tools directly, every call carries heavy overhead:
- Tool discovery: list tools → read schemas → understand parameters → construct call → parse response
- Each MCP tool definition (schema, descriptions, types) stays in context, consuming tokens for the **entire conversation**
- A single "search GitHub and summarize" task can burn thousands of tokens just on protocol overhead

**mcp2cli compresses that into one bash command:**

```
# Direct MCP: 3 rounds, ~2000+ tokens of context
1. list_tools(server)          → get all tool schemas
2. get_tool_details(server, tool) → read parameter definitions
3. call_tool(server, tool, args)  → get result

# mcp2cli: 1 bash call, ~200 tokens
mcp openDeepWiki get_repo_structure repoOwner=github repoName=vscode
```

**Result: 80-90% fewer tokens per tool interaction.**

## Quick Start

```bash
# Install
npm install -g @weibaohui/mcp2cli

# List servers
mcp

# Explore tools on a server
mcp openDeepWiki

# View tool details + param examples
mcp openDeepWiki get_repo_structure

# Call a tool
mcp openDeepWiki get_repo_structure repoOwner=github repoName=vscode
```

## How It Saves Tokens

### Before: Direct MCP (verbose)

Every MCP interaction requires the LLM to maintain tool schemas in context:

```json
// Tool schema alone = ~500 tokens, stays in EVERY message
{
  "name": "get_repo_structure",
  "description": "Retrieves the complete file and directory structure of a Git repository...",
  "inputSchema": {
    "type": "object",
    "properties": {
      "repoOwner": {"type": "string", "description": "..."},
      "repoName": {"type": "string", "description": "..."},
      ...
    },
    "required": ["repoOwner", "repoName"]
  }
}
```

Multiply this by **every tool on every server** — a server with 20 tools = ~10,000 tokens permanently in context.

### After: mcp2cli (lean)

The LLM only needs a short bash command. No schemas, no tool definitions, no protocol overhead:

```bash
# Token cost: just the command string (~30 tokens)
mcp openDeepWiki get_repo_structure repoOwner=github repoName=vscode
```

```json
// Output is concise JSON (~100 tokens)
{
  "success": true,
  "data": { "server": "openDeepWiki", "method": "get_repo_structure", "result": "..." },
  "meta": { "timestamp": "2026-03-28T10:00:00Z", "version": "v0.3.0" }
}
```

### Token Comparison

| Scenario | Direct MCP | mcp2cli | Saving |
|----------|-----------|---------|--------|
| Discover 1 tool | ~500 tokens (schema in context) | ~100 tokens (one bash call) | 80% |
| Call 1 tool | ~300 tokens (schema + call overhead) | ~130 tokens (command + output) | 57% |
| 10-tool server in context | ~10,000 tokens (persistent) | 0 tokens (loaded on demand) | 100% |
| Full workflow (discover + call) | ~2,000 tokens | ~230 tokens | 89% |

## Usage in Claude Code / Cursor / Windsurf

Add to your project's MCP config (e.g. `.mcp/config.json` or `~/.config/mcp/config.json`):

```json
{
  "mcpServers": {
    "openDeepWiki": {
      "url": "https://opendeepwiki.k8m.site/mcp/streamable"
    }
  }
}
```

Then in your AI tool, just run bash commands:

```
# The LLM can explore and call tools in one step
$ mcp openDeepWiki list_repositories limit=3

# No need to load tool schemas — just call
$ mcp openDeepWiki get_repo_structure repoOwner=weibaohui repoName=mcp2cli
```

## Argument Format

```bash
# Simple key=value (string by default)
mcp server tool name=John age=30

# Typed key:type=value (for precision)
mcp server tool name:string=John age:number=30 enabled:bool=true
```

**Supported types:** `string`, `number`, `int`, `float`, `bool`

## Installation

### npm (recommended)

```bash
npm install -g @weibaohui/mcp2cli
```

Supports Linux, macOS, Windows on amd64/arm64. `mcp` command ready immediately.

### Go install

```bash
go install github.com/weibaohui/mcp2cli@latest
mv $(go env GOPATH)/bin/mcp2cli $(go env GOPATH)/bin/mcp
```

### Binary download

Download from [GitHub Releases](https://github.com/weibaohui/mcp2cli/releases/latest):

```bash
# macOS / Linux
mv mcp2cli-darwin-arm64 mcp && chmod +x mcp && sudo mv mcp /usr/local/bin/

# Windows
ren mcp2cli-windows-amd64.exe mcp.exe
```

## Command Reference

| Command | Description |
|---------|-------------|
| `mcp` | List configured servers |
| `mcp <server>` | List tools on a server |
| `mcp <server> <tool>` | Show tool details + param examples |
| `mcp <server> <tool> key=value ...` | Call a tool |

## License

MIT License - see [LICENSE](LICENSE) for details.
