package service

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valms/conevent-backend-unifor/internal/db"
)

type serviceFakeDBTX struct {
	event  db.Event
	events []db.Event
	err    error
}

func (f serviceFakeDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.err
}

func (f serviceFakeDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &serviceFakeRows{events: f.events}, nil
}

func (f serviceFakeDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return serviceFakeRow{event: f.event, err: f.err}
}

type serviceFakeRow struct {
	event db.Event
	err   error
}

func (r serviceFakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	assignServiceEvent(dest, r.event)
	return nil
}

type serviceFakeRows struct {
	events []db.Event
	idx    int
}

func (r *serviceFakeRows) Close()                                       {}
func (r *serviceFakeRows) Err() error                                   { return nil }
func (r *serviceFakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *serviceFakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *serviceFakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *serviceFakeRows) RawValues() [][]byte                          { return nil }
func (r *serviceFakeRows) Conn() *pgx.Conn                              { return nil }

func (r *serviceFakeRows) Next() bool {
	if r.idx >= len(r.events) {
		return false
	}
	r.idx++
	return true
}

func (r *serviceFakeRows) Scan(dest ...any) error {
	assignServiceEvent(dest, r.events[r.idx-1])
	return nil
}

func assignServiceEvent(dest []any, event db.Event) {
	*(dest[0].(*pgtype.UUID)) = event.ID
	*(dest[1].(*string)) = event.Name
	*(dest[2].(*pgtype.Date)) = event.IniDate
	*(dest[3].(*pgtype.Date)) = event.EndDate
	*(dest[4].(*pgtype.Time)) = event.IniTime
	*(dest[5].(*pgtype.Time)) = event.EndTime
	*(dest[6].(*string)) = event.Location
	*(dest[7].(*pgtype.Numeric)) = event.Budget
	*(dest[8].(*string)) = event.Status
	*(dest[9].(*pgtype.Timestamptz)) = event.CreatedAt
}

func sampleServiceDBEvent() db.Event {
	return db.Event{
		ID:        pgtype.UUID{Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00}, Valid: true},
		Name:      "Service Event",
		IniDate:   pgtype.Date{Time: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		EndDate:   pgtype.Date{Time: time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC), Valid: true},
		IniTime:   pgtype.Time{Microseconds: 9 * 60 * 60 * 1000000, Valid: true},
		EndTime:   pgtype.Time{Microseconds: 17 * 60 * 60 * 1000000, Valid: true},
		Location:  "Room",
		Budget:    pgtype.Numeric{Int: big.NewInt(12345), Exp: -2, Valid: true},
		Status:    "Confirmado",
		CreatedAt: pgtype.Timestamptz{Time: time.Date(2026, 5, 1, 1, 2, 3, 0, time.UTC), Valid: true},
	}
}

func validServiceEvent() *Event {
	return &Event{Name: "Service Event", IniDate: "2026-05-01", EndDate: "2026-05-02", IniTime: "09:00", EndTime: "17:00", Location: "Room", Budget: 123.45, Status: "Confirmado"}
}

func TestEventServiceDBOperationsSuccess(t *testing.T) {
	dbEvent := sampleServiceDBEvent()
	svc := NewEventService(db.New(serviceFakeDBTX{event: dbEvent, events: []db.Event{dbEvent}}))
	ctx := context.Background()

	found, err := svc.GetEvent(ctx, dbEvent.ID.String())
	require.NoError(t, err)
	assert.Equal(t, "Service Event", found.Name)

	list, err := svc.ListEvents(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	created := validServiceEvent()
	require.NoError(t, svc.CreateEvent(ctx, created))
	assert.Equal(t, dbEvent.ID.String(), created.ID)
	assert.Equal(t, "2026-05-01T01:02:03Z", created.CreatedAt)

	updated := validServiceEvent()
	updated.ID = dbEvent.ID.String()
	require.NoError(t, svc.UpdateEvent(ctx, updated))

	require.NoError(t, svc.DeleteEvent(ctx, dbEvent.ID.String()))
}

func TestEventServiceDBOperationErrors(t *testing.T) {
	wantErr := errors.New("db unavailable")
	svc := NewEventService(db.New(serviceFakeDBTX{err: wantErr}))
	ctx := context.Background()
	id := "550e8400-e29b-41d4-a716-446655440000"

	_, err := svc.GetEvent(ctx, id)
	assert.ErrorIs(t, err, wantErr)
	_, err = svc.ListEvents(ctx)
	assert.ErrorIs(t, err, wantErr)
	assert.ErrorIs(t, svc.CreateEvent(ctx, validServiceEvent()), wantErr)

	updated := validServiceEvent()
	updated.ID = id
	assert.ErrorIs(t, svc.UpdateEvent(ctx, updated), wantErr)
	assert.ErrorIs(t, svc.DeleteEvent(ctx, id), wantErr)
}

func TestEventServiceValidationAndParseErrors(t *testing.T) {
	svc := NewEventService(db.New(serviceFakeDBTX{}))
	ctx := context.Background()

	_, err := svc.GetEvent(ctx, "invalid")
	assert.ErrorIs(t, err, ErrEventInvalid)
	assert.ErrorIs(t, svc.DeleteEvent(ctx, ""), ErrEventInvalid)
	assert.ErrorIs(t, svc.DeleteEvent(ctx, "invalid"), ErrEventInvalid)
	assert.ErrorIs(t, svc.CreateEvent(ctx, &Event{}), ErrEventInvalid)
	createInvalidDate := validServiceEvent()
	createInvalidDate.IniDate = "invalid"
	assert.Error(t, svc.CreateEvent(ctx, createInvalidDate))

	updated := validServiceEvent()
	updated.ID = ""
	assert.ErrorIs(t, svc.UpdateEvent(ctx, updated), ErrEventInvalid)
	updated.ID = "invalid"
	assert.ErrorIs(t, svc.UpdateEvent(ctx, updated), ErrEventInvalid)
	updated.ID = "550e8400-e29b-41d4-a716-446655440000"
	updated.IniDate = "invalid"
	assert.Error(t, svc.UpdateEvent(ctx, updated))
}
