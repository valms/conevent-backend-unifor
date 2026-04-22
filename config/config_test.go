package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Set up environment variables
	os.Setenv("SERVER_PORT", "8080")
	os.Setenv("SERVER_READ_TIMEOUT", "10")
	os.Setenv("SERVER_WRITE_TIMEOUT", "20")
	os.Setenv("SERVER_IDLE_TIMEOUT", "120")
	os.Setenv("DB_HOST", "testhost")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "testuser")
	os.Setenv("DB_PASSWORD", "testpass")
	os.Setenv("DB_NAME", "testdb")
	os.Setenv("DB_SSLMODE", "require")

	defer func() {
		// Clean up
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("SERVER_READ_TIMEOUT")
		os.Unsetenv("SERVER_WRITE_TIMEOUT")
		os.Unsetenv("SERVER_IDLE_TIMEOUT")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Validate server config
	if cfg.Server.Port != "8080" {
		t.Errorf("Expected Server.Port to be '8080', got '%s'", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("Expected Server.ReadTimeout to be 10s, got '%v'", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 20*time.Second {
		t.Errorf("Expected Server.WriteTimeout to be 20s, got '%v'", cfg.Server.WriteTimeout)
	}
	if cfg.Server.IdleTimeout != 120*time.Second {
		t.Errorf("Expected Server.IdleTimeout to be 120s, got '%v'", cfg.Server.IdleTimeout)
	}

	// Validate database config
	if cfg.Database.Host != "testhost" {
		t.Errorf("Expected Database.Host to be 'testhost', got '%s'", cfg.Database.Host)
	}
	if cfg.Database.Port != "5433" {
		t.Errorf("Expected Database.Port to be '5433', got '%s'", cfg.Database.Port)
	}
	if cfg.Database.User != "testuser" {
		t.Errorf("Expected Database.User to be 'testuser', got '%s'", cfg.Database.User)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("Expected Database.Password to be 'testpass', got '%s'", cfg.Database.Password)
	}
	if cfg.Database.Name != "testdb" {
		t.Errorf("Expected Database.Name to be 'testdb', got '%s'", cfg.Database.Name)
	}
	if cfg.Database.SSLMode != "require" {
		t.Errorf("Expected Database.SSLMode to be 'require', got '%s'", cfg.Database.SSLMode)
	}
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	// Clear all DB-related env vars to test validation
	originalHost := os.Getenv("DB_HOST")
	originalPort := os.Getenv("DB_PORT")
	originalUser := os.Getenv("DB_USER")
	originalPassword := os.Getenv("DB_PASSWORD")
	originalName := os.Getenv("DB_NAME")

	os.Unsetenv("DB_HOST")
	os.Unsetenv("DB_PORT")
	os.Unsetenv("DB_USER")
	os.Unsetenv("DB_PASSWORD")
	os.Unsetenv("DB_NAME")

	defer func() {
		// Restore original values
		if originalHost != "" {
			os.Setenv("DB_HOST", originalHost)
		} else {
			os.Unsetenv("DB_HOST")
		}
		if originalPort != "" {
			os.Setenv("DB_PORT", originalPort)
		} else {
			os.Unsetenv("DB_PORT")
		}
		if originalUser != "" {
			os.Setenv("DB_USER", originalUser)
		} else {
			os.Unsetenv("DB_USER")
		}
		if originalPassword != "" {
			os.Setenv("DB_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("DB_PASSWORD")
		}
		if originalName != "" {
			os.Setenv("DB_NAME", originalName)
		} else {
			os.Unsetenv("DB_NAME")
		}
	}()

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatalf("Expected error due to missing DB config, got cfg=%v", cfg)
	}
	if cfg != nil {
		t.Fatalf("Expected nil config, got %v", cfg)
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "conevent",
			SSLMode:  "disable",
		},
	}

	if err := cfg.validate(); err != nil {
		t.Fatalf("Expected no validation error, got %v", err)
	}
}

func TestValidate_MissingHost(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Host:     "", // Missing
			Port:     "5432",
			User:     "postgres",
			Password: "postgres",
			Name:     "conevent",
			SSLMode:  "disable",
		},
	}

	err := cfg.validate()
	if err == nil {
		t.Fatalf("Expected validation error for missing host")
		return
	}
	if err.Error() != "DB_HOST is required" {
		t.Errorf("Expected error 'DB_HOST is required', got '%v'", err)
	}
}
