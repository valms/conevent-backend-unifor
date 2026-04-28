package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitTracingStdout(t *testing.T) {
	tp, err := InitTracing(context.Background(), Config{ServiceName: "test", ServiceVersion: "1.0.0", Exporter: "stdout"})
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NoError(t, tp.Shutdown(context.Background()))
}

func TestInitTracingDefaultExporter(t *testing.T) {
	tp, err := InitTracing(context.Background(), Config{ServiceName: "test", ServiceVersion: "1.0.0", Exporter: "unknown"})
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NoError(t, tp.Shutdown(context.Background()))
}

func TestInitTracingOTLPExporters(t *testing.T) {
	tests := []struct {
		name     string
		exporter string
		endpoint string
	}{
		{"grpc", "otlp", "localhost:4317"},
		{"jaeger alias", "jaeger", "localhost:4317"},
		{"http", "otlphttp", "localhost:4318"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tp, err := InitTracing(context.Background(), Config{ServiceName: "test", ServiceVersion: "1.0.0", Exporter: tt.exporter, OTLPEndpoint: tt.endpoint})
			require.NoError(t, err)
			require.NotNil(t, tp)
			require.NoError(t, tp.Shutdown(context.Background()))
		})
	}
}

func TestInitMetricsAndSetupApp(t *testing.T) {
	mp, reg, err := InitMetrics(context.Background(), Config{ServiceName: "test", ServiceVersion: "1.0.0"})
	require.NoError(t, err)
	require.NotNil(t, mp)
	require.NotNil(t, reg)
	defer mp.Shutdown(context.Background())

	app := fiber.New()
	SetupApp(app, "test")
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestShouldSkipTrace(t *testing.T) {
	app := fiber.New()
	app.Get("/health", func(c *fiber.Ctx) error {
		assert.True(t, ShouldSkipTrace(c))
		return c.SendStatus(http.StatusNoContent)
	})
	app.Get("/events", func(c *fiber.Ctx) error {
		assert.False(t, ShouldSkipTrace(c))
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	resp, err = app.Test(httptest.NewRequest(http.MethodGet, "/events", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
