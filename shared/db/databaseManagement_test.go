// shared/db/databaseManagement_test.go
package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGetConfigFilePath verifies that the config file path is correctly constructed
func TestGetConfigFilePath(t *testing.T) {
	path, err := getConfigFilePath()
	if err != nil {
		t.Fatalf("Failed to get config file path: %v", err)
	}

	// Check that path ends with the expected filename
	if filepath.Base(path) != "database_config.json" {
		t.Errorf("Expected filename to be database_config.json, got %s", filepath.Base(path))
	}

	// Check that the parent directory is BlissfulBytesManagement
	parentDir := filepath.Base(filepath.Dir(path))
	if parentDir != "BlissfulBytesManagement" {
		t.Errorf("Expected parent directory to be BlissfulBytesManagement, got %s", parentDir)
	}
}

// TestSaveAndLoadDbConfig tests the save and load operations for database configuration
func TestSaveAndLoadDbConfig(t *testing.T) {
	// Create a test config
	testConfig := DatabaseConfig{
		DatabaseURL: "libsql://test-database.turso.io",
		AuthToken:   "test-token-12345",
	}

	// Save the test config
	err := SaveDbConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to save database config: %v", err)
	}

	// Load the config back
	loadedConfig, err := LoadDbConfig()
	if err != nil {
		t.Fatalf("Failed to load database config: %v", err)
	}

	// Verify the loaded config matches the saved config
	if loadedConfig.DatabaseURL != testConfig.DatabaseURL {
		t.Errorf("Expected Database URL %s, got %s", testConfig.DatabaseURL, loadedConfig.DatabaseURL)
	}

	if loadedConfig.AuthToken != testConfig.AuthToken {
		t.Errorf("Expected Auth Token %s, got %s", testConfig.AuthToken, loadedConfig.AuthToken)
	}

	// Clean up test config file
	configPath, _ := getConfigFilePath()
	os.Remove(configPath)
}

// TestUpdateEnvForDbConfig tests that environment variables are correctly updated
func TestUpdateEnvForDbConfig(t *testing.T) {
	// Create and save a test config
	testConfig := DatabaseConfig{
		DatabaseURL: "libsql://test-env-database.turso.io",
		AuthToken:   "test-env-token-67890",
	}

	err := SaveDbConfig(testConfig)
	if err != nil {
		t.Fatalf("Failed to save database config: %v", err)
	}

	// Update environment variables
	err = UpdateEnvForDbConfig()
	if err != nil {
		t.Fatalf("Failed to update environment variables: %v", err)
	}

	// Verify environment variables were set correctly
	if os.Getenv("TURSO_DATABASE_URL") != testConfig.DatabaseURL {
		t.Errorf("Expected env TURSO_DATABASE_URL to be %s, got %s",
			testConfig.DatabaseURL, os.Getenv("TURSO_DATABASE_URL"))
	}

	if os.Getenv("TURSO_AUTH_TOKEN") != testConfig.AuthToken {
		t.Errorf("Expected env TURSO_AUTH_TOKEN to be %s, got %s",
			testConfig.AuthToken, os.Getenv("TURSO_AUTH_TOKEN"))
	}

	// Clean up
	configPath, _ := getConfigFilePath()
	os.Remove(configPath)
	os.Unsetenv("TURSO_DATABASE_URL")
	os.Unsetenv("TURSO_AUTH_TOKEN")
}

// TestValidateDbConfig tests configuration validation
func TestValidateDbConfig(t *testing.T) {
	// Test with empty config (should fail)
	emptyConfig := DatabaseConfig{}
	err := SaveDbConfig(emptyConfig)
	if err != nil {
		t.Fatalf("Failed to save empty config: %v", err)
	}

	err = ValidateDbConfig()
	if err == nil {
		t.Error("Expected validation to fail with empty config, but it passed")
	}

	// Test with URL only (should fail)
	urlOnlyConfig := DatabaseConfig{
		DatabaseURL: "libsql://test-database.turso.io",
		AuthToken:   "",
	}
	err = SaveDbConfig(urlOnlyConfig)
	if err != nil {
		t.Fatalf("Failed to save URL-only config: %v", err)
	}

	err = ValidateDbConfig()
	if err == nil {
		t.Error("Expected validation to fail with missing auth token, but it passed")
	}

	// Test with token only (should fail)
	tokenOnlyConfig := DatabaseConfig{
		DatabaseURL: "",
		AuthToken:   "test-token-12345",
	}
	err = SaveDbConfig(tokenOnlyConfig)
	if err != nil {
		t.Fatalf("Failed to save token-only config: %v", err)
	}

	err = ValidateDbConfig()
	if err == nil {
		t.Error("Expected validation to fail with missing URL, but it passed")
	}

	// Test with complete config (should pass)
	completeConfig := DatabaseConfig{
		DatabaseURL: "libsql://test-database.turso.io",
		AuthToken:   "test-token-12345",
	}
	err = SaveDbConfig(completeConfig)
	if err != nil {
		t.Fatalf("Failed to save complete config: %v", err)
	}

	err = ValidateDbConfig()
	if err != nil {
		t.Errorf("Expected validation to pass with complete config, but got error: %v", err)
	}

	// Clean up
	configPath, _ := getConfigFilePath()
	os.Remove(configPath)
}

// TestLoadDbConfig_NonExistentFile tests loading when config file doesn't exist
func TestLoadDbConfig_NonExistentFile(t *testing.T) {
	// Ensure config file doesn't exist
	configPath, _ := getConfigFilePath()
	os.Remove(configPath)

	// Attempt to load non-existent config
	config, err := LoadDbConfig()
	if err != nil {
		t.Errorf("Expected no error when loading non-existent config, got: %v", err)
	}

	// Check that returned config is empty
	emptyConfig := DatabaseConfig{}
	if config != emptyConfig {
		t.Errorf("Expected empty config when file doesn't exist, got: %+v", config)
	}
}

// TestLoadDbConfig_InvalidJSON tests loading when config file contains invalid JSON
func TestLoadDbConfig_InvalidJSON(t *testing.T) {
	// Get config path
	configPath, err := getConfigFilePath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	// Create directory if needed
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Write invalid JSON
	err = os.WriteFile(configPath, []byte("this is not valid json"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	// Attempt to load invalid config
	_, err = LoadDbConfig()
	if err == nil {
		t.Error("Expected error when loading invalid JSON, but got none")
	}

	// Clean up
	os.Remove(configPath)
}

// Helper function to create a mock config for testing
func createMockConfig(t *testing.T, config DatabaseConfig) {
	configPath, err := getConfigFilePath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}

	// Create directory if needed
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, configJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}
