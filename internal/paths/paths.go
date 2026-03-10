package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigDir returns the config directory and ensures it exists
// Priority: 1) CARELINK_CONFIG_DIR env var, 2) ~/.carelink/config
func GetConfigDir() (string, error) {
	var configDir string

	// Check for custom config directory from environment variable
	if customDir := os.Getenv("CARELINK_CONFIG_DIR"); customDir != "" {
		configDir = customDir
	} else {
		// Default to ~/.carelink/config
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		configDir = filepath.Join(home, ".carelink", "config")
	}

	// Create if doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// GetDataDir returns the data directory and ensures it exists
// Priority: 1) CARELINK_DATA_DIR env var, 2) ~/.carelink/data
func GetDataDir() (string, error) {
	var dataDir string

	// Check for custom data directory from environment variable
	if customDir := os.Getenv("CARELINK_DATA_DIR"); customDir != "" {
		dataDir = customDir
	} else {
		// Default to ~/.carelink/data
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".carelink", "data")
	}

	// Create if doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return dataDir, nil
}

// FindFile searches for a file in multiple locations
// Priority: 1) current directory, 2) config directory
func FindFile(filename string) (string, error) {
	// Check current directory
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	// Check config directory
	configDir, err := GetConfigDir()
	if err == nil {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("file %s not found in current directory or config directory", filename)
}

// GetDefaultDBPath returns the default database path (data_dir/carelink.db)
func GetDefaultDBPath() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "carelink.db"), nil
}
