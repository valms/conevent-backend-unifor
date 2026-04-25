package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
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
}

// NewEventService creates a new EventService instance
func NewEventService(querier *db.Queries) EventService {
	return &eventService{querier: querier}
}

// GetEvent returns an event by ID
func (s *eventService) GetEvent(eventID string) (*Event, error) {
	// Parse UUID
	var idBytes [16]byte
	if len(eventID) == 36 {
		// Standard UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		if len(eventID) != 36 {
			return nil, ErrEventInvalid
		}
		// Remove hyphens
		hexStr := eventID[0:8] + eventID[9:13] + eventID[14:18] + eventID[19:23] + eventID[24:]
		if len(hexStr) != 32 {
			return nil, ErrEventInvalid
		}
		// Decode hex
		if _, err := hex.Decode(idBytes[:], []byte(hexStr)); err != nil {
			return nil, ErrEventInvalid
		}
	} else if len(eventID) == 32 {
		// Compact UUID format
		if _, err := hex.Decode(idBytes[:], []byte(eventID)); err != nil {
			return nil, ErrEventInvalid
		}
	} else {
		return nil, ErrEventInvalid
	}
	id := pgtype.UUID{Bytes: idBytes, Valid: true}

	event, err := s.querier.GetEvent(context.Background(), id)
	if err != nil {
		return nil, err
	}

	// Convert pgtype values to our API format
	var iniDateStr, endDateStr, iniTimeStr, endTimeStr string
	if event.IniDate.Valid {
		iniDateStr = event.IniDate.Time.Format("2006-01-02")
	}
	if event.EndDate.Valid {
		endDateStr = event.EndDate.Time.Format("2006-01-02")
	}
	if event.IniTime.Valid {
		// Convert microseconds to HH:MM format
		hours := event.IniTime.Microseconds / 3600000000
		minutes := (event.IniTime.Microseconds % 3600000000) / 60000000
		iniTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
	}
	if event.EndTime.Valid {
		// Convert microseconds to HH:MM format
		hours := event.EndTime.Microseconds / 3600000000
		minutes := (event.EndTime.Microseconds % 3600000000) / 60000000
		endTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
	}
	var budgetFloat64 float64
	if event.Budget.Valid {
		f8, err := event.Budget.Float64Value()
		if err != nil {
			return nil, err
		}
		budgetFloat64 = f8.Float64
	}
	var createdAtStr string
	if event.CreatedAt.Valid {
		createdAtStr = event.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return &Event{
		ID:        event.ID.String(),
		Name:      event.Name,
		IniDate:   iniDateStr,
		EndDate:   endDateStr,
		IniTime:   iniTimeStr,
		EndTime:   endTimeStr,
		Location:  event.Location,
		Budget:    budgetFloat64,
		Status:    event.Status,
		CreatedAt: createdAtStr,
	}, nil
}

// ListEvents returns a list of events
func (s *eventService) ListEvents() ([]*Event, error) {
	events, err := s.querier.ListEvents(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]*Event, 0, len(events))
	for _, event := range events {
		// Convert microseconds to HH:MM format for times
		var iniTimeStr, endTimeStr string
		if event.IniTime.Valid {
			hours := event.IniTime.Microseconds / 3600000000
			minutes := (event.IniTime.Microseconds % 3600000000) / 60000000
			iniTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
		}
		if event.EndTime.Valid {
			hours := event.EndTime.Microseconds / 3600000000
			minutes := (event.EndTime.Microseconds % 3600000000) / 60000000
			endTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
		}

		// Convert Numeric to float64
		var budgetFloat64 float64
		if event.Budget.Valid {
			f8, err := event.Budget.Float64Value()
			if err != nil {
				return nil, err
			}
			budgetFloat64 = f8.Float64
		}

		result = append(result, &Event{
			ID:        event.ID.String(),
			Name:      event.Name,
			IniDate:   event.IniDate.Time.Format("2006-01-02"),
			EndDate:   event.EndDate.Time.Format("2006-01-02"),
			IniTime:   iniTimeStr,
			EndTime:   endTimeStr,
			Location:  event.Location,
			Budget:    budgetFloat64,
			Status:    event.Status,
			CreatedAt: event.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return result, nil
}

// CreateEvent creates a new event
func (s *eventService) CreateEvent(event *Event) error {
	// Validate required fields
	if event.Name == "" || event.IniDate == "" || event.EndDate == "" ||
		event.IniTime == "" || event.EndTime == "" || event.Location == "" ||
		event.Status == "" {
		return ErrEventInvalid
	}

	// Parse date values
	iniDateTime, err := time.Parse("2006-01-02", event.IniDate)
	if err != nil {
		return err
	}
	endDateTime, err := time.Parse("2006-01-02", event.EndDate)
	if err != nil {
		return err
	}

	// Parse time values
	iniTimeParsed, err := time.Parse("15:04", event.IniTime)
	if err != nil {
		return err
	}
	endTimeParsed, err := time.Parse("15:04", event.EndTime)
	if err != nil {
		return err
	}

	// Convert to pgtype values
	iniDate := pgtype.Date{Time: iniDateTime, Valid: true}
	endDate := pgtype.Date{Time: endDateTime, Valid: true}
	iniTime := pgtype.Time{Microseconds: int64(iniTimeParsed.Hour()*3600+iniTimeParsed.Minute()*60) * 1000000, Valid: true}
	endTime := pgtype.Time{Microseconds: int64(endTimeParsed.Hour()*3600+endTimeParsed.Minute()*60) * 1000000, Valid: true}

	// Convert budget to pgtype.Numeric (assuming 2 decimal places)
	budgetInt := new(big.Int).Mul(big.NewInt(int64(event.Budget*100)), big.NewInt(1))
	budget := pgtype.Numeric{Int: budgetInt, Exp: -2, Valid: true}

	dbEvent := db.CreateEventParams{
		Name:     event.Name,
		IniDate:  iniDate,
		EndDate:  endDate,
		IniTime:  iniTime,
		EndTime:  endTime,
		Location: event.Location,
		Budget:   budget,
		Status:   event.Status,
	}

	createdEvent, err := s.querier.CreateEvent(context.Background(), dbEvent)
	if err != nil {
		return err
	}

	// Convert pgtype values to our API format
	var iniDateStr, endDateStr, iniTimeStr, endTimeStr string
	if createdEvent.IniDate.Valid {
		iniDateStr = createdEvent.IniDate.Time.Format("2006-01-02")
	}
	if createdEvent.EndDate.Valid {
		endDateStr = createdEvent.EndDate.Time.Format("2006-01-02")
	}
	if createdEvent.IniTime.Valid {
		// Convert microseconds to HH:MM format
		hours := createdEvent.IniTime.Microseconds / 3600000000
		minutes := (createdEvent.IniTime.Microseconds % 3600000000) / 60000000
		iniTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
	}
	if createdEvent.EndTime.Valid {
		// Convert microseconds to HH:MM format
		hours := createdEvent.EndTime.Microseconds / 3600000000
		minutes := (createdEvent.EndTime.Microseconds % 3600000000) / 60000000
		endTimeStr = fmt.Sprintf("%02d:%02d", hours, minutes)
	}
	var budgetFloat64 float64
	if createdEvent.Budget.Valid {
		f8, err := createdEvent.Budget.Float64Value()
		if err != nil {
			return err
		}
		budgetFloat64 = f8.Float64
	}
	var createdAtStr string
	if createdEvent.CreatedAt.Valid {
		createdAtStr = createdEvent.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	// Update the event with the created values
	event.ID = createdEvent.ID.String()
	event.IniDate = iniDateStr
	event.EndDate = endDateStr
	event.IniTime = iniTimeStr
	event.EndTime = endTimeStr
	event.Location = createdEvent.Location
	event.Budget = budgetFloat64
	event.Status = createdEvent.Status
	event.CreatedAt = createdAtStr

	return nil
}

// UpdateEvent updates an existing event
func (s *eventService) UpdateEvent(event *Event) error {
	if event.ID == "" {
		return ErrEventInvalid
	}

	// Parse UUID
	var idBytes [16]byte
	if len(event.ID) == 36 {
		// Standard UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		if len(event.ID) != 36 {
			return ErrEventInvalid
		}
		// Remove hyphens
		hexStr := event.ID[0:8] + event.ID[9:13] + event.ID[14:18] + event.ID[19:23] + event.ID[24:]
		if len(hexStr) != 32 {
			return ErrEventInvalid
		}
		// Decode hex
		if _, err := hex.Decode(idBytes[:], []byte(hexStr)); err != nil {
			return ErrEventInvalid
		}
	} else if len(event.ID) == 32 {
		// Compact UUID format
		if _, err := hex.Decode(idBytes[:], []byte(event.ID)); err != nil {
			return ErrEventInvalid
		}
	} else {
		return ErrEventInvalid
	}
	id := pgtype.UUID{Bytes: idBytes, Valid: true}

	// Parse date values
	iniDateTime, err := time.Parse("2006-01-02", event.IniDate)
	if err != nil {
		return err
	}
	endDateTime, err := time.Parse("2006-01-02", event.EndDate)
	if err != nil {
		return err
	}

	// Parse time values
	iniTimeParsed, err := time.Parse("15:04", event.IniTime)
	if err != nil {
		return err
	}
	endTimeParsed, err := time.Parse("15:04", event.EndTime)
	if err != nil {
		return err
	}

	// Convert to pgtype values
	iniDate := pgtype.Date{Time: iniDateTime, Valid: true}
	endDate := pgtype.Date{Time: endDateTime, Valid: true}
	iniTime := pgtype.Time{Microseconds: int64(iniTimeParsed.Hour()*3600+iniTimeParsed.Minute()*60) * 1000000, Valid: true}
	endTime := pgtype.Time{Microseconds: int64(endTimeParsed.Hour()*3600+endTimeParsed.Minute()*60) * 1000000, Valid: true}

	// Convert budget to pgtype.Numeric (assuming 2 decimal places)
	budgetInt := new(big.Int).Mul(big.NewInt(int64(event.Budget*100)), big.NewInt(1))
	budget := pgtype.Numeric{Int: budgetInt, Exp: -2, Valid: true}

	dbEvent := db.UpdateEventParams{
		ID:       id,
		Name:     event.Name,
		IniDate:  iniDate,
		EndDate:  endDate,
		IniTime:  iniTime,
		EndTime:  endTime,
		Location: event.Location,
		Budget:   budget,
		Status:   event.Status,
	}

	_, err = s.querier.UpdateEvent(context.Background(), dbEvent)
	return err
}

// DeleteEvent deletes an event by ID
func (s *eventService) DeleteEvent(eventID string) error {
	if eventID == "" {
		return ErrEventInvalid
	}

	// Parse UUID (same logic as in UpdateEvent)
	var idBytes [16]byte
	if len(eventID) == 36 {
		// Standard UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		if len(eventID) != 36 {
			return ErrEventInvalid
		}
		// Remove hyphens
		hexStr := eventID[0:8] + eventID[9:13] + eventID[14:18] + eventID[19:23] + eventID[24:]
		if len(hexStr) != 32 {
			return ErrEventInvalid
		}
		// Decode hex
		if _, err := hex.Decode(idBytes[:], []byte(hexStr)); err != nil {
			return ErrEventInvalid
		}
	} else if len(eventID) == 32 {
		// Compact UUID format
		if _, err := hex.Decode(idBytes[:], []byte(eventID)); err != nil {
			return ErrEventInvalid
		}
	} else {
		return ErrEventInvalid
	}
	id := pgtype.UUID{Bytes: idBytes, Valid: true}

	return s.querier.DeleteEvent(context.Background(), id)
}

// Helper function to decode hex string
func hexDecode(dst []byte, src []byte) (int, error) {
	if len(src)%2 != 0 {
		return 0, errors.New("hex string length must be even")
	}
	if len(dst) < len(src)/2 {
		return 0, errors.New("buffer too small")
	}
	for i := 0; i < len(src); i += 2 {
		h := hexToByte(src[i])
		if h == 0xff {
			return 0, errors.New("invalid hex digit: " + string(src[i]))
		}
		l := hexToByte(src[i+1])
		if l == 0xff {
			return 0, errors.New("invalid hex digit: " + string(src[i+1]))
		}
		dst[i/2] = h<<4 | l
	}
	return len(src) / 2, nil
}

// Helper function to convert hex character to value
func hexToByte(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0xff
}
