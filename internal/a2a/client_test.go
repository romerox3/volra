package a2a

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCard_Success(t *testing.T) {
	card := &AgentCard{
		Name:            "remote-agent",
		URL:             "http://remote:8000",
		Version:         "1.0.0",
		DocumentVersion: "0.3.0",
	}
	data, _ := json.Marshal(card)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/agent-card.json", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer srv.Close()

	result, err := FetchCard(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, "remote-agent", result.Name)
	assert.Equal(t, "0.3.0", result.DocumentVersion)
}

func TestFetchCard_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := FetchCard(context.Background(), srv.URL)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeA2ACardFetchFailed, ue.Code)
}

func TestFetchCard_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := FetchCard(context.Background(), srv.URL)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeA2ACardInvalid, ue.Code)
}

func TestFetchCard_Timeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancelled

	_, err := FetchCard(ctx, "http://192.0.2.1:9999") // non-routable
	require.Error(t, err)
}

func TestFetchCards_MixedResults(t *testing.T) {
	// Successful server
	card := &AgentCard{Name: "good-agent", DocumentVersion: "0.3.0"}
	data, _ := json.Marshal(card)
	goodSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(data)
	}))
	defer goodSrv.Close()

	// Bad server (invalid JSON)
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("{invalid"))
	}))
	defer badSrv.Close()

	results := FetchCards(context.Background(), []string{goodSrv.URL, badSrv.URL})

	require.Len(t, results, 2)

	// First: success
	assert.NoError(t, results[0].Error)
	assert.Equal(t, "good-agent", results[0].Card.Name)

	// Second: error
	assert.Error(t, results[1].Error)
	assert.Nil(t, results[1].Card)
}

func TestFetchCards_AllSuccess(t *testing.T) {
	card1 := &AgentCard{Name: "agent-1"}
	card2 := &AgentCard{Name: "agent-2"}
	data1, _ := json.Marshal(card1)
	data2, _ := json.Marshal(card2)

	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(data1)
	}))
	defer srv1.Close()

	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(data2)
	}))
	defer srv2.Close()

	results := FetchCards(context.Background(), []string{srv1.URL, srv2.URL})

	require.Len(t, results, 2)
	assert.Equal(t, "agent-1", results[0].Card.Name)
	assert.Equal(t, "agent-2", results[1].Card.Name)
}

func TestFetchCards_Empty(t *testing.T) {
	results := FetchCards(context.Background(), []string{})
	assert.Empty(t, results)
}
