package eval

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromClient_Query_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/query", r.URL.Path)
		assert.Contains(t, r.URL.Query().Get("query"), "up")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{"metric": {"__name__": "up"}, "value": [1710000000, "0.998"]}
				]
			}
		}`))
	}))
	defer srv.Close()

	client := &PromClient{BaseURL: srv.URL}
	val, err := client.Query(context.Background(), "up{job='agent'}")

	require.NoError(t, err)
	assert.InDelta(t, 0.998, val, 0.001)
}

func TestPromClient_Query_NoData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "success",
			"data": {"resultType": "vector", "result": []}
		}`))
	}))
	defer srv.Close()

	client := &PromClient{BaseURL: srv.URL}
	_, err := client.Query(context.Background(), "nonexistent_metric")

	require.Error(t, err)
	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodePromQLFailed, ue.Code)
	assert.Contains(t, ue.What, "no data")
}

func TestPromClient_Query_PromQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": "error",
			"errorType": "bad_data",
			"error": "parse error at char 5"
		}`))
	}))
	defer srv.Close()

	client := &PromClient{BaseURL: srv.URL}
	_, err := client.Query(context.Background(), "bad{query")

	require.Error(t, err)
	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodePromQLFailed, ue.Code)
	assert.Contains(t, ue.What, "parse error")
}

func TestPromClient_Query_Unreachable(t *testing.T) {
	client := &PromClient{BaseURL: "http://localhost:1"}
	_, err := client.Query(context.Background(), "up")

	require.Error(t, err)
	var ue *output.UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, output.CodePrometheusUnreachable, ue.Code)
}
