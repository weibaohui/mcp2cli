package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchPaths_notEmpty(t *testing.T) {
	paths := GetConfigSearchPaths()
	if len(paths) == 0 {
		t.Error("expected non-empty search paths")
	}
}

func TestLoadConfig_noneExist(t *testing.T) {
	// Use non-existent paths
	paths := []string{
		"/nonexistent/path/config.json",
		"/another/nonexistent/config.json",
	}
	_, loadedPaths, err := LoadConfigFromPaths(paths)

	if err == nil {
		t.Error("expected error when no config files exist")
	}

	if len(loadedPaths) != 0 {
		t.Errorf("expected no loaded paths, got %d", len(loadedPaths))
	}
}

func TestLoadConfig_mergesAllFiles(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create first config file
	config1 := `{
		"mcpServers": {
			"server1": {
				"url": "http://example1.com"
			}
		}
	}`
	config1Path := filepath.Join(tmpDir, "config1.json")
	if err := os.WriteFile(config1Path, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}

	// Create second config file
	config2 := `{
		"mcpServers": {
			"server2": {
				"url": "http://example2.com"
			}
		}
	}`
	config2Path := filepath.Join(tmpDir, "config2.json")
	if err := os.WriteFile(config2Path, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	// Load both configs
	cfg, loadedPaths, err := LoadConfigFromPaths([]string{config1Path, config2Path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loadedPaths) != 2 {
		t.Errorf("expected 2 loaded paths, got %d", len(loadedPaths))
	}

	if len(cfg.MCPServers) != 2 {
		t.Errorf("expected 2 servers, got %d", len(cfg.MCPServers))
	}

	if _, ok := cfg.MCPServers["server1"]; !ok {
		t.Error("expected server1 to be present")
	}

	if _, ok := cfg.MCPServers["server2"]; !ok {
		t.Error("expected server2 to be present")
	}
}

func TestLoadConfig_highPriorityWinsOnConflict(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create first (low priority) config file
	config1 := `{
		"mcpServers": {
			"server1": {
				"url": "http://low-priority.com",
				"command": "low-priority-cmd"
			}
		}
	}`
	config1Path := filepath.Join(tmpDir, "config1.json")
	if err := os.WriteFile(config1Path, []byte(config1), 0644); err != nil {
		t.Fatalf("failed to write config1: %v", err)
	}

	// Create second (high priority) config file - higher priority path is second
	config2 := `{
		"mcpServers": {
			"server1": {
				"url": "http://high-priority.com",
				"command": "high-priority-cmd"
			}
		}
	}`
	config2Path := filepath.Join(tmpDir, "config2.json")
	if err := os.WriteFile(config2Path, []byte(config2), 0644); err != nil {
		t.Fatalf("failed to write config2: %v", err)
	}

	// Load configs - config2 should have higher priority (it's later in the list)
	cfg, _, err := LoadConfigFromPaths([]string{config1Path, config2Path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Higher priority should win
	if cfg.MCPServers["server1"].URL != "http://high-priority.com" {
		t.Errorf("expected high-priority URL, got %s", cfg.MCPServers["server1"].URL)
	}
}

func TestInferTransportType_explicitTransport(t *testing.T) {
	cfg := ServerConfig{Transport: "sse"}
	if got := InferTransportType(cfg); got != "sse" {
		t.Errorf("expected 'sse', got %q", got)
	}

	cfg = ServerConfig{Transport: "streamable"}
	if got := InferTransportType(cfg); got != "streamable" {
		t.Errorf("expected 'streamable', got %q", got)
	}

	cfg = ServerConfig{Transport: "stdio"}
	if got := InferTransportType(cfg); got != "stdio" {
		t.Errorf("expected 'stdio', got %q", got)
	}
}

func TestInferTransportType_typeField(t *testing.T) {
	cfg := ServerConfig{Type: "streamable-http"}
	if got := InferTransportType(cfg); got != "streamable" {
		t.Errorf("expected 'streamable', got %q", got)
	}

	cfg = ServerConfig{Type: "sse"}
	if got := InferTransportType(cfg); got != "sse" {
		t.Errorf("expected 'sse', got %q", got)
	}

	cfg = ServerConfig{Type: "stdio"}
	if got := InferTransportType(cfg); got != "stdio" {
		t.Errorf("expected 'stdio', got %q", got)
	}
}

func TestInferTransportType_commandImpliesStdio(t *testing.T) {
	cfg := ServerConfig{Command: "npx"}
	if got := InferTransportType(cfg); got != "stdio" {
		t.Errorf("expected 'stdio', got %q", got)
	}
}

func TestInferTransportType_urlWithSse(t *testing.T) {
	cfg := ServerConfig{URL: "https://example.com/sse/endpoint"}
	if got := InferTransportType(cfg); got != "sse" {
		t.Errorf("expected 'sse', got %q", got)
	}
}

func TestInferTransportType_urlWithStream(t *testing.T) {
	cfg := ServerConfig{URL: "https://example.com/stream/endpoint"}
	if got := InferTransportType(cfg); got != "streamable" {
		t.Errorf("expected 'streamable', got %q", got)
	}
}

func TestInferTransportType_defaultStreamable(t *testing.T) {
	cfg := ServerConfig{URL: "https://example.com/other/endpoint"}
	if got := InferTransportType(cfg); got != "streamable" {
		t.Errorf("expected 'streamable', got %q", got)
	}
}

func TestParseKVArgs_valid(t *testing.T) {
	args := []string{"name=John", "age=30"}
	result, err := ParseKVArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("expected 'John', got %v", result["name"])
	}

	// Without type annotation, age=30 is treated as string "30"
	if result["age"] != "30" {
		t.Errorf("expected '30', got %v", result["age"])
	}
}

func TestParseKVArgs_typeNumber(t *testing.T) {
	args := []string{"count:number=42", "price:number=19.99"}
	result, err := ParseKVArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["count"] != float64(42) {
		t.Errorf("expected 42, got %v", result["count"])
	}

	if result["price"] != 19.99 {
		t.Errorf("expected 19.99, got %v", result["price"])
	}
}

func TestParseKVArgs_typeInt(t *testing.T) {
	args := []string{"count:int=42"}
	result, err := ParseKVArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["count"] != float64(42) {
		t.Errorf("expected 42, got %v", result["count"])
	}
}

func TestParseKVArgs_typeBool(t *testing.T) {
	args := []string{"enabled:bool=true", "disabled:bool=false"}
	result, err := ParseKVArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["enabled"] != true {
		t.Errorf("expected true, got %v", result["enabled"])
	}

	if result["disabled"] != false {
		t.Errorf("expected false, got %v", result["disabled"])
	}
}

func TestParseKVArgs_invalidFormat(t *testing.T) {
	args := []string{"invalid-no-equals"}
	_, err := ParseKVArgs(args)
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestParseKVArgs_invalidType(t *testing.T) {
	args := []string{"key:invalidtype=value"}
	_, err := ParseKVArgs(args)
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestParseKVArgs_emptyKey(t *testing.T) {
	args := []string{"=value"}
	_, err := ParseKVArgs(args)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParseKVArgs_emptyValue(t *testing.T) {
	args := []string{"key="}
	result, err := ParseKVArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "" {
		t.Errorf("expected empty string, got %v", result["key"])
	}
}

func TestFormatInputSchema_basic(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The person's name",
			},
			"age": map[string]any{
				"type": "number",
			},
		},
	}

	result := FormatInputSchema(schema)
	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	// Check that result contains expected format
	foundName := false
	foundAge := false
	for _, line := range result {
		if line == "name:string={value} // The person's name" {
			foundName = true
		}
		if line == "age:number={value}" {
			foundAge = true
		}
	}

	if !foundName {
		t.Error("expected to find formatted name line")
	}
	if !foundAge {
		t.Error("expected to find formatted age line")
	}
}

func TestGetRequiredParams_basic(t *testing.T) {
	schema := map[string]any{
		"required": []any{"name", "age"},
	}

	result := GetRequiredParams(schema)
	if len(result) != 2 {
		t.Fatalf("expected 2 required params, got %d", len(result))
	}

	expected := map[string]bool{"name": true, "age": true}
	for _, r := range result {
		if !expected[r] {
			t.Errorf("unexpected required param: %s", r)
		}
	}
}

func TestGetParamInfoList_basic(t *testing.T) {
	schema := map[string]any{
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The person's name",
			},
			"age": map[string]any{
				"type": "number",
			},
		},
		"required": []any{"name"},
	}

	result := GetParamInfoList(schema)
	if len(result) != 2 {
		t.Fatalf("expected 2 params, got %d", len(result))
	}

	// Find name param
	var nameParam ParamInfo
	for _, p := range result {
		if p.Name == "name" {
			nameParam = p
			break
		}
	}

	if nameParam.Type != "string" {
		t.Errorf("expected type 'string', got %q", nameParam.Type)
	}
	if !nameParam.Required {
		t.Error("expected name to be required")
	}
	if nameParam.Description != "The person's name" {
		t.Errorf("expected description, got %q", nameParam.Description)
	}
}

func TestConfigPaths_unix(t *testing.T) {
	paths := GetConfigSearchPaths()
	if len(paths) < 4 {
		t.Errorf("expected at least 4 paths on unix, got %d", len(paths))
	}

	// Check that first path contains .config
	if len(paths) > 0 && !contains(paths[0], ".config") {
		t.Error("expected first path to contain .config")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
