package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Arrange
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("SERVER_READ_TIMEOUT", "10")
	t.Setenv("SERVER_WRITE_TIMEOUT", "20")
	t.Setenv("SERVER_IDLE_TIMEOUT", "120")
	t.Setenv("DB_HOST", "testhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("DB_SSLMODE", "require")

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
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")

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
				Observability: ObservabilityConfig{
					ServiceName:    "conevent-backend",
					ServiceVersion: "1.0.0",
					TraceExporter:  "stdout",
					OTLPEndpoint:   "",
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
		{
			name: "missing trace exporter",
			cfg: Config{
				Database:      DatabaseConfig{Host: "localhost", Port: "5432", User: "postgres", Password: "postgres", Name: "conevent", SSLMode: "disable"},
				Observability: ObservabilityConfig{TraceExporter: ""},
			},
			expectedError: "OBS_TRACE_EXPORTER is required",
		},
		{
			name: "missing otlp endpoint",
			cfg: Config{
				Database:      DatabaseConfig{Host: "localhost", Port: "5432", User: "postgres", Password: "postgres", Name: "conevent", SSLMode: "disable"},
				Observability: ObservabilityConfig{TraceExporter: "jaeger", OTLPEndpoint: ""},
			},
			expectedError: "OBS_OTLP_ENDPOINT is required",
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

func TestDatabaseURL(t *testing.T) {
	cfg := &Config{Database: DatabaseConfig{Host: "localhost", Port: "5432", User: "user", Password: "pass", Name: "db", SSLMode: "disable"}}
	assert.Equal(t, "host=localhost port=5432 user=user password=pass dbname=db sslmode=disable", cfg.DatabaseURL())
}

func TestGetEnvAsIntFallbacks(t *testing.T) {
	t.Setenv("INT_VALUE", "invalid")
	assert.Equal(t, 5, getEnvAsInt("INT_VALUE", 5))

	t.Setenv("INT_VALUE", "0")
	assert.Equal(t, 5, getEnvAsInt("INT_VALUE", 5))

	t.Setenv("INT_VALUE", "7")
	assert.Equal(t, 7, getEnvAsInt("INT_VALUE", 5))
}
