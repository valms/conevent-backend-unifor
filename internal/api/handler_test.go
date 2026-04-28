package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valms/conevent-backend-unifor/internal/observability"
	"github.com/valms/conevent-backend-unifor/internal/service"
)

type mockEventService struct {
	event  *service.Event
	events []*service.Event
	err    error
	lastID string
}

func (m *mockEventService) GetEvent(ctx context.Context, eventID string) (*service.Event, error) {
	m.lastID = eventID
	return m.event, m.err
}

func (m *mockEventService) ListEvents(ctx context.Context) ([]*service.Event, error) {
	return m.events, m.err
}

func (m *mockEventService) CreateEvent(ctx context.Context, event *service.Event) error {
	if m.err != nil {
		return m.err
	}
	event.ID = "550e8400-e29b-41d4-a716-446655440000"
	return nil
}

func (m *mockEventService) UpdateEvent(ctx context.Context, event *service.Event) error {
	m.lastID = event.ID
	return m.err
}

func (m *mockEventService) DeleteEvent(ctx context.Context, eventID string) error {
	m.lastID = eventID
	return m.err
}

func newTestApp(s service.EventService) *fiber.App {
	app := fiber.New()
	h := NewEventHandler(s, nil)
	app.Get("/health", HealthCheck)
	app.Get("/events", h.ListEvents)
	app.Post("/events", h.CreateEvent)
	app.Get("/events/:id", h.GetEvent)
	app.Put("/events/:id", h.UpdateEvent)
	app.Delete("/events/:id", h.DeleteEvent)
	RegisterDocs(app)
	return app
}

func newTestAppWithMetrics(t *testing.T, s service.EventService) *fiber.App {
	t.Helper()
	bm, err := observability.NewBusinessMetrics("api-handler-test")
	require.NoError(t, err)
	app := fiber.New()
	h := NewEventHandler(s, bm)
	app.Get("/events", h.ListEvents)
	app.Post("/events", h.CreateEvent)
	app.Get("/events/:id", h.GetEvent)
	app.Put("/events/:id", h.UpdateEvent)
	app.Delete("/events/:id", h.DeleteEvent)
	return app
}

func doRequest(t *testing.T, app *fiber.App, method, target, body string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

func TestHealthCheck(t *testing.T) {
	app := newTestApp(&mockEventService{})
	resp := doRequest(t, app, http.MethodGet, "/health", "")
	body := readBody(t, resp)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `"status":"ok"`)
	assert.Contains(t, body, `"message":"Conevent API is running"`)
}

func TestEventHandlers_Success(t *testing.T) {
	event := &service.Event{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Test"}
	app := newTestApp(&mockEventService{event: event, events: []*service.Event{event}})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{"list", http.MethodGet, "/events", "", http.StatusOK},
		{"get", http.MethodGet, "/events/550e8400-e29b-41d4-a716-446655440000", "", http.StatusOK},
		{"create", http.MethodPost, "/events", `{"name":"Test"}`, http.StatusCreated},
		{"update", http.MethodPut, "/events/550e8400-e29b-41d4-a716-446655440000", `{"name":"Test"}`, http.StatusOK},
		{"delete", http.MethodDelete, "/events/550e8400-e29b-41d4-a716-446655440000", "", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRequest(t, app, tt.method, tt.path, tt.body)
			assert.Equal(t, tt.status, resp.StatusCode)
		})
	}
}

func TestEventHandlers_BadRequests(t *testing.T) {
	app := newTestApp(&mockEventService{})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   string
	}{
		{"create invalid body", http.MethodPost, "/events", `{`, "Invalid request body"},
		{"update invalid body", http.MethodPut, "/events/123", `{`, "Invalid request body"},
		{"update id mismatch", http.MethodPut, "/events/123", `{"id":"456"}`, "Event ID mismatch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRequest(t, app, tt.method, tt.path, tt.body)
			body := readBody(t, resp)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, body, tt.want)
		})
	}
}

func TestEventHandlers_ServiceErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"invalid", service.ErrEventInvalid, http.StatusBadRequest, "Invalid event data"},
		{"not found", pgx.ErrNoRows, http.StatusNotFound, "Event not found"},
		{"internal", errors.New("database down"), http.StatusInternalServerError, "Internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(&mockEventService{err: tt.err})
			resp := doRequest(t, app, http.MethodGet, "/events/550e8400-e29b-41d4-a716-446655440000", "")
			body := readBody(t, resp)
			assert.Equal(t, tt.status, resp.StatusCode)
			assert.Contains(t, body, tt.want)
		})
	}
}

func TestAllEventHandlers_ServiceErrors(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"list", http.MethodGet, "/events", ""},
		{"create", http.MethodPost, "/events", `{"name":"Test"}`},
		{"update", http.MethodPut, "/events/550e8400-e29b-41d4-a716-446655440000", `{"name":"Test"}`},
		{"delete", http.MethodDelete, "/events/550e8400-e29b-41d4-a716-446655440000", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newTestApp(&mockEventService{err: errors.New("boom")})
			resp := doRequest(t, app, tt.method, tt.path, tt.body)
			body := readBody(t, resp)
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
			assert.Contains(t, body, "Internal server error")
		})
	}
}

func TestEventHandlers_WithBusinessMetrics(t *testing.T) {
	event := &service.Event{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Test"}
	app := newTestAppWithMetrics(t, &mockEventService{event: event, events: []*service.Event{event}})

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/events", ""},
		{http.MethodGet, "/events/550e8400-e29b-41d4-a716-446655440000", ""},
		{http.MethodPost, "/events", `{"name":"Test"}`},
		{http.MethodPut, "/events/550e8400-e29b-41d4-a716-446655440000", `{"name":"Test"}`},
		{http.MethodDelete, "/events/550e8400-e29b-41d4-a716-446655440000", ""},
	}

	for _, req := range requests {
		resp := doRequest(t, app, req.method, req.path, req.body)
		assert.Less(t, resp.StatusCode, http.StatusInternalServerError)
	}

	app = newTestAppWithMetrics(t, &mockEventService{err: service.ErrEventInvalid})
	resp := doRequest(t, app, http.MethodGet, "/events/550e8400-e29b-41d4-a716-446655440000", "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDocsHandlers(t *testing.T) {
	app := newTestApp(&mockEventService{})

	resp := doRequest(t, app, http.MethodGet, "/openapi.json", "")
	body := readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `"openapi":"3.1.0"`)
	assert.Contains(t, body, `"/events/{id}"`)

	resp = doRequest(t, app, http.MethodGet, "/docs", "")
	body = readBody(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), fiber.MIMETextHTML)
	assert.Contains(t, body, "/openapi.json")
}

func TestOpenAPISpecShape(t *testing.T) {
	spec := OpenAPISpec()
	assert.Equal(t, "3.1.0", spec["openapi"])
	assert.Contains(t, spec, "paths")
	assert.Contains(t, spec, "components")
}
