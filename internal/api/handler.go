package api

import (
	"github.com/gofiber/fiber/v2"
)

// HealthCheck returns a simple health check response
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Conevent API is running",
	})
}
