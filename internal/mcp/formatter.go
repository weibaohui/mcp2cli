package mcp

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// FormatInputSchema converts JSON Schema to human-readable format
// Returns: ["key:type={value} // description", ...]
func FormatInputSchema(schema any) []string {
	if schema == nil {
		return nil
	}

	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	properties, _ := schemaMap["properties"].(map[string]any)
	if properties == nil {
		return nil
	}

	var result []string
	for key, prop := range properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		jsonType, _ := propMap["type"].(string)
		typeHint := jsonTypeToTypeHint(jsonType)

		// Format: key:type={value} // description
		line := key + ":" + typeHint + "={value}"
		if desc, ok := propMap["description"].(string); ok && desc != "" {
			line += " // " + desc
		}
		result = append(result, line)
	}

	return result
}

// jsonTypeToTypeHint maps JSON types to type hints
func jsonTypeToTypeHint(jsonType string) string {
	switch jsonType {
	case "number":
		return "number"
	case "integer":
		return "int"
	case "boolean":
		return "bool"
	case "array":
		return "array"
	case "object":
		return "object"
	case "string":
		return "string"
	default:
		return "string"
	}
}

// GetRequiredParams extracts required parameter names
func GetRequiredParams(schema any) []string {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	var result []string
	if required, ok := schemaMap["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

// GetParamInfoList extracts structured parameter information
func GetParamInfoList(schema any) []ParamInfo {
	schemaMap, ok := schema.(map[string]any)
	if !ok {
		return nil
	}

	properties, _ := schemaMap["properties"].(map[string]any)
	if properties == nil {
		return nil
	}

	// Build required set
	requiredSet := make(map[string]bool)
	if required, ok := schemaMap["required"].([]any); ok {
		for _, r := range required {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	var result []ParamInfo
	for key, prop := range properties {
		propMap, ok := prop.(map[string]any)
		if !ok {
			continue
		}

		jsonType, _ := propMap["type"].(string)
		info := ParamInfo{
			Name:        key,
			Type:        jsonTypeToTypeHint(jsonType),
			Required:    requiredSet[key],
			Description: "",
		}
		if desc, ok := propMap["description"].(string); ok {
			info.Description = desc
		}
		result = append(result, info)
	}

	return result
}

// buildCallExample builds a command example string
func buildCallExample(params []ParamInfo) string {
	if len(params) == 0 {
		return ""
	}

	var parts []string
	for _, p := range params {
		parts = append(parts, fmt.Sprintf("%s:%s={value}", p.Name, p.Type))
	}
	return strings.Join(parts, " ")
}

// ParseKVArgs parses "key=value" or "key:type=value" format arguments
func ParseKVArgs(args []string) (map[string]any, error) {
	result := make(map[string]any, len(args))

	for _, arg := range args {
		// Find first '='
		idx := strings.Index(arg, "=")
		if idx < 0 {
			return nil, NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("invalid argument format: %q (expected key=value or key:type=value)", arg))
		}

		keyPart := arg[:idx]
		valStr := arg[idx+1:]

		// Parse key and optional type annotation
		key, typeHint, err := parseKeyWithType(keyPart)
		if err != nil {
			return nil, err
		}

		// Convert value based on type
		convertedVal, err := convertValue(valStr, typeHint)
		if err != nil {
			return nil, NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("argument %q value %q cannot be converted to type %q: %v", key, valStr, typeHint, err))
		}

		result[key] = convertedVal
	}

	return result, nil
}

// parseKeyWithType parses "key" or "key:type"
func parseKeyWithType(keyPart string) (string, string, error) {
	// Find colon separator
	colonIdx := strings.Index(keyPart, ":")
	if colonIdx < 0 {
		// No type annotation, default to string
		if keyPart == "" {
			return "", "", NewMCPError(ErrCodeParamInvalid, "argument key cannot be empty")
		}
		return keyPart, "string", nil
	}

	key := keyPart[:colonIdx]
	typeHint := keyPart[colonIdx+1:]

	if key == "" {
		return "", "", NewMCPError(ErrCodeParamInvalid, "argument key cannot be empty")
	}

	// Validate type
	validTypes := map[string]bool{
		"string": true, "number": true, "int": true,
		"float": true, "bool": true, "boolean": true,
	}
	if !validTypes[typeHint] {
		return "", "", NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("argument %q uses unsupported type %q, supported types: string, number, int, float, bool", key, typeHint))
	}

	return key, typeHint, nil
}

// convertValue converts a string value based on type hint
func convertValue(valStr, typeHint string) (any, error) {
	switch typeHint {
	case "string":
		return valStr, nil

	case "number", "int":
		// Try integer first
		if intVal, err := strconv.ParseInt(valStr, 10, 64); err == nil {
			return float64(intVal), nil
		}
		// Try float
		return strconv.ParseFloat(valStr, 64)

	case "float":
		return strconv.ParseFloat(valStr, 64)

	case "bool", "boolean":
		lower := strings.ToLower(valStr)
		if lower == "true" || lower == "1" || lower == "yes" {
			return true, nil
		}
		if lower == "false" || lower == "0" || lower == "no" {
			return false, nil
		}
		return nil, fmt.Errorf("cannot parse %q as boolean", valStr)

	default:
		return valStr, nil
	}
}

// ParseYAML parses a YAML string and returns a map[string]any
func ParseYAML(yamlStr string) (map[string]any, error) {
	if strings.TrimSpace(yamlStr) == "" {
		return nil, NewMCPError(ErrCodeParamInvalid, "empty YAML input")
	}

	var result map[string]any
	if err := yaml.Unmarshal([]byte(yamlStr), &result); err != nil {
		return nil, NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("invalid YAML: %v", err))
	}

	if result == nil {
		return nil, NewMCPError(ErrCodeParamInvalid, "empty YAML input")
	}

	return result, nil
}

// ReadYAMLFile reads and parses a YAML file
func ReadYAMLFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("cannot read YAML file %q: %v", path, err))
	}

	return ParseYAML(string(data))
}

// IsPipedInput checks if stdin is a pipe (piped input)
func IsPipedInput() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// Check if stdin is a pipe (mode char device + named pipe)
	// ModeCharDevice = 0x2000 for Windows, not relevant for Unix
	// ModeNamedPipe = 0x4000 for named pipe
	mode := stat.Mode()
	return (mode & os.ModeNamedPipe) != 0
}

// ReadStdinYAML reads YAML from stdin and returns a map
func ReadStdinYAML() (map[string]any, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, NewMCPError(ErrCodeParamInvalid, fmt.Sprintf("failed to read stdin: %v", err))
	}

	if len(data) == 0 {
		return nil, NewMCPError(ErrCodeParamInvalid, "empty stdin input")
	}

	return ParseYAML(string(data))
}
