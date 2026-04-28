package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsRecorderAndCustomMetrics(t *testing.T) {
	recorder := NewMetricsRecorder("test-service")
	require.NotNil(t, recorder)
	require.NoError(t, InitCustomMetrics("test-service"))

	IncActiveRequests(context.Background())
	DecActiveRequests(context.Background())
}

func TestFirstError(t *testing.T) {
	wantErr := errors.New("boom")
	assert.NoError(t, firstError(nil, nil))
	assert.ErrorIs(t, firstError(nil, wantErr), wantErr)
}

func TestMetricsMiddleware(t *testing.T) {
	require.NoError(t, InitCustomMetrics("middleware-test"))
	app := fiber.New()
	middleware := NewMetricsMiddleware()
	app.Use(middleware.Handle)
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBusinessMetricsRecorders(t *testing.T) {
	m, err := NewBusinessMetrics("business-test")
	require.NoError(t, err)
	require.NotNil(t, m)

	ctx := context.Background()
	m.RecordListEvents(ctx, time.Millisecond, 2)
	m.RecordGetEvent(ctx, time.Millisecond, true)
	m.RecordGetEvent(ctx, time.Millisecond, false)
	m.RecordCreateEvent(ctx, time.Millisecond)
	m.RecordUpdateEvent(ctx, time.Millisecond, true)
	m.RecordUpdateEvent(ctx, time.Millisecond, false)
	m.RecordDeleteEvent(ctx, time.Millisecond, true)
	m.RecordDeleteEvent(ctx, time.Millisecond, false)
	m.RecordDatabaseQuery(ctx, "select", time.Millisecond)
}

func TestRecordRequestMetricsWithoutInitDoesNotPanic(t *testing.T) {
	httpRequestsTotal = nil
	httpRequestDuration = nil

	app := fiber.New()
	app.Get("/ping", func(c *fiber.Ctx) error {
		RecordRequestMetrics(c, time.Millisecond)
		return c.SendStatus(http.StatusNoContent)
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/ping", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestStartHTTPServerConfiguration(t *testing.T) {
	reg := prometheus.NewRegistry()
	err := StartHTTPServer("-1", reg)
	assert.Error(t, err)
}
