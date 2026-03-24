package mcp

import (
	"errors"
	"fmt"
)

// Error codes
const (
	ErrCodeConfigNotFound  = "MCP_CONFIG_NOT_FOUND"
	ErrCodeConnectFailed   = "MCP_CONNECT_FAILED"
	ErrCodeServerNotFound  = "MCP_SERVER_NOT_FOUND"
	ErrCodeMethodNotFound  = "MCP_METHOD_NOT_FOUND"
	ErrCodeMethodAmbiguous = "MCP_METHOD_AMBIGUOUS"
	ErrCodeCallFailed      = "MCP_CALL_FAILED"
	ErrCodeParamInvalid    = "MCP_PARAM_INVALID"
)

// MCPError represents an MCP-specific error
type MCPError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewMCPError creates a new MCPError
func NewMCPError(code, message string) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
	}
}

// NewMCPErrorWithDetails creates a new MCPError with details
func NewMCPErrorWithDetails(code, message string, details map[string]any) *MCPError {
	return &MCPError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ConfigErrors returns a ConfigNotFound error
func ConfigErrors(message string) *MCPError {
	return NewMCPError(ErrCodeConfigNotFound, message)
}

// ConnectErrors returns a ConnectFailed error
func ConnectErrors(serverName, message string) *MCPError {
	return NewMCPErrorWithDetails(ErrCodeConnectFailed, message, map[string]any{
		"server": serverName,
	})
}

// ServerNotFoundErrors returns a ServerNotFound error
func ServerNotFoundErrors(serverName string) *MCPError {
	return NewMCPErrorWithDetails(ErrCodeServerNotFound, fmt.Sprintf("Server %q not found in config", serverName), map[string]any{
		"server": serverName,
	})
}

// MethodNotFoundErrors returns a MethodNotFound error
func MethodNotFoundErrors(toolName, serverName string) *MCPError {
	return NewMCPErrorWithDetails(ErrCodeMethodNotFound, fmt.Sprintf("Tool %q not found on server %q", toolName, serverName), map[string]any{
		"method": toolName,
		"server": serverName,
	})
}

// CallErrors returns a CallFailed error
func CallErrors(toolName, serverName string, err error) *MCPError {
	return NewMCPErrorWithDetails(ErrCodeCallFailed, fmt.Sprintf("Failed to call tool %q on server %q: %v", toolName, serverName, err), map[string]any{
		"method": toolName,
		"server": serverName,
	})
}

// ParamErrors returns a ParamInvalid error
func ParamErrors(message string) *MCPError {
	return NewMCPError(ErrCodeParamInvalid, message)
}

// IsMCPError checks if an error is an MCPError
func IsMCPError(err error) bool {
	var mcpErr *MCPError
	return errors.As(err, &mcpErr)
}

// GetMCPError extracts MCPError from an error
func GetMCPError(err error) *MCPError {
	var mcpErr *MCPError
	if errors.As(err, &mcpErr) {
		return mcpErr
	}
	return nil
}
