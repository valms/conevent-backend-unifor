package service

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valms/conevent-backend-unifor/config"
	"github.com/valms/conevent-backend-unifor/internal/db"
)

// Helper function to create a test database pool using environment variables
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "postgres")
	t.Setenv("DB_NAME", "conevent_test")
	t.Setenv("DB_SSLMODE", "disable")

	// Load configuration using environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create database connection pool
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
	}
	if err := dbpool.Ping(context.Background()); err != nil {
		dbpool.Close()
		t.Skipf("Skipping integration test; test database is unavailable: %v", err)
	}

	// Run migrations to create tables
	if err := runMigrations(dbpool); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return dbpool
}

// runMigrations runs the SQL migrations to create the necessary tables
func runMigrations(dbpool *pgxpool.Pool) error {
	// First, drop existing tables in reverse order to avoid foreign key constraints
	dropStatements := `
	DROP TABLE IF EXISTS guests;
	DROP TABLE IF EXISTS budget_items;
	DROP TABLE IF EXISTS suppliers;
	DROP TABLE IF EXISTS events;
	`

	_, err := dbpool.Exec(context.Background(), dropStatements)
	if err != nil {
		return fmt.Errorf("failed to drop existing tables: %w", err)
	}

	// Read the schema file
	schemaPath := filepath.Join("..", "db", "schema", "001_events.sql")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Execute the SQL statements
	_, err = dbpool.Exec(context.Background(), string(schemaData))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// truncateEvents clears all events from the database
func truncateEvents(dbpool *pgxpool.Pool) error {
	_, err := dbpool.Exec(context.Background(), "TRUNCATE TABLE events RESTART IDENTITY CASCADE")
	if err != nil {
		return fmt.Errorf("failed to truncate events table: %w", err)
	}
	return nil
}

func TestEventService_CRUD(t *testing.T) {
	dbpool := newTestPool(t)
	defer dbpool.Close()

	querier := db.New(dbpool)
	service := NewEventService(querier)

	// Test CreateEvent
	t.Run("CreateEvent", func(t *testing.T) {
		event := &Event{
			Name:     "Test Event",
			IniDate:  "2026-05-01",
			EndDate:  "2026-05-02",
			IniTime:  "09:00",
			EndTime:  "17:00",
			Location: "Test Location",
			Budget:   1000.50,
			Status:   "Planejamento",
		}

		err := service.CreateEvent(context.Background(), event)
		require.NoError(t, err, "CreateEvent should not return an error")
		assert.NotEmpty(t, event.ID, "Event ID should be set after creation")
		assert.NotZero(t, event.CreatedAt, "CreatedAt should be set after creation")
	})

	// Test GetEvent
	t.Run("GetEvent", func(t *testing.T) {
		// First create an event to retrieve
		event := &Event{
			Name:     "Test Event 2",
			IniDate:  "2026-05-03",
			EndDate:  "2026-05-04",
			IniTime:  "10:00",
			EndTime:  "18:00",
			Location: "Test Location 2",
			Budget:   2000.75,
			Status:   "Confirmado",
		}

		err := service.CreateEvent(context.Background(), event)
		require.NoError(t, err, "CreateEvent should not return an error")
		eventID := event.ID

		// Now retrieve it
		retrievedEvent, err := service.GetEvent(context.Background(), eventID)
		require.NoError(t, err, "GetEvent should not return an error")
		assert.NotNil(t, retrievedEvent, "Retrieved event should not be nil")
		assert.Equal(t, event.Name, retrievedEvent.Name, "Event names should match")
		assert.Equal(t, event.IniDate, retrievedEvent.IniDate, "Initial dates should match")
		assert.Equal(t, event.EndDate, retrievedEvent.EndDate, "End dates should match")
		assert.Equal(t, event.IniTime, retrievedEvent.IniTime, "Initial times should match")
		assert.Equal(t, event.EndTime, retrievedEvent.EndTime, "End times should match")
		assert.Equal(t, event.Location, retrievedEvent.Location, "Locations should match")
		assert.Equal(t, event.Budget, retrievedEvent.Budget, "Budgets should match")
		assert.Equal(t, event.Status, retrievedEvent.Status, "Statuses should match")
	})

	// Test ListEvents
	t.Run("ListEvents", func(t *testing.T) {
		// Clear any existing events
		if err := truncateEvents(dbpool); err != nil {
			t.Fatalf("Failed to truncate events: %v", err)
		}

		// Create a couple of events
		event1 := &Event{
			Name:     "Test Event 3",
			IniDate:  "2026-05-05",
			EndDate:  "2026-05-06",
			IniTime:  "11:00",
			EndTime:  "19:00",
			Location: "Test Location 3",
			Budget:   3000.00,
			Status:   "Concluído",
		}
		err := service.CreateEvent(context.Background(), event1)
		require.NoError(t, err, "CreateEvent should not return an error")

		event2 := &Event{
			Name:     "Test Event 4",
			IniDate:  "2026-05-07",
			EndDate:  "2026-05-08",
			IniTime:  "12:00",
			EndTime:  "20:00",
			Location: "Test Location 4",
			Budget:   4000.25,
			Status:   "Cancelado",
		}
		err = service.CreateEvent(context.Background(), event2)
		require.NoError(t, err, "CreateEvent should not return an error")

		// List all events
		events, err := service.ListEvents(context.Background())
		require.NoError(t, err, "ListEvents should not return an error")
		assert.Len(t, events, 2, "Should have 2 events")
		assert.Contains(t, events[0].Name, "Test Event", "First event should contain 'Test Event'")
		assert.Contains(t, events[1].Name, "Test Event", "Second event should contain 'Test Event'")
	})

	// Test UpdateEvent
	t.Run("UpdateEvent", func(t *testing.T) {
		// Create an event to update
		event := &Event{
			Name:     "Original Event",
			IniDate:  "2026-05-09",
			EndDate:  "2026-05-10",
			IniTime:  "13:00",
			EndTime:  "21:00",
			Location: "Original Location",
			Budget:   5000.00,
			Status:   "Planejamento",
		}
		err := service.CreateEvent(context.Background(), event)
		require.NoError(t, err, "CreateEvent should not return an error")
		eventID := event.ID

		// Update the event
		updatedEvent := &Event{
			ID:       eventID,
			Name:     "Updated Event",
			IniDate:  "2026-05-11",
			EndDate:  "2026-05-12",
			IniTime:  "14:00",
			EndTime:  "22:00",
			Location: "Updated Location",
			Budget:   6000.50,
			Status:   "Confirmado",
		}

		err = service.UpdateEvent(context.Background(), updatedEvent)
		require.NoError(t, err, "UpdateEvent should not return an error")

		// Retrieve and verify the update
		retrievedEvent, err := service.GetEvent(context.Background(), eventID)
		require.NoError(t, err, "GetEvent should not return an error")
		assert.Equal(t, updatedEvent.Name, retrievedEvent.Name, "Event names should match after update")
		assert.Equal(t, updatedEvent.IniDate, retrievedEvent.IniDate, "Initial dates should match after update")
		assert.Equal(t, updatedEvent.EndDate, retrievedEvent.EndDate, "End dates should match after update")
		assert.Equal(t, updatedEvent.IniTime, retrievedEvent.IniTime, "Initial times should match after update")
		assert.Equal(t, updatedEvent.EndTime, retrievedEvent.EndTime, "End times should match after update")
		assert.Equal(t, updatedEvent.Location, retrievedEvent.Location, "Locations should match after update")
		assert.Equal(t, updatedEvent.Budget, retrievedEvent.Budget, "Budgets should match after update")
		assert.Equal(t, updatedEvent.Status, retrievedEvent.Status, "Statuses should match after update")
	})

	// Test DeleteEvent
	t.Run("DeleteEvent", func(t *testing.T) {
		// Create an event to delete
		event := &Event{
			Name:     "Event to Delete",
			IniDate:  "2026-05-13",
			EndDate:  "2026-05-14",
			IniTime:  "15:00",
			EndTime:  "23:00",
			Location: "Location to Delete",
			Budget:   7000.75,
			Status:   "Cancelado",
		}
		err := service.CreateEvent(context.Background(), event)
		require.NoError(t, err, "CreateEvent should not return an error")
		eventID := event.ID

		// Delete the event
		err = service.DeleteEvent(context.Background(), eventID)
		require.NoError(t, err, "DeleteEvent should not return an error")

		// Verify the event is deleted
		_, err = service.GetEvent(context.Background(), eventID)
		assert.Error(t, err, "GetEvent should return an error for deleted event")
	})
}

// TestUpdateEvent_Error validates error handling in UpdateEvent
func TestUpdateEvent_Error(t *testing.T) {
	service := NewEventService(nil)

	// Test with empty ID
	event := &Event{
		ID:       "",
		Name:     "Test Event",
		IniDate:  "2026-05-01",
		EndDate:  "2026-05-02",
		IniTime:  "09:00",
		EndTime:  "17:00",
		Location: "Test Location",
		Budget:   1000.50,
		Status:   "Planejamento",
	}

	err := service.UpdateEvent(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventInvalid)

	// Test with invalid UUID format
	event.ID = "invalid-uuid"
	err = service.UpdateEvent(context.Background(), event)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventInvalid)
}

// TestParseUUID_Error tests UUID parsing edge cases
func TestParseUUID_Error(t *testing.T) {
	service := &eventService{}

	// Test empty string
	_, err := service.ParseUUID("")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventInvalid)

	// Test too short
	_, err = service.ParseUUID("123")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventInvalid)

	// Test invalid hex
	_, err = service.ParseUUID("zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEventInvalid)

	// Test valid UUID formats
	id1, err := service.ParseUUID("550e8400-e29b-41d4-a716-446655440000")
	require.NoError(t, err)
	assert.True(t, id1.Valid)

	id2, err := service.ParseUUID("550e8400e29b41d4a716446655440000")
	require.NoError(t, err)
	assert.True(t, id2.Valid)

	// Both should produce the same UUID
	assert.Equal(t, id1, id2)
}

// TestFormatDate tests date formatting
func TestFormatDate(t *testing.T) {
	service := &eventService{}

	// Test valid date
	date, err := service.ParseDate("2026-05-15")
	require.NoError(t, err)
	assert.True(t, date.Valid)
	assert.Equal(t, "2026-05-15", service.FormatDate(date))

	// Test empty string
	date, err = service.ParseDate("")
	require.NoError(t, err)
	assert.False(t, date.Valid)
	assert.Equal(t, "", service.FormatDate(date))
}

// TestFormatTime tests time formatting
func TestFormatTime(t *testing.T) {
	service := &eventService{}

	// Test valid time
	timeVal, err := service.ParseTime("14:30")
	require.NoError(t, err)
	assert.True(t, timeVal.Valid)
	assert.Equal(t, "14:30", service.FormatTime(timeVal))

	// Test empty string
	timeVal, err = service.ParseTime("")
	require.NoError(t, err)
	assert.False(t, timeVal.Valid)
	assert.Equal(t, "", service.FormatTime(timeVal))
}

// TestConvertBudget tests budget conversion
func TestConvertBudget(t *testing.T) {
	service := &eventService{}

	// Test zero budget
	assert.Equal(t, 0.0, service.convertBudget(pgtype.Numeric{}))

	// Test positive budget
	var budget pgtype.Numeric
	budget.Int = big.NewInt(12345)
	budget.Exp = -2
	budget.Valid = true
	assert.Equal(t, 123.45, service.convertBudget(budget))

	// Test negative budget
	budget.Int = big.NewInt(-9876)
	budget.Exp = -2
	budget.Valid = true
	assert.Equal(t, -98.76, service.convertBudget(budget))

	budget.Int = nil
	budget.Valid = true
	assert.Equal(t, 0.0, service.convertBudget(budget))
}

func TestEventConversionHelpers(t *testing.T) {
	service := &eventService{}
	event := &Event{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Name:     "Conversion Test",
		IniDate:  "2026-06-01",
		EndDate:  "2026-06-02",
		IniTime:  "08:30",
		EndTime:  "17:45",
		Location: "Auditório",
		Budget:   10.129,
		Status:   "Planejamento",
	}

	createParams, err := service.convertToDBEvent(event)
	require.NoError(t, err)
	assert.Equal(t, event.Name, createParams.Name)
	assert.Equal(t, int64(1013), createParams.Budget.Int.Int64())
	assert.Equal(t, int32(-2), createParams.Budget.Exp)

	id, err := service.ParseUUID(event.ID)
	require.NoError(t, err)
	updateParams, err := service.convertToDBEventForUpdate(event, id)
	require.NoError(t, err)
	assert.Equal(t, id, updateParams.ID)
	assert.Equal(t, event.Location, updateParams.Location)
}

func TestEventConversionErrors(t *testing.T) {
	service := &eventService{}
	base := Event{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Name:     "Conversion Test",
		IniDate:  "2026-06-01",
		EndDate:  "2026-06-02",
		IniTime:  "08:30",
		EndTime:  "17:45",
		Location: "Auditório",
		Budget:   10,
		Status:   "Planejamento",
	}

	tests := []struct {
		name   string
		mutate func(*Event)
	}{
		{"invalid ini date", func(e *Event) { e.IniDate = "invalid" }},
		{"invalid end date", func(e *Event) { e.EndDate = "invalid" }},
		{"invalid ini time", func(e *Event) { e.IniTime = "99:99" }},
		{"invalid end time", func(e *Event) { e.EndTime = "99:99" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := base
			tt.mutate(&event)
			_, err := service.convertToDBEvent(&event)
			require.Error(t, err)

			id, parseErr := service.ParseUUID(base.ID)
			require.NoError(t, parseErr)
			_, err = service.convertToDBEventForUpdate(&event, id)
			require.Error(t, err)
		})
	}
}

func TestValidateEvent(t *testing.T) {
	service := &eventService{}
	valid := &Event{Name: "Name", IniDate: "2026-01-01", EndDate: "2026-01-02", IniTime: "09:00", EndTime: "10:00", Location: "Room", Status: "Confirmado"}
	assert.NoError(t, service.validateEvent(valid))

	invalid := *valid
	invalid.Name = ""
	assert.ErrorIs(t, service.validateEvent(&invalid), ErrEventInvalid)
}

func TestFormatTimestampAndCopyCreatedEvent(t *testing.T) {
	service := &eventService{}
	assert.Equal(t, "", service.formatTimeStamp(pgtype.Timestamptz{}))

	createdAt := pgtype.Timestamptz{Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Valid: true}
	assert.Equal(t, "2026-01-02T03:04:05Z", service.formatTimeStamp(createdAt))

	iniDate, _ := service.ParseDate("2026-01-01")
	endDate, _ := service.ParseDate("2026-01-02")
	iniTime, _ := service.ParseTime("09:00")
	endTime, _ := service.ParseTime("10:30")
	budget, _ := service.convertBudgetToNumeric(123.45)
	event := &Event{}
	service.copyCreatedEvent(event, db.Event{
		IniDate:   iniDate,
		EndDate:   endDate,
		IniTime:   iniTime,
		EndTime:   endTime,
		Location:  "Room",
		Budget:    budget,
		Status:    "Confirmado",
		CreatedAt: createdAt,
	})

	assert.Equal(t, "2026-01-01", event.IniDate)
	assert.Equal(t, "10:30", event.EndTime)
	assert.Equal(t, "Room", event.Location)
	assert.Equal(t, 123.45, event.Budget)
	assert.Equal(t, "Confirmado", event.Status)
	assert.Equal(t, "2026-01-02T03:04:05Z", event.CreatedAt)
}
