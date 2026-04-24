package service

import (
	"context"
	"fmt"
	"os"
	"testing"

	"conevent-backend/config"
	"conevent-backend/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test database pool using environment variables
func newTestPool(t *testing.T) *pgxpool.Pool {
	// Set environment variables for test database
	// These should be set before calling LoadConfig
	originalValues := map[string]string{
		"DB_HOST":     os.Getenv("DB_HOST"),
		"DB_PORT":     os.Getenv("DB_PORT"),
		"DB_USER":     os.Getenv("DB_USER"),
		"DB_PASSWORD": os.Getenv("DB_PASSWORD"),
		"DB_NAME":     os.Getenv("DB_NAME"),
		"DB_SSLMODE":  os.Getenv("DB_SSLMODE"),
	}

	// Set test database values (adjust as needed for your environment)
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5433") // Use different port to avoid conflict with main db
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "postgres")
	os.Setenv("DB_NAME", "conevent_test")
	os.Setenv("DB_SSLMODE", "disable")

	// Ensure cleanup after test
	t.Cleanup(func() {
		for k, v := range originalValues {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})

	// Load configuration using environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Override database name and port for testing
	cfg.Database.Name = "conevent_test"
	cfg.Database.Port = "5433"

	// Create database connection pool
	dbpool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())
	if err != nil {
		t.Fatalf("Unable to connect to database: %v", err)
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
	schemaData, err := os.ReadFile("/home/fjunior/Development/Projects/Educational/Unifor/conevent-backend/internal/db/schema/001_events.sql")
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

		err := service.CreateEvent(event)
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

		err := service.CreateEvent(event)
		require.NoError(t, err, "CreateEvent should not return an error")
		eventID := event.ID

		// Now retrieve it
		retrievedEvent, err := service.GetEvent(eventID)
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
		err := service.CreateEvent(event1)
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
		err = service.CreateEvent(event2)
		require.NoError(t, err, "CreateEvent should not return an error")

		// List all events
		events, err := service.ListEvents()
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
		err := service.CreateEvent(event)
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

		err = service.UpdateEvent(updatedEvent)
		require.NoError(t, err, "UpdateEvent should not return an error")

		// Retrieve and verify the update
		retrievedEvent, err := service.GetEvent(eventID)
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
		err := service.CreateEvent(event)
		require.NoError(t, err, "CreateEvent should not return an error")
		eventID := event.ID

		// Delete the event
		err = service.DeleteEvent(eventID)
		require.NoError(t, err, "DeleteEvent should not return an error")

		// Verify the event is deleted
		_, err = service.GetEvent(eventID)
		assert.Error(t, err, "GetEvent should return an error for deleted event")
	})
}
