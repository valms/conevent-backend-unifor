package db

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestNewAndWithTx(t *testing.T) {
	queries := New(nil)
	require.NotNil(t, queries)

	var tx pgx.Tx
	txQueries := queries.WithTx(tx)
	require.NotNil(t, txQueries)
}
