package planner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestNewMiniMaxPlannerDefaultsToOfficialBaseURL(t *testing.T) {
	t.Parallel()

	client := NewMiniMaxPlanner(MiniMaxConfig{
		APIKey: "test-key",
	})

	require.Equal(t, "https://api.minimaxi.com", client.baseURL)
	require.Equal(t, "MiniMax-M2.7", client.model)
}

func TestMiniMaxPlannerReturnsUpstreamErrorBodyForNon2xxResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"insufficient balance (1008)"}}`))
	}))
	defer server.Close()

	client := NewMiniMaxPlanner(MiniMaxConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "MiniMax-M2.7-highspeed",
	})

	_, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "insufficient balance (1008)")
}

func TestMiniMaxPlannerParsesJSONWrappedInMarkdownFence(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{
					"type": "thinking",
				},
				{
					"type": "text",
					"text": "```json\n{\"query_mode\":\"ranking\",\"metrics\":[\"取消率\"],\"dimensions\":[\"城市\"],\"limit\":10}\n```",
				},
			},
		}))
	}))
	defer server.Close()

	client := NewMiniMaxPlanner(MiniMaxConfig{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "MiniMax-M2.7",
	})

	plan, err := client.Plan(context.Background(), "最近30天取消率最高的城市", "ride_hailing")
	require.NoError(t, err)
	require.Equal(t, "ranking", plan.QueryMode)
	require.Equal(t, []string{"取消率"}, plan.Metrics)
	require.Equal(t, []string{"城市"}, plan.Dimensions)
}

func TestBuildMiniMaxSystemPromptDefinesRawPlanContractAndExamples(t *testing.T) {
	t.Parallel()

	prompt := buildMiniMaxSystemPrompt()

	require.True(t, strings.Contains(prompt, `"query_mode"`))
	require.True(t, strings.Contains(prompt, `"aggregate_overview"`))
	require.True(t, strings.Contains(prompt, `"ranking"`))
	require.True(t, strings.Contains(prompt, `"trend"`))
	require.True(t, strings.Contains(prompt, `"detail_list"`))
	require.True(t, strings.Contains(prompt, "Do not wrap the JSON in markdown code fences"))
	require.True(t, strings.Contains(prompt, "最近30天取消率最高的城市"))
	require.True(t, strings.Contains(prompt, `"metrics":["取消率"]`))
	require.True(t, strings.Contains(prompt, `"dimensions":["城市"]`))
}
