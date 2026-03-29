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

// Authentication tests

func TestLoadConfig_withHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	config := `{
		"mcpServers": {
			"server-with-auth": {
				"url": "https://example.com/mcp",
				"headers": {
					"Authorization": "Bearer secret-token",
					"X-API-Key": "api-key-123"
				}
			}
		}
	}`
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, _, err := LoadConfigFromPaths([]string{configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server, ok := cfg.MCPServers["server-with-auth"]
	if !ok {
		t.Fatal("expected server-with-auth to be present")
	}

	if server.Headers == nil {
		t.Fatal("expected headers to be present")
	}

	if server.Headers["Authorization"] != "Bearer secret-token" {
		t.Errorf("expected 'Bearer secret-token', got %q", server.Headers["Authorization"])
	}

	if server.Headers["X-API-Key"] != "api-key-123" {
		t.Errorf("expected 'api-key-123', got %q", server.Headers["X-API-Key"])
	}
}

func TestLoadConfig_withOAuthConfig(t *testing.T) {
	tmpDir := t.TempDir()
	config := `{
		"mcpServers": {
			"server-with-oauth": {
				"url": "https://example.com/mcp",
				"auth": {
					"oauth": {
						"clientId": "my-client-id",
						"clientSecret": "my-client-secret",
						"authorizationURL": "https://auth.example.com/authorize",
						"tokenURL": "https://auth.example.com/token",
						"scopes": "openid profile"
					}
				}
			}
		}
	}`
	configPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, _, err := LoadConfigFromPaths([]string{configPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	server, ok := cfg.MCPServers["server-with-oauth"]
	if !ok {
		t.Fatal("expected server-with-oauth to be present")
	}

	if server.Auth.OAuth == nil {
		t.Fatal("expected oauth config to be present")
	}

	if server.Auth.OAuth.ClientID != "my-client-id" {
		t.Errorf("expected 'my-client-id', got %q", server.Auth.OAuth.ClientID)
	}

	if server.Auth.OAuth.AuthorizationURL != "https://auth.example.com/authorize" {
		t.Errorf("unexpected authorization URL: %s", server.Auth.OAuth.AuthorizationURL)
	}
}

func TestResolveHeaders_envVarSubstitution(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_API_TOKEN", "my-secret-token")
	defer os.Unsetenv("TEST_API_TOKEN")

	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
		Headers: map[string]string{
			"Authorization":   "Bearer ${TEST_API_TOKEN}",
			"X-Custom-Header": "static-value",
		},
	})

	headers := client.resolveHeaders()

	if headers["Authorization"] != "Bearer my-secret-token" {
		t.Errorf("expected 'Bearer my-secret-token', got %q", headers["Authorization"])
	}

	if headers["X-Custom-Header"] != "static-value" {
		t.Errorf("expected 'static-value', got %q", headers["X-Custom-Header"])
	}
}

func TestResolveHeaders_dollarVarSyntax(t *testing.T) {
	os.Setenv("MY_TOKEN", "token-value")
	defer os.Unsetenv("MY_TOKEN")

	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer $MY_TOKEN",
		},
	})

	headers := client.resolveHeaders()

	if headers["Authorization"] != "Bearer token-value" {
		t.Errorf("expected 'Bearer token-value', got %q", headers["Authorization"])
	}
}

func TestResolveHeaders_unsetVar(t *testing.T) {
	// Ensure the env var is not set
	os.Unsetenv("NONEXISTENT_VAR")

	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
		Headers: map[string]string{
			"Authorization": "Bearer ${NONEXISTENT_VAR}",
		},
	})

	headers := client.resolveHeaders()

	// Should keep the original placeholder when env var is not set
	if headers["Authorization"] != "Bearer ${NONEXISTENT_VAR}" {
		t.Errorf("expected unchanged placeholder, got %q", headers["Authorization"])
	}
}

func TestResolveHeaders_nilHeaders(t *testing.T) {
	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
	})

	headers := client.resolveHeaders()

	if headers != nil {
		t.Errorf("expected nil headers, got %v", headers)
	}
}

func TestResolveOAuthToken_withToken(t *testing.T) {
	os.Setenv("OAUTH_TOKEN", "my-oauth-token")
	defer os.Unsetenv("OAUTH_TOKEN")

	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
		Auth: AuthConfig{
			OAuth: &OAuthConfig{
				AccessToken: "${OAUTH_TOKEN}",
			},
		},
	})

	token, err := configureOAuth(client.config.Auth.OAuth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "my-oauth-token" {
		t.Errorf("expected 'my-oauth-token', got %q", token)
	}
}

func TestConfigureOAuth_withoutAuth(t *testing.T) {
	client := NewClient("test", ServerConfig{
		URL: "https://example.com/mcp",
	})

	token, err := configureOAuth(client.config.Auth.OAuth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

func TestConfigureOAuth_withoutOAuth(t *testing.T) {
	client := NewClient("test", ServerConfig{
		URL:  "https://example.com/mcp",
		Auth: AuthConfig{},
	})

	token, err := configureOAuth(client.config.Auth.OAuth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if token != "" {
		t.Errorf("expected empty token, got %q", token)
	}
}

// YAML parsing tests

func TestParseYAML_simple(t *testing.T) {
	yamlStr := `name: John
age: 30`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("expected 'John', got %v", result["name"])
	}

	// YAML may return int or float64 depending on library version
	age := result["age"]
	ageNum := 0.0
	switch v := age.(type) {
	case int:
		ageNum = float64(v)
	case float64:
		ageNum = v
	default:
		t.Fatalf("expected age to be numeric, got %T", age)
	}
	if ageNum != 30 {
		t.Errorf("expected 30, got %v", age)
	}
}

func TestParseYAML_nestedObject(t *testing.T) {
	yamlStr := `name: John
details:
  age: 30
  city: NYC`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("expected 'John', got %v", result["name"])
	}

	details, ok := result["details"].(map[string]any)
	if !ok {
		t.Fatalf("expected details to be map, got %T", result["details"])
	}

	// YAML may return int or float64
	age := details["age"]
	ageNum := 0.0
	switch v := age.(type) {
	case int:
		ageNum = float64(v)
	case float64:
		ageNum = v
	default:
		t.Fatalf("expected age to be numeric, got %T", age)
	}
	if ageNum != 30 {
		t.Errorf("expected age 30, got %v", age)
	}

	if details["city"] != "NYC" {
		t.Errorf("expected 'NYC', got %v", details["city"])
	}
}

func TestParseYAML_array(t *testing.T) {
	yamlStr := `tags:
  - dev
  - ops
  - ci`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags to be array, got %T", result["tags"])
	}

	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}

	if tags[0] != "dev" || tags[1] != "ops" || tags[2] != "ci" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

func TestParseYAML_bool(t *testing.T) {
	yamlStr := `enabled: true
disabled: false`
	result, err := ParseYAML(yamlStr)
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

func TestParseYAML_float(t *testing.T) {
	yamlStr := `price: 19.99
ratio: 0.5`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["price"] != 19.99 {
		t.Errorf("expected 19.99, got %v", result["price"])
	}

	if result["ratio"] != 0.5 {
		t.Errorf("expected 0.5, got %v", result["ratio"])
	}
}

func TestParseYAML_invalidYAML(t *testing.T) {
	yamlStr := `name: [invalid yaml`
	_, err := ParseYAML(yamlStr)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseYAML_empty(t *testing.T) {
	yamlStr := ``
	_, err := ParseYAML(yamlStr)
	if err == nil {
		t.Error("expected error for empty YAML")
	}
}

func TestParseYAML_comments(t *testing.T) {
	yamlStr := `# This is a comment
name: John  # inline comment
# Another comment
age: 30`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("expected 'John', got %v", result["name"])
	}

	// YAML may return int or float64
	age := result["age"]
	ageNum := 0.0
	switch v := age.(type) {
	case int:
		ageNum = float64(v)
	case float64:
		ageNum = v
	default:
		t.Fatalf("expected age to be numeric, got %T", age)
	}
	if ageNum != 30 {
		t.Errorf("expected 30, got %v", age)
	}
}

func TestParseYAML_multilineString(t *testing.T) {
	yamlStr := `description: |
  This is a multiline
  string that spans
  multiple lines`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	desc, ok := result["description"].(string)
	if !ok {
		t.Fatalf("expected description to be string, got %T", result["description"])
	}

	if desc != "This is a multiline\nstring that spans\nmultiple lines" {
		t.Errorf("unexpected multiline content: %q", desc)
	}
}

func TestReadYAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlContent := `name: John
age: 30`
	yamlPath := filepath.Join(tmpDir, "params.yaml")
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write yaml file: %v", err)
	}

	result, err := ReadYAMLFile(yamlPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "John" {
		t.Errorf("expected 'John', got %v", result["name"])
	}

	// YAML may return int or float64
	age := result["age"]
	ageNum := 0.0
	switch v := age.(type) {
	case int:
		ageNum = float64(v)
	case float64:
		ageNum = v
	default:
		t.Fatalf("expected age to be numeric, got %T", age)
	}
	if ageNum != 30 {
		t.Errorf("expected 30, got %v", age)
	}
}

func TestReadYAMLFile_notFound(t *testing.T) {
	_, err := ReadYAMLFile("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadYAMLFile_invalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidContent := `name: [broken`
	yamlPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(yamlPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write invalid yaml: %v", err)
	}

	_, err := ReadYAMLFile(yamlPath)
	if err == nil {
		t.Error("expected error for invalid YAML in file")
	}
}

func TestIsPipedInput_true(t *testing.T) {
	// Create a pipe with data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	// Write some data
	w.Write([]byte("test"))
	w.Close()

	// Save original stdin and restore after test
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Replace stdin with our pipe reader
	os.Stdin = r

	if !IsPipedInput() {
		t.Error("expected IsPipedInput to return true for pipe with data")
	}
}

func TestIsPipedInput_false(t *testing.T) {
	// This test works because in normal test execution,
	// stdin is not a pipe. We can't easily mock this,
	// but we can verify the function returns false when
	// stdin is a terminal (which is the default in tests)
	result := IsPipedInput()
	if result != false {
		t.Errorf("expected false for non-piped stdin, got %v", result)
	}
}

func TestReadStdinYAML(t *testing.T) {
	// Create a pipe with YAML data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()

	yamlContent := "name: piped\nage: 25"
	w.Write([]byte(yamlContent))
	w.Close()

	// Save original stdin
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	// Replace stdin
	os.Stdin = r

	result, err := ReadStdinYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "piped" {
		t.Errorf("expected 'piped', got %v", result["name"])
	}

	// YAML may return int or float64
	age := result["age"]
	ageNum := 0.0
	switch v := age.(type) {
	case int:
		ageNum = float64(v)
	case float64:
		ageNum = v
	default:
		t.Fatalf("expected age to be numeric, got %T", age)
	}
	if ageNum != 25 {
		t.Errorf("expected 25, got %v", age)
	}
}

// Output format tests

func TestParseYAML_variousFormats(t *testing.T) {
	stringTests := []struct {
		name    string
		yaml    string
		wantKey string
		wantVal any
	}{
		{"string value", "name: test", "name", "test"},
		{"number value", "count: 42", "count", 42},
		{"float value", "price: 19.99", "price", 19.99},
		{"boolean true", "enabled: true", "enabled", true},
		{"boolean false", "disabled: false", "disabled", false},
	}

	for _, tt := range stringTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseYAML(tt.yaml)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result[tt.wantKey] != tt.wantVal {
				t.Errorf("got %v, want %v", result[tt.wantKey], tt.wantVal)
			}
		})
	}

	// Test array separately (slices can't be compared with ==)
	t.Run("array value", func(t *testing.T) {
		result, err := ParseYAML("items:\n  - a\n  - b\n  - c")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		arr, ok := result["items"].([]any)
		if !ok {
			t.Fatalf("expected array, got %T", result["items"])
		}
		if len(arr) != 3 {
			t.Errorf("expected 3 items, got %d", len(arr))
		}
		if arr[0] != "a" || arr[1] != "b" || arr[2] != "c" {
			t.Errorf("unexpected array content: %v", arr)
		}
	})

	// Test nested object separately
	t.Run("nested object", func(t *testing.T) {
		result, err := ParseYAML("config:\n  debug: true\n  level: 5")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		cfg, ok := result["config"].(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result["config"])
		}
		if cfg["debug"] != true || cfg["level"] != 5 {
			t.Errorf("unexpected config: %v", cfg)
		}
	})
}

func TestParseYAML_mixedContent(t *testing.T) {
	yamlStr := `# Comment line
name: mixed-test
tags:
  - yaml
  - input
config:
  debug: true
  timeout: 30
emptyArray: []
emptyObj: {}
`
	result, err := ParseYAML(yamlStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["name"] != "mixed-test" {
		t.Errorf("expected 'mixed-test', got %v", result["name"])
	}

	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags to be array")
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}

	config, ok := result["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected config to be map")
	}
	if config["debug"] != true {
		t.Errorf("expected debug=true")
	}
}

func TestParseYAML_quotedStrings(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		key  string
		want string
	}{
		{"double quotes", `s1: "hello"`, "s1", "hello"},
		{"single quotes", `s2: 'world'`, "s2", "world"},
		{"mixed quotes", `s3: "it's cool"`, "s3", "it's cool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseYAML(tt.yaml)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val := result[tt.key]
			if val != tt.want {
				t.Errorf("got %v, want %v", val, tt.want)
			}
		})
	}
}
