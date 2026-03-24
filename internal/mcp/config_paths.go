package mcp

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigSearchPaths returns platform-specific config file search paths
func GetConfigSearchPaths() []string {
	if runtime.GOOS == "windows" {
		return getWindowsConfigPaths()
	}
	return getUnixConfigPaths()
}

func getUnixConfigPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}

	var paths []string

	// High priority paths
	if home != "" {
		paths = append(paths,
			filepath.Join(home, ".config", "modelcontextprotocol", "mcp.json"),
			filepath.Join(home, ".config", "mcp", "config.json"),
		)
	}

	// Current directory paths
	paths = append(paths,
		"./mcp.json",
		"./.mcp/config.json",
	)

	// System-level path
	paths = append(paths, "/etc/mcp/config.json")

	return paths
}

func getWindowsConfigPaths() []string {
	appData := os.Getenv("APPDATA")
	userProfile := os.Getenv("USERPROFILE")
	programData := os.Getenv("ProgramData")

	var paths []string

	// High priority: APPDATA paths
	if appData != "" {
		paths = append(paths,
			filepath.Join(appData, "modelcontextprotocol", "mcp.json"),
			filepath.Join(appData, "mcp", "config.json"),
		)
	}

	// USERPROFILE path
	if userProfile != "" {
		paths = append(paths, filepath.Join(userProfile, ".mcp", "config.json"))
	}

	// Current directory paths
	paths = append(paths,
		".\\mcp.json",
		".\\.mcp\\config.json",
	)

	// System-level path
	if programData != "" {
		paths = append(paths, filepath.Join(programData, "mcp", "config.json"))
	}

	return paths
}

// ExpandHome expands ~ in paths to user's home directory
func ExpandHome(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
