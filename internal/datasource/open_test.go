package datasource

import (
	"context"
	"database/sql"
	"testing"

	"nl2sql/internal/config"

	"github.com/stretchr/testify/require"
)

func TestOpenRegistryFromConfigRegistersConfiguredDatasource(t *testing.T) {
	t.Setenv("MYSQL_RIDE_HAILING_RO_DSN", "readonly-dsn")

	cfg, err := config.LoadFromDir("../config/testdata/basic")
	require.NoError(t, err)

	registry := NewRegistry()
	var gotDriver string
	var gotDSN string

	openFuncStub := func(driverName string, dsn string) (*sql.DB, error) {
		gotDriver = driverName
		gotDSN = dsn
		return &sql.DB{}, nil
	}

	err = OpenAndRegister(context.Background(), registry, cfg.Datasources, openFuncStub)
	require.NoError(t, err)

	db, err := registry.ForDatasource("ride_hailing_ro")
	require.NoError(t, err)
	require.NotNil(t, db)
	require.Equal(t, "mysql", gotDriver)
	require.Equal(t, "readonly-dsn", gotDSN)
}
