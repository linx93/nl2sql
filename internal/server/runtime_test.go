package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildRuntimeWiresPlannerExecutorAndAudit(t *testing.T) {
	t.Setenv("MYSQL_RIDE_HAILING_RO_DSN", "readonly-dsn")

	runtime, err := BuildRuntime(RuntimeConfig{
		ConfigDir:      filepath.Join("..", "..", "configs"),
		MiniMaxAPIKey:  "test-key",
		MiniMaxBaseURL: "http://127.0.0.1",
		AuditDSN:       "audit-dsn",
		OpenDatasource: func(driverName string, dsn string) (*sql.DB, error) {
			return &sql.DB{}, nil
		},
		OpenAuditDB: func(driverName string, dsn string) (*sql.DB, error) {
			return &sql.DB{}, nil
		},
	})
	require.NoError(t, err)
	require.NotNil(t, runtime.Service)
	require.NotNil(t, runtime.Mux)
}

