package api

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"conevent-backend/internal/observability"
	"conevent-backend/internal/service"
)

type EventHandler struct {
	service         service.EventService
	businessMetrics *observability.BusinessMetrics
}

func NewEventHandler(s service.EventService, bm *observability.BusinessMetrics) *EventHandler {
	return &EventHandler{service: s, businessMetrics: bm}
}

func (h *EventHandler) GetEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	if eventID == "" {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordGetEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	event, err := h.service.GetEvent(eventID)
	if err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordGetEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordGetEvent(c.UserContext(), time.Since(start), true)
	}
	return c.JSON(event)
}

func (h *EventHandler) ListEvents(c *fiber.Ctx) error {
	start := time.Now()
	events, err := h.service.ListEvents()
	if err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordListEvents(c.UserContext(), time.Since(start), 0)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordListEvents(c.UserContext(), time.Since(start), len(events))
	}
	return c.JSON(events)
}

func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	start := time.Now()
	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := h.service.CreateEvent(&event); err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
	}
	return c.Status(fiber.StatusCreated).JSON(event)
}

func (h *EventHandler) UpdateEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	if eventID == "" {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if event.ID != "" && event.ID != eventID {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID mismatch",
		})
	}
	event.ID = eventID

	if err := h.service.UpdateEvent(&event); err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), true)
	}
	return c.JSON(event)
}

func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	if eventID == "" {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordDeleteEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Event ID is required",
		})
	}

	if err := h.service.DeleteEvent(eventID); err != nil {
		if h.businessMetrics != nil {
			h.businessMetrics.RecordDeleteEvent(c.UserContext(), time.Since(start), false)
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordDeleteEvent(c.UserContext(), time.Since(start), true)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"message": "Conevent API is running",
	})
}
