// shared/db/databaseManagement.go
package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DatabaseConfig stores the Turso database connection details
type DatabaseConfig struct {
	DatabaseURL string `json:"database_url"`
	AuthToken   string `json:"auth_token"`
}

// getConfigFilePath returns the path to the config file
func getConfigFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(configDir, "BlissfulBytesManagement")

	// Create the directory if it doesn't exist
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(appConfigDir, "database_config.json"), nil
}

// SaveDbConfig saves the database configuration to a JSON file
func SaveDbConfig(config DatabaseConfig) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, configJSON, 0644)
}

// LoadDbConfig loads the database configuration from the JSON file
func LoadDbConfig() (DatabaseConfig, error) {
	configPath, err := getConfigFilePath()
	if err != nil {
		return DatabaseConfig{}, err
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		// If file doesn't exist, return an empty config
		if os.IsNotExist(err) {
			return DatabaseConfig{}, nil
		}
		return DatabaseConfig{}, err
	}

	var config DatabaseConfig
	err = json.Unmarshal(configData, &config)
	return config, err
}

// UpdateEnvForDbConfig updates the environment variables with the saved config
func UpdateEnvForDbConfig() error {
	config, err := LoadDbConfig()
	if err != nil {
		return fmt.Errorf("error loading database config: %v", err)
	}

	// Set environment variables if config exists
	if config.DatabaseURL != "" {
		os.Setenv("TURSO_DATABASE_URL", config.DatabaseURL)
	}
	if config.AuthToken != "" {
		os.Setenv("TURSO_AUTH_TOKEN", config.AuthToken)
	}

	return nil
}

// ValidateDbConfig checks if the database configuration is complete
func ValidateDbConfig() error {
	config, err := LoadDbConfig()
	if err != nil {
		return fmt.Errorf("error loading database config: %v", err)
	}

	if config.DatabaseURL == "" {
		return fmt.Errorf("database URL is missing")
	}

	if config.AuthToken == "" {
		return fmt.Errorf("authentication token is missing")
	}

	return nil
}
