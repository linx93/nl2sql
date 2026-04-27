package planner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMiniMaxPlannerBuildsAnthropicRequestAndParsesRawPlanJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()

		require.Equal(t, "/anthropic/v1/messages", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "MiniMax-M2.7-highspeed", payload["model"])

		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": `{"query_mode":"ranking","metrics":["取消率"],"dimensions":["城市"],"limit":10}`,
				},
			},
		}))
	}))
	defer server.Close()

	client := NewMiniMaxPlanner(MiniMaxConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "MiniMax-M2.7-highspeed",
	})

	plan, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.NoError(t, err)
	require.Equal(t, "ranking", plan.QueryMode)
	require.Equal(t, []string{"取消率"}, plan.Metrics)
	require.Equal(t, []string{"城市"}, plan.Dimensions)
	require.Equal(t, 10, plan.Limit)
}

