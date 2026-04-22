package service

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
	// ID is the unique identifier for the event
	ID string
	// Title is the name of the event
	Title string
	// Description provides details about the event
	Description string
	// StartTime is when the event begins (in RFC3339 format)
	StartTime string
	// EndTime is when the event ends (in RFC3339 format)
	EndTime string
	// Location is where the event takes place
	Location string
}
