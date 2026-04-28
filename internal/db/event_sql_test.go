package db

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
)

type fakeDBTX struct {
	event   Event
	events  []Event
	err     error
	rowsErr error
	scanErr error
}

func (f fakeDBTX) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, f.err
}

func (f fakeDBTX) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeRows{events: f.events, err: f.rowsErr, scanErr: f.scanErr}, nil
}

func (f fakeDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return fakeRow{event: f.event, err: f.err}
}

type fakeRow struct {
	event Event
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	assignEvent(dest, r.event)
	return nil
}

type fakeRows struct {
	events  []Event
	idx     int
	err     error
	scanErr error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeRows) Next() bool {
	if r.idx >= len(r.events) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	assignEvent(dest, r.events[r.idx-1])
	return nil
}

func assignEvent(dest []any, event Event) {
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

func sampleDBEvent() Event {
	return Event{
		ID:        pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		Name:      "Sample",
		IniDate:   pgtype.Date{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		EndDate:   pgtype.Date{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Valid: true},
		IniTime:   pgtype.Time{Microseconds: 9 * 60 * 60 * 1000000, Valid: true},
		EndTime:   pgtype.Time{Microseconds: 10 * 60 * 60 * 1000000, Valid: true},
		Location:  "Room",
		Budget:    pgtype.Numeric{Int: bigInt(1000), Exp: -2, Valid: true},
		Status:    "Confirmado",
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
}

func bigInt(value int64) *big.Int { return big.NewInt(value) }

func TestGeneratedEventQueries(t *testing.T) {
	event := sampleDBEvent()
	queries := New(fakeDBTX{event: event, events: []Event{event}})
	ctx := context.Background()

	created, err := queries.CreateEvent(ctx, CreateEventParams{Name: "Sample"})
	require.NoError(t, err)
	assert.Equal(t, event.Name, created.Name)

	found, err := queries.GetEvent(ctx, event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.Name, found.Name)

	updated, err := queries.UpdateEvent(ctx, UpdateEventParams{ID: event.ID, Name: "Sample"})
	require.NoError(t, err)
	assert.Equal(t, event.Name, updated.Name)

	list, err := queries.ListEvents(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, queries.DeleteEvent(ctx, event.ID))
}

func TestGeneratedEventQueryErrors(t *testing.T) {
	wantErr := errors.New("db error")
	queries := New(fakeDBTX{err: wantErr})
	ctx := context.Background()

	_, err := queries.CreateEvent(ctx, CreateEventParams{})
	assert.ErrorIs(t, err, wantErr)
	_, err = queries.GetEvent(ctx, pgtype.UUID{})
	assert.ErrorIs(t, err, wantErr)
	_, err = queries.UpdateEvent(ctx, UpdateEventParams{})
	assert.ErrorIs(t, err, wantErr)
	_, err = queries.ListEvents(ctx)
	assert.ErrorIs(t, err, wantErr)
	assert.ErrorIs(t, queries.DeleteEvent(ctx, pgtype.UUID{}), wantErr)

	queries = New(fakeDBTX{events: []Event{sampleDBEvent()}, scanErr: wantErr})
	_, err = queries.ListEvents(ctx)
	assert.ErrorIs(t, err, wantErr)

	queries = New(fakeDBTX{events: []Event{sampleDBEvent()}, rowsErr: wantErr})
	_, err = queries.ListEvents(ctx)
	assert.ErrorIs(t, err, wantErr)
}
