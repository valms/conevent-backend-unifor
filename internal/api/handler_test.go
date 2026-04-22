package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()
	app.Get("/health", HealthCheck)

	// Create a request to test our endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Serve the request using Fiber's test interface
	respFiber, err := app.Test(req)
	assert.NoError(t, err, "Expected no error when testing handler")

	// Check the response status code
	assert.Equal(t, http.StatusOK, respFiber.StatusCode, "Expected HTTP OK status")

	// Read and check the response body
	bodyBytes, _ := io.ReadAll(respFiber.Body)
	body := string(bodyBytes)
	assert.Contains(t, body, `"status":"ok"`, "Expected response to contain 'status\":\"ok\"'")
	assert.Contains(t, body, `"message":"Conevent API is running"`, "Expected response to contain 'message\":\"Conevent API is running\"'")
}
