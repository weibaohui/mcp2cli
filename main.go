package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcpSDK "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/weibaohui/mcp2cli/internal/mcp"
	"gopkg.in/yaml.v3"
)

const version = "v0.3.0"

var streamOutput bool
var yamlParams string
var yamlFile string
var outputFormat string

const outputFormatJSON = "json"
const outputFormatYAML = "yaml"
const outputFormatText = "text"

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
  mcp openDeepWiki list_repositories limit=3

  # Call a tool with streaming output
  mcp --stream openDeepWiki list_repositories limit=3

  # Call with YAML parameters (inline)
  mcp openDeepWiki list_repositories --yaml 'limit: 3 repoOwner: github'

  # Call with YAML from file (like kubectl apply -f)
  mcp openDeepWiki create_issue -f issue.yaml

  # Pipe YAML to stdin
  cat issue.yaml | mcp openDeepWiki create_issue

  # Output in different formats (default: json)
  mcp --output yaml openDeepWiki list_repositories
  mcp --output text openDeepWiki list_repositories`,
	Args: cobra.ArbitraryArgs,
	Run:  runMCP,
}

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Start interactive REPL mode",
	Long: `Start an interactive REPL for calling MCP tools.

Commands:
  server tool [args...]  Call a tool
  servers               List available servers
  use <server>          Set default server
  tool [name]           List tools or show tool details
  help                  Show this help
  exit                  Exit interactive mode

Examples:
  mcp> openDeepWiki list_repositories limit=3
  mcp> use openDeepWiki
  mcp> list_repositories limit=5`,
	RunE: runInteractive,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&streamOutput, "stream", "s", false, "Enable streaming output for text results")
	rootCmd.PersistentFlags().StringVarP(&yamlParams, "yaml", "y", "", "YAML parameters (inline)")
	rootCmd.PersistentFlags().StringVarP(&yamlFile, "file", "f", "", "YAML file with parameters (like kubectl apply -f)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json|yaml|text)")
	rootCmd.AddCommand(interactiveCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runMCP(cmd *cobra.Command, args []string) {
	// Validate output format
	switch outputFormat {
	case outputFormatJSON, outputFormatYAML, outputFormatText:
		// valid
	default:
		fmt.Fprintf(os.Stderr, "invalid output format %q: must be json, yaml, or text\n", outputFormat)
		os.Exit(1)
	}

	ctx := context.Background()

	// Load config
	config, loadedPaths, err := mcp.LoadConfig()
	if err != nil {
		printMCPError(err)
		os.Exit(1)
	}

	dispatcher := mcp.NewDispatcher(config, loadedPaths)

	// Check if YAML input mode is active
	hasYAMLInput := yamlParams != "" || yamlFile != "" || (mcp.IsPipedInput() && len(args) <= 2)

	// Route based on argument count and YAML input mode
	switch {
	case len(args) == 0:
		runMCPList(ctx, dispatcher)
	case len(args) == 1:
		runMCPServerInfo(ctx, dispatcher, args[0])
	case len(args) == 2 && !hasYAMLInput:
		// Only show tool info if no YAML input (user might want to see params)
		runMCPInfo(ctx, dispatcher, args[0], args[1])
	default:
		// 3+ args, or 2 args with YAML input -> call mode
		runMCPCall(ctx, dispatcher, args[0], args[1], args[2:])
	}
}

func runInteractive(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load config
	config, loadedPaths, err := mcp.LoadConfig()
	if err != nil {
		return err
	}

	dispatcher := mcp.NewDispatcher(config, loadedPaths)
	defer dispatcher.Close()
	currentServer := ""

	// Set up signal handling for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("MCP Interactive Mode")
	fmt.Println("Type 'help' for available commands, 'exit' to quit")
	fmt.Println()

	for {
		fmt.Print("mcp> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Handle signals
		select {
		case sig := <-sigChan:
			fmt.Printf("\nReceived signal %v, exiting...\n", sig)
			return nil
		default:
		}

		parts := parseInput(input)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "exit", "quit", "q":
			fmt.Println("Goodbye!")
			return nil
		case "help", "?":
			printInteractiveHelp()
		case "servers", "list":
			runMCPList(ctx, dispatcher)
		case "use":
			if len(parts) < 2 {
				fmt.Println("Usage: use <server>")
				continue
			}
			currentServer = parts[1]
			fmt.Printf("Default server set to: %s\n", currentServer)
		case "tool", "tools":
			if len(parts) < 2 {
				if currentServer == "" {
					fmt.Println("No default server. Use 'use <server>' or 'servers' to list available servers")
					continue
				}
				if err := runMCPServerInfo(ctx, dispatcher, currentServer); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			} else {
				if currentServer == "" {
					fmt.Println("No default server. Use 'use <server>' first")
					continue
				}
				if err := runMCPInfo(ctx, dispatcher, currentServer, parts[1]); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		default:
			// Tool call: check if parts[0] is a server, otherwise use currentServer
			var server, tool string
			var toolArgs []string

			if len(parts) >= 2 {
				if _, ok := config.MCPServers[parts[0]]; ok {
					// parts[0] is a server
					server = parts[0]
					tool = parts[1]
					toolArgs = parts[2:]
				} else if currentServer != "" {
					// Use currentServer, parts[0] is the tool
					server = currentServer
					tool = parts[0]
					toolArgs = parts[1:]
				} else {
					fmt.Println("No default server. Use 'use <server>' or specify server: <server> <tool> [args...]")
					continue
				}
			} else if len(parts) == 1 && currentServer != "" {
				server = currentServer
				tool = parts[0]
			} else {
				fmt.Println("Invalid command. Type 'help' for available commands")
				continue
			}

			if err := runMCPCall(ctx, dispatcher, server, tool, toolArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}

	return nil
}

func parseInput(input string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for _, ch := range input {
		switch ch {
		case '"':
			inQuote = !inQuote
		case ' ':
			if !inQuote {
				if current.Len() > 0 {
					parts = append(parts, current.String())
					current.Reset()
				}
				continue
			}
		}
		current.WriteRune(ch)
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func printInteractiveHelp() {
	fmt.Println(`Available commands:
  server tool [args...]  Call a tool on a specific server
  use <server>          Set default server for short commands
  servers               List all configured servers
  tool [name]           List tools on default server or show tool details
  help                  Show this help
  exit                  Exit interactive mode

Shortcut: If no server prefix, uses the default server (set with 'use')

Examples:
  openDeepWiki list_repositories limit=3
  use openDeepWiki
  list_repositories limit=5`)
}

func runMCPList(ctx context.Context, d *mcp.Dispatcher) {
	servers := d.ListServersConfig()
	printMCPSuccess(map[string]any{
		"configFiles": d.ConfigPaths(),
		"servers":     servers,
	})
}

func runMCPServerInfo(ctx context.Context, d *mcp.Dispatcher, serverName string) error {
	info, err := d.GetServerInfo(ctx, serverName)
	if err != nil {
		return err
	}
	printMCPSuccess(map[string]any{
		"configFiles": d.ConfigPaths(),
		"server":      info,
	})
	return nil
}

func runMCPInfo(ctx context.Context, d *mcp.Dispatcher, serverName, toolName string) error {
	match, err := d.GetToolInfo(ctx, serverName, toolName)
	if err != nil {
		return err
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
	return nil
}

func runMCPCall(ctx context.Context, d *mcp.Dispatcher, serverName, toolName string, kvArgs []string) error {
	var params map[string]any
	var err error

	// Priority: -f > --yaml > stdin pipe > key=value

	// 1. File input (-f)
	if yamlFile != "" {
		params, err = mcp.ReadYAMLFile(yamlFile)
		if err != nil {
			return err
		}
	} else if yamlParams != "" {
		// 2. Inline YAML (--yaml / -y)
		params, err = mcp.ParseYAML(yamlParams)
		if err != nil {
			return err
		}
	} else if mcp.IsPipedInput() && len(kvArgs) == 0 {
		// 3. Stdin pipe (only if no positional args)
		params, err = mcp.ReadStdinYAML()
		if err != nil {
			return err
		}
	} else {
		// 4. Fall back to key=value parsing
		params, err = mcp.ParseKVArgs(kvArgs)
		if err != nil {
			return err
		}
	}

	// Call tool
	actualServer, callResult, err := d.CallTool(ctx, toolName, serverName, params)
	if err != nil {
		return err
	}

	// Format result - callResult is *mcp.CallToolResult
	output := formatCallToolResult(callResult)

	if streamOutput {
		if cr, ok := callResult.(*mcpSDK.CallToolResult); ok {
			if streamed, err := streamTextContent(cr); err != nil {
				return err
			} else if streamed {
				return nil
			}
		}
	}

	printMCPSuccess(map[string]any{
		"server": actualServer,
		"method": toolName,
		"result": output,
	})
	return nil
}

// streamTextContent outputs text content progressively for streaming mode
// Returns (streamed, error)
func streamTextContent(result *mcpSDK.CallToolResult) (bool, error) {
	if result == nil || len(result.Content) == 0 {
		return false, nil
	}

	writer := bufio.NewWriter(os.Stdout)
	streamed := false
	for _, item := range result.Content {
		if tc, ok := item.(*mcpSDK.TextContent); ok && tc.Text != "" {
			if _, err := writer.WriteString(tc.Text); err != nil {
				return streamed, err
			}
			if err := writer.WriteByte('\n'); err != nil {
				return streamed, err
			}
			if err := writer.Flush(); err != nil {
				return streamed, err
			}
			streamed = true
		}
	}
	return streamed, nil
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
	printOutput(output)
}

func printMCPError(err error) {
	var mcpErr *mcp.MCPError
	var output map[string]any

	if errors.As(err, &mcpErr) {
		errObj := map[string]any{
			"code":    mcpErr.Code,
			"message": mcpErr.Message,
		}
		if len(mcpErr.Details) > 0 {
			errObj["details"] = mcpErr.Details
		}
		output = map[string]any{
			"success": false,
			"error":   errObj,
			"meta": map[string]any{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"version":   version,
			},
		}
	} else {
		// Generic error
		output = map[string]any{
			"success": false,
			"error": map[string]any{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
			"meta": map[string]any{
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"version":   version,
			},
		}
	}
	printOutput(output)
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

// PrintYAML prints YAML to stdout
func PrintYAML(v any) {
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode YAML: %v\n", err)
		os.Exit(1)
	}
	enc.Close()
}

// PrintText prints text content to stdout (compact JSON)
func PrintText(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// printOutput prints output in the specified format
func printOutput(v any) {
	switch outputFormat {
	case outputFormatYAML:
		PrintYAML(v)
	case outputFormatText:
		PrintText(v)
	default:
		PrintJSON(v)
	}
}
