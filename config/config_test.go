package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Arrange
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
		vars := []string{
			"SERVER_PORT",
			"SERVER_READ_TIMEOUT",
			"SERVER_WRITE_TIMEOUT",
			"SERVER_IDLE_TIMEOUT",
			"DB_HOST",
			"DB_PORT",
			"DB_USER",
			"DB_PASSWORD",
			"DB_NAME",
			"DB_SSLMODE",
		}
		for _, v := range vars {
			if err := os.Unsetenv(v); err != nil {
				t.Fatalf("failed to unset %s: %v", v, err)
			}
		}
	}()

	// Act
	cfg, err := LoadConfig()

	// Assert
	assert.NoError(t, err, "Expected no error")
	assert.Equal(t, "8080", cfg.Server.Port, "Expected Server.Port to be '8080'")
	assert.Equal(t, 10*time.Second, cfg.Server.ReadTimeout, "Expected Server.ReadTimeout to be 10s")
	assert.Equal(t, 20*time.Second, cfg.Server.WriteTimeout, "Expected Server.WriteTimeout to be 20s")
	assert.Equal(t, 120*time.Second, cfg.Server.IdleTimeout, "Expected Server.IdleTimeout to be 120s")
	assert.Equal(t, "testhost", cfg.Database.Host, "Expected Database.Host to be 'testhost'")
	assert.Equal(t, "5433", cfg.Database.Port, "Expected Database.Port to be '5433'")
	assert.Equal(t, "testuser", cfg.Database.User, "Expected Database.User to be 'testuser'")
	assert.Equal(t, "testpass", cfg.Database.Password, "Expected Database.Password to be 'testpass'")
	assert.Equal(t, "testdb", cfg.Database.Name, "Expected Database.Name to be 'testdb'")
	assert.Equal(t, "require", cfg.Database.SSLMode, "Expected Database.SSLMode to be 'require'")
}

func TestLoadConfig_MissingRequired(t *testing.T) {
	// Arrange
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
			if err := os.Setenv("DB_HOST", originalHost); err != nil {
				t.Fatalf("failed to set DB_HOST: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_HOST"); err != nil {
				t.Fatalf("failed to unset DB_HOST: %v", err)
			}
		}
		if originalPort != "" {
			if err := os.Setenv("DB_PORT", originalPort); err != nil {
				t.Fatalf("failed to set DB_PORT: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_PORT"); err != nil {
				t.Fatalf("failed to unset DB_PORT: %v", err)
			}
		}
		if originalUser != "" {
			if err := os.Setenv("DB_USER", originalUser); err != nil {
				t.Fatalf("failed to set DB_USER: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_USER"); err != nil {
				t.Fatalf("failed to unset DB_USER: %v", err)
			}
		}
		if originalPassword != "" {
			if err := os.Setenv("DB_PASSWORD", originalPassword); err != nil {
				t.Fatalf("failed to set DB_PASSWORD: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_PASSWORD"); err != nil {
				t.Fatalf("failed to unset DB_PASSWORD: %v", err)
			}
		}
		if originalName != "" {
			if err := os.Setenv("DB_NAME", originalName); err != nil {
				t.Fatalf("failed to set DB_NAME: %v", err)
			}
		} else {
			if err := os.Unsetenv("DB_NAME"); err != nil {
				t.Fatalf("failed to unset DB_NAME: %v", err)
			}
		}
	}()

	// Act
	cfg, err := LoadConfig()

	// Assert
	assert.Error(t, err, "Expected error due to missing DB config")
	assert.Nil(t, cfg, "Expected nil config")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		cfg           Config
		expectedError string
	}{
		{
			name: "valid config",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "5432",
					User:     "postgres",
					Password: "postgres",
					Name:     "conevent",
					SSLMode:  "disable",
				},
			},
			expectedError: "",
		},
		{
			name: "missing host",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "",
					Port:     "5432",
					User:     "postgres",
					Password: "postgres",
					Name:     "conevent",
					SSLMode:  "disable",
				},
			},
			expectedError: "DB_HOST is required",
		},
		{
			name: "missing port",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "",
					User:     "postgres",
					Password: "postgres",
					Name:     "conevent",
					SSLMode:  "disable",
				},
			},
			expectedError: "DB_PORT is required",
		},
		{
			name: "missing user",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "5432",
					User:     "",
					Password: "postgres",
					Name:     "conevent",
					SSLMode:  "disable",
				},
			},
			expectedError: "DB_USER is required",
		},
		{
			name: "missing password",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "5432",
					User:     "postgres",
					Password: "",
					Name:     "conevent",
					SSLMode:  "disable",
				},
			},
			expectedError: "DB_PASSWORD is required",
		},
		{
			name: "missing name",
			cfg: Config{
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     "5432",
					User:     "postgres",
					Password: "postgres",
					Name:     "",
					SSLMode:  "disable",
				},
			},
			expectedError: "DB_NAME is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := tt.cfg.validate()

			// Assert
			if tt.expectedError == "" {
				assert.NoError(t, err, "Expected no error")
			} else {
				assert.Error(t, err, "Expected validation error")
				assert.Equal(t, tt.expectedError, err.Error(), "Expected specific error message")
			}
		})
	}
}
