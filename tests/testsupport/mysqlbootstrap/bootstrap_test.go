package mysqlbootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBootstrapCreatesDemoTablesAndSeedsQueryableData(t *testing.T) {
	env := StartMySQLContainer(t)
	defer env.Terminate(t)

	now := time.Date(2026, 4, 27, 10, 0, 0, 0, time.UTC)
	err := Bootstrap(context.Background(), env.RootDB, now)
	require.NoError(t, err)

	var count int
	err = env.RootDB.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM ride_hailing.trip_orders").Scan(&count)
	require.NoError(t, err)
	require.Greater(t, count, 0)
}

