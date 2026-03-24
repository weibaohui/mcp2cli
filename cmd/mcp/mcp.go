package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/weibaohui/mcp2cli/internal/mcp"
	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

const version = "v0.2.8"

var rootCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Interact with MCP (Model Context Protocol) Servers",
	Long: `Interact with MCP (Model Context Protocol) Servers

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
  mcp openDeepWiki list_repositories limit=3`,
	Args: cobra.ArbitraryArgs,
	Run:  runMCP,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMCP(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Load config
	config, loadedPaths, err := mcp.LoadConfig()
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}

	dispatcher := mcp.NewDispatcher(config, loadedPaths)

	// Route based on argument count
	switch len(args) {
	case 0:
		runMCPList(ctx, dispatcher)
	case 1:
		runMCPServerInfo(ctx, dispatcher, args[0])
	case 2:
		runMCPInfo(ctx, dispatcher, args[0], args[1])
	default:
		runMCPCall(ctx, dispatcher, args[0], args[1], args[2:])
	}
}

func runMCPList(ctx context.Context, d *mcp.Dispatcher) {
	servers := d.ListServersConfig()
	printMCPSuccess(map[string]any{
		"configFiles": d.ConfigPaths(),
		"servers":     servers,
	})
}

func runMCPServerInfo(ctx context.Context, d *mcp.Dispatcher, serverName string) {
	info, err := d.GetServerInfo(ctx, serverName)
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}
	printMCPSuccess(map[string]any{
		"configFiles": d.ConfigPaths(),
		"server":      info,
	})
}

func runMCPInfo(ctx context.Context, d *mcp.Dispatcher, serverName, toolName string) {
	match, err := d.GetToolInfo(ctx, serverName, toolName)
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}

	// Format tool information
	formattedParams := mcp.FormatInputSchema(match.Tool.InputSchema)
	requiredParams := mcp.GetRequiredParams(match.Tool.InputSchema)
	paramInfoList := mcp.GetParamInfoList(match.Tool.InputSchema)

	toolData := map[string]any{
		"name":        match.Tool.Name,
		"description": match.Tool.Description,
	}

	// Add required field if there are required params
	if len(requiredParams) > 0 {
		toolData["required"] = strings.Join(requiredParams, " OR ")
	}

	if formattedParams != nil {
		toolData["param_format"] = "key:type=value (type: string/number/bool)"
		toolData["param_example"] = formattedParams
		toolData["call_example"] = fmt.Sprintf("mcp %s %s %s",
			match.ServerName,
			match.Tool.Name,
			buildCallExample(paramInfoList))
	} else {
		toolData["inputSchema"] = match.Tool.InputSchema
	}

	printMCPSuccess(map[string]any{
		"server": match.ServerName,
		"tool":   toolData,
	})
}

func runMCPCall(ctx context.Context, d *mcp.Dispatcher, serverName, toolName string, kvArgs []string) {
	// Parse key=value arguments
	params, err := mcp.ParseKVArgs(kvArgs)
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}

	// Call tool
	actualServer, callResult, err := d.CallTool(ctx, toolName, serverName, params)
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}

	// Format result - callResult is *mcp.CallToolResult
	output := formatCallToolResult(callResult)

	printMCPSuccess(map[string]any{
		"server": actualServer,
		"method": toolName,
		"result": output,
	})
}

// formatCallToolResult formats the CallToolResult for JSON output
func formatCallToolResult(result any) any {
	if result == nil {
		return nil
	}

	// Try to type assert to see if we can extract content
	callResult, ok := result.(*mcpSDK.CallToolResult)
	if !ok {
		return result
	}

	if callResult.Content == nil || len(callResult.Content) == 0 {
		return callResult
	}

	// Extract text content
	var contentData []map[string]any
	for _, item := range callResult.Content {
		if tc, ok := item.(*mcpSDK.TextContent); ok && tc.Text != "" {
			contentData = append(contentData, map[string]any{
				"type": "text",
				"text": tc.Text,
			})
		}
	}

	if len(contentData) > 0 {
		return contentData
	}

	return callResult
}

func buildCallExample(params []mcp.ParamInfo) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s:%s={value}", p.Name, p.Type))
	}
	return strings.Join(parts, " ")
}

func printMCPSuccess(data any) {
	output := map[string]any{
		"success": true,
		"data":    data,
		"meta": map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
		},
	}
	PrintJSON(output)
}

func printMCPError(err error) {
	var mcpErr *mcp.MCPError
	if errors.As(err, &mcpErr) {
		errObj := map[string]any{
			"code":    mcpErr.Code,
			"message": mcpErr.Message,
		}
		if len(mcpErr.Details) > 0 {
			errObj["details"] = mcpErr.Details
		}
		PrintJSON(map[string]any{
			"success": false,
			"error":   errObj,
			"meta": map[string]any{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"version":   version,
			},
		})
		return
	}

	// Generic error
	PrintJSON(map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    "INTERNAL_ERROR",
			"message": err.Error(),
		},
		"meta": map[string]any{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   version,
		},
	})
}

// PrintJSON prints JSON to stdout
func PrintJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}
