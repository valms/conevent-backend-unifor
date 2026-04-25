package api

import (
	"github.com/gofiber/fiber/v2"

	"conevent-backend/internal/service"
)

// EventHandler handles HTTP requests for event management
type EventHandler struct {
	service service.EventService
}

// NewEventHandler creates a new EventHandler instance
func NewEventHandler(s service.EventService) *EventHandler {
	return &EventHandler{service: s}
}

// GetEvent returns a single event by ID
func (h *EventHandler) GetEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	event, err := h.service.GetEvent(eventID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(event)
}

// ListEvents returns a list of all events
func (h *EventHandler) ListEvents(c *fiber.Ctx) error {
	events, err := h.service.ListEvents()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(events)
}

// CreateEvent creates a new event
func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.CreateEvent(&event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(event)
}

// UpdateEvent updates an existing event
func (h *EventHandler) UpdateEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Ensure the ID from URL matches the ID in body (if provided)
	if event.ID != "" && event.ID != eventID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID mismatch",
		})
	}
	event.ID = eventID

	if err := h.service.UpdateEvent(&event); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(event)
}

// DeleteEvent deletes an event by ID
func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	eventID := c.Params("id")
	if eventID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	if err := h.service.DeleteEvent(eventID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// HealthCheck returns a simple health check response
func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Conevent API is running",
	})
}
