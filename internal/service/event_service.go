package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"time"

	"conevent-backend/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// EventService defines the interface for event-related operations
type EventService interface {
	// GetEvent returns an event by ID
	GetEvent(eventID string) (*Event, error)
	// ListEvents returns a list of events
	ListEvents() ([]*Event, error)
	// CreateEvent creates a new event
	CreateEvent(event *Event) error
	// UpdateEvent updates an existing event
	UpdateEvent(event *Event) error
	// DeleteEvent deletes an event by ID
	DeleteEvent(eventID string) error
}

// Event represents an event in the system
type Event struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IniDate   string  `json:"iniDate"`
	EndDate   string  `json:"endDate"`
	IniTime   string  `json:"iniTime"`
	EndTime   string  `json:"endTime"`
	Location  string  `json:"location"`
	Budget    float64 `json:"budget"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"createdAt"`
}

// ErrEventInvalid error definitions
var (
	ErrEventInvalid = errors.New("invalid event data")
)

// eventService implements EventService using database operations
type eventService struct {
	querier *db.Queries
	logger  *slog.Logger
}

// NewEventService creates a new EventService instance
func NewEventService(querier *db.Queries) EventService {
	return &eventService{
		querier: querier,
		logger:  slog.Default(),
	}
}

// GetEvent returns an event by ID
func (s *eventService) GetEvent(eventID string) (*Event, error) {
	s.logger.Info("getting event", "event_id", eventID)

	id, err := s.ParseUUID(eventID)
	if err != nil {
		s.logger.Error("failed to parse event ID", "error", err)
		return nil, err
	}

	event, err := s.querier.GetEvent(context.Background(), id)
	if err != nil {
		s.logger.Error("failed to get event", "error", err)
		return nil, err
	}

	s.logger.Info("event retrieved successfully", "event_id", event.ID.String())
	return s.convertDBEvent(event), nil
}

// ListEvents returns a list of events
func (s *eventService) ListEvents() ([]*Event, error) {
	s.logger.Info("listing events")

	events, err := s.querier.ListEvents(context.Background())
	if err != nil {
		s.logger.Error("failed to list events", "error", err)
		return nil, err
	}

	result := make([]*Event, 0, len(events))
	for _, event := range events {
		result = append(result, s.convertDBEvent(event))
	}

	s.logger.Info("events listed successfully", "count", len(events))
	return result, nil
}

// CreateEvent creates a new event
func (s *eventService) CreateEvent(event *Event) error {
	s.logger.Info("creating event", "name", event.Name)

	if err := s.validateEvent(event); err != nil {
		s.logger.Error("event validation failed", "error", err)
		return err
	}

	dbEvent, err := s.convertToDBEvent(event)
	if err != nil {
		s.logger.Error("failed to convert event", "error", err)
		return err
	}

	createdEvent, err := s.querier.CreateEvent(context.Background(), dbEvent)
	if err != nil {
		s.logger.Error("failed to create event", "error", err)
		return err
	}

	s.logger.Info("event created successfully", "event_id", createdEvent.ID.String())

	// Update the event with the created values
	event.ID = createdEvent.ID.String()
	s.copyCreatedEvent(event, createdEvent)

	return nil
}

// UpdateEvent updates an existing event
func (s *eventService) UpdateEvent(event *Event) error {
	s.logger.Info("updating event", "event_id", event.ID)

	if event.ID == "" {
		s.logger.Error("event ID is required")
		return ErrEventInvalid
	}

	id, err := s.ParseUUID(event.ID)
	if err != nil {
		s.logger.Error("failed to parse event ID", "error", err)
		return err
	}

	dbEvent, err := s.convertToDBEventForUpdate(event, id)
	if err != nil {
		s.logger.Error("failed to convert event", "error", err)
		return err
	}
	dbEvent.ID = id

	_, err = s.querier.UpdateEvent(context.Background(), *dbEvent)
	if err != nil {
		s.logger.Error("failed to update event", "error", err)
		return err
	}

	s.logger.Info("event updated successfully", "event_id", event.ID)
	return nil
}

// DeleteEvent deletes an event by ID
func (s *eventService) DeleteEvent(eventID string) error {
	s.logger.Info("deleting event", "event_id", eventID)

	if eventID == "" {
		s.logger.Error("event ID is required")
		return ErrEventInvalid
	}

	id, err := s.ParseUUID(eventID)
	if err != nil {
		s.logger.Error("failed to parse event ID", "error", err)
		return err
	}

	if err := s.querier.DeleteEvent(context.Background(), id); err != nil {
		s.logger.Error("failed to delete event", "error", err)
		return err
	}

	s.logger.Info("event deleted successfully", "event_id", eventID)
	return nil
}

// ParseUUID parses a UUID string (with or without hyphens) to pgtype.UUID
// Takes last 32 characters removing any hyphens
func (s *eventService) ParseUUID(id string) (pgtype.UUID, error) {
	var idBytes [16]byte

	// Extract only hex digits (last 32 chars)
	idToParse := id
	if len(id) == 36 {
		// Remove hyphens: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		idToParse = id[0:8] + id[9:13] + id[14:18] + id[19:23] + id[24:]
	}

	if len(idToParse) != 32 {
		return pgtype.UUID{}, ErrEventInvalid
	}

	if _, err := hex.Decode(idBytes[:], []byte(idToParse)); err != nil {
		return pgtype.UUID{}, ErrEventInvalid
	}

	return pgtype.UUID{Bytes: idBytes, Valid: true}, nil
}

// FormatDate formats pgtype.Date to string (YYYY-MM-DD)
func (s *eventService) FormatDate(date pgtype.Date) string {
	if !date.Valid {
		return ""
	}
	return date.Time.Format("2006-01-02")
}

// FormatTime formats pgtype.Time to string (HH:MM)
func (s *eventService) FormatTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	hours := t.Microseconds / 3600000000
	minutes := (t.Microseconds % 3600000000) / 60000000
	return fmt.Sprintf("%02d:%02d", hours, minutes)
}

// ParseDate parses date string (YYYY-MM-DD) to pgtype.Date
func (s *eventService) ParseDate(dateStr string) (pgtype.Date, error) {
	if dateStr == "" {
		return pgtype.Date{}, nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

// ParseTime parses time string (HH:MM) to pgtype.Time
func (s *eventService) ParseTime(timeStr string) (pgtype.Time, error) {
	if timeStr == "" {
		return pgtype.Time{}, nil
	}
	t, err := time.Parse("15:04", timeStr)
	if err != nil {
		return pgtype.Time{}, err
	}
	return pgtype.Time{Microseconds: int64(t.Hour()*3600+t.Minute()*60) * 1000000, Valid: true}, nil
}

// Helper: validate event required fields
func (s *eventService) validateEvent(event *Event) error {
	if event.Name == "" || event.IniDate == "" || event.EndDate == "" ||
		event.IniTime == "" || event.EndTime == "" || event.Location == "" ||
		event.Status == "" {
		return ErrEventInvalid
	}
	return nil
}

// Helper: convert budget float64 to pgtype.Numeric (assuming 2 decimal places)
func (s *eventService) convertBudgetToNumeric(budget float64) (pgtype.Numeric, error) {
	budgetInt := new(big.Int).Mul(big.NewInt(int64(budget*100)), big.NewInt(1))
	return pgtype.Numeric{Int: budgetInt, Exp: -2, Valid: true}, nil
}

// Helper: convert service Event to db CreateEventParams
func (s *eventService) convertToDBEvent(event *Event) (db.CreateEventParams, error) {
	iniDate, err := s.ParseDate(event.IniDate)
	if err != nil {
		return db.CreateEventParams{}, err
	}
	endDate, err := s.ParseDate(event.EndDate)
	if err != nil {
		return db.CreateEventParams{}, err
	}
	iniTime, err := s.ParseTime(event.IniTime)
	if err != nil {
		return db.CreateEventParams{}, err
	}
	endTime, err := s.ParseTime(event.EndTime)
	if err != nil {
		return db.CreateEventParams{}, err
	}

	budget, err := s.convertBudgetToNumeric(event.Budget)
	if err != nil {
		return db.CreateEventParams{}, err
	}

	return db.CreateEventParams{
		Name:     event.Name,
		IniDate:  iniDate,
		EndDate:  endDate,
		IniTime:  iniTime,
		EndTime:  endTime,
		Location: event.Location,
		Budget:   budget,
		Status:   event.Status,
	}, nil
}

// Helper: convert service Event to db UpdateEventParams
func (s *eventService) convertToDBEventForUpdate(event *Event, id pgtype.UUID) (*db.UpdateEventParams, error) {
	iniDate, err := s.ParseDate(event.IniDate)
	if err != nil {
		return nil, err
	}
	endDate, err := s.ParseDate(event.EndDate)
	if err != nil {
		return nil, err
	}
	iniTime, err := s.ParseTime(event.IniTime)
	if err != nil {
		return nil, err
	}
	endTime, err := s.ParseTime(event.EndTime)
	if err != nil {
		return nil, err
	}

	budget, err := s.convertBudgetToNumeric(event.Budget)
	if err != nil {
		return nil, err
	}

	return &db.UpdateEventParams{
		ID:       id,
		Name:     event.Name,
		IniDate:  iniDate,
		EndDate:  endDate,
		IniTime:  iniTime,
		EndTime:  endTime,
		Location: event.Location,
		Budget:   budget,
		Status:   event.Status,
	}, nil
}

// Helper: convert db.Event to service Event
func (s *eventService) convertDBEvent(event db.Event) *Event {
	return &Event{
		ID:        event.ID.String(),
		Name:      event.Name,
		IniDate:   s.FormatDate(event.IniDate),
		EndDate:   s.FormatDate(event.EndDate),
		IniTime:   s.FormatTime(event.IniTime),
		EndTime:   s.FormatTime(event.EndTime),
		Location:  event.Location,
		Budget:    s.convertBudget(event.Budget),
		Status:    event.Status,
		CreatedAt: s.formatTimeStamp(event.CreatedAt),
	}
}

// Helper: convert pgtype.Numeric to float64
func (s *eventService) convertBudget(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	if f, err := n.Float64Value(); err == nil {
		return f.Float64
	}
	return 0
}

// Helper: format pgtype.Timestamptz to string (ISO 8601)
func (s *eventService) formatTimeStamp(ts pgtype.Timestamptz) string {
	if !ts.Valid {
		return ""
	}
	return ts.Time.Format("2006-01-02T15:04:05Z07:00")
}

// Helper: copy created event values to original event
func (s *eventService) copyCreatedEvent(event *Event, createdEvent db.Event) {
	event.IniDate = s.FormatDate(createdEvent.IniDate)
	event.EndDate = s.FormatDate(createdEvent.EndDate)
	event.IniTime = s.FormatTime(createdEvent.IniTime)
	event.EndTime = s.FormatTime(createdEvent.EndTime)
	event.Location = createdEvent.Location
	event.Budget = s.convertBudget(createdEvent.Budget)
	event.Status = createdEvent.Status
	event.CreatedAt = s.formatTimeStamp(createdEvent.CreatedAt)
}
