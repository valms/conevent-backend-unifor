package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestHealthCheck(t *testing.T) {
	// Create a new Fiber app for testing
	app := fiber.New()
	app.Get("/health", HealthCheck)

	// Create a request to test our endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	// Serve the request using Fiber's test interface
	respFiber, err := app.Test(req)
	if err != nil {
		t.Fatalf("Error during test: %v", err)
	}

	// Check the response
	if respFiber.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, respFiber.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(respFiber.Body)
	body := string(bodyBytes)
	if !bytes.Contains([]byte(body), []byte(`"status":"ok"`)) {
		t.Errorf("Expected response to contain 'status\":\"ok\"', got %s", body)
	}
	if !bytes.Contains([]byte(body), []byte(`"message":"Conevent API is running"`)) {
		t.Errorf("Expected response to contain 'message\":\"Conevent API is running\"', got %s", body)
	}
}
