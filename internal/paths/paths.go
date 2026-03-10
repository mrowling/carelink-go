package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetConfigDir returns ~/.carelink and ensures it exists
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(home, ".carelink")

	// Create if doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// FindFile searches for a file in multiple locations
// Priority: 1) current directory, 2) ~/.carelink/
func FindFile(filename string) (string, error) {
	// Check current directory
	if _, err := os.Stat(filename); err == nil {
		return filename, nil
	}

	// Check ~/.carelink/
	configDir, err := GetConfigDir()
	if err == nil {
		path := filepath.Join(configDir, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("file %s not found in current directory or ~/.carelink/", filename)
}

// GetDefaultDBPath returns the default database path (~/.carelink/carelink.db)
func GetDefaultDBPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "carelink.db"), nil
}
