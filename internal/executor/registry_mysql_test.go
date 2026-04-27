package executor

import (
	"context"
	"database/sql"
	"testing"

	"nl2sql/internal/formatter"

	"github.com/stretchr/testify/require"
)

func TestRegistryExecutorUsesDatasourceIDFromResolvedPlan(t *testing.T) {
	t.Parallel()

	expectedDB := &sql.DB{}
	registry := &stubRegistry{db: expectedDB}
	exec := NewRegistryExecutor(registry)
	exec.run = func(_ context.Context, db *sql.DB, query string, args []any) (formatter.QueryResult, error) {
		require.Same(t, expectedDB, db)
		require.Equal(t, "SELECT 1", query)
		require.Nil(t, args)
		return formatter.QueryResult{}, nil
	}

	_, err := exec.Query(context.Background(), "ride_hailing_ro", "SELECT 1", nil)
	require.NoError(t, err)
	require.Equal(t, "ride_hailing_ro", registry.requestedID)
}

type stubRegistry struct {
	requestedID string
	db          *sql.DB
	err         error
}

func (s *stubRegistry) ForDatasource(datasourceID string) (*sql.DB, error) {
	s.requestedID = datasourceID
	if s.err != nil {
		return nil, s.err
	}

	return s.db, nil
}

