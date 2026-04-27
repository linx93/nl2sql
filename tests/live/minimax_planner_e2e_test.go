package live_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"nl2sql/internal/planner"
)

func TestMiniMaxPlannerReturnsValidRankingRawPlan(t *testing.T) {
	client := newLivePlannerFromEnv(t)

	plan, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.NoError(t, err)
	require.Equal(t, "ranking", plan.QueryMode)
	require.Contains(t, plan.Metrics, "取消率")
	require.Contains(t, plan.Dimensions, "城市")
}

func newLivePlannerFromEnv(t *testing.T) planner.MiniMaxPlanner {
	t.Helper()

	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Fatal("MINIMAX_API_KEY is required")
	}

	return planner.NewMiniMaxPlanner(planner.MiniMaxConfig{
		BaseURL: os.Getenv("MINIMAX_BASE_URL"),
		APIKey:  apiKey,
		Model:   os.Getenv("MINIMAX_MODEL"),
	})
}
