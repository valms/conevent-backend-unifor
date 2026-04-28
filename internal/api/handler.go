package api

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/valms/conevent-backend-unifor/internal/observability"
	"github.com/valms/conevent-backend-unifor/internal/service"
)

type EventHandler struct {
	service         service.EventService
	businessMetrics *observability.BusinessMetrics
	tracer          trace.Tracer
}

func NewEventHandler(s service.EventService, bm *observability.BusinessMetrics) *EventHandler {
	return &EventHandler{
		service:         s,
		businessMetrics: bm,
		tracer:          otel.Tracer("github.com/valms/conevent-backend-unifor/internal/api"),
	}
}

func startHandlerSpan(c *fiber.Ctx, tracer trace.Tracer, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := tracer.Start(c.UserContext(), operation,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
	c.SetUserContext(ctx)
	span.AddEvent("handler.started", trace.WithAttributes(attribute.String("http.route", c.Route().Path)))
	return ctx, span
}

func finishHandlerSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return
	}
	span.SetStatus(codes.Ok, "handler completed")
	span.AddEvent("handler.completed")
	span.End()
}

func errorResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func serviceErrorResponse(c *fiber.Ctx, err error) error {
	if errors.Is(err, service.ErrEventInvalid) {
		return errorResponse(c, fiber.StatusBadRequest, "Invalid event data")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return errorResponse(c, fiber.StatusNotFound, "Event not found")
	}
	return errorResponse(c, fiber.StatusInternalServerError, "Internal server error")
}

func (h *EventHandler) GetEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	ctx, span := startHandlerSpan(c, h.tracer, "EventHandler.GetEvent", attribute.String("event.id", eventID))
	var spanErr error
	defer func() { finishHandlerSpan(span, spanErr) }()

	event, err := h.service.GetEvent(ctx, eventID)
	if err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordGetEvent(c.UserContext(), time.Since(start), false)
		}
		return serviceErrorResponse(c, err)
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordGetEvent(c.UserContext(), time.Since(start), true)
	}
	return c.JSON(event)
}

func (h *EventHandler) ListEvents(c *fiber.Ctx) error {
	start := time.Now()
	ctx, span := startHandlerSpan(c, h.tracer, "EventHandler.ListEvents")
	var spanErr error
	defer func() { finishHandlerSpan(span, spanErr) }()

	events, err := h.service.ListEvents(ctx)
	if err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordListEvents(c.UserContext(), time.Since(start), 0)
		}
		return serviceErrorResponse(c, err)
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordListEvents(c.UserContext(), time.Since(start), len(events))
	}
	span.SetAttributes(attribute.Int("events.count", len(events)))
	return c.JSON(events)
}

func (h *EventHandler) CreateEvent(c *fiber.Ctx) error {
	start := time.Now()
	ctx, span := startHandlerSpan(c, h.tracer, "EventHandler.CreateEvent")
	var spanErr error
	defer func() { finishHandlerSpan(span, spanErr) }()

	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
		}
		return errorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	span.SetAttributes(attribute.String("event.name", event.Name))
	if err := h.service.CreateEvent(ctx, &event); err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
		}
		return serviceErrorResponse(c, err)
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordCreateEvent(c.UserContext(), time.Since(start))
	}
	span.SetAttributes(attribute.String("event.id", event.ID))
	return c.Status(fiber.StatusCreated).JSON(event)
}

func (h *EventHandler) UpdateEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	ctx, span := startHandlerSpan(c, h.tracer, "EventHandler.UpdateEvent", attribute.String("event.id", eventID))
	var spanErr error
	defer func() { finishHandlerSpan(span, spanErr) }()

	var event service.Event
	if err := c.BodyParser(&event); err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return errorResponse(c, fiber.StatusBadRequest, "Invalid request body")
	}

	if event.ID != "" && event.ID != eventID {
		spanErr = service.ErrEventInvalid
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return errorResponse(c, fiber.StatusBadRequest, "Event ID mismatch")
	}
	event.ID = eventID

	span.SetAttributes(attribute.String("event.name", event.Name))
	if err := h.service.UpdateEvent(ctx, &event); err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), false)
		}
		return serviceErrorResponse(c, err)
	}

	if h.businessMetrics != nil {
		h.businessMetrics.RecordUpdateEvent(c.UserContext(), time.Since(start), true)
	}
	return c.JSON(event)
}

func (h *EventHandler) DeleteEvent(c *fiber.Ctx) error {
	start := time.Now()
	eventID := c.Params("id")
	ctx, span := startHandlerSpan(c, h.tracer, "EventHandler.DeleteEvent", attribute.String("event.id", eventID))
	var spanErr error
	defer func() { finishHandlerSpan(span, spanErr) }()

	if err := h.service.DeleteEvent(ctx, eventID); err != nil {
		spanErr = err
		if h.businessMetrics != nil {
			h.businessMetrics.RecordDeleteEvent(c.UserContext(), time.Since(start), false)
		}
		return serviceErrorResponse(c, err)
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
