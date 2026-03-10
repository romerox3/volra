package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/romerox3/volra/internal/a2a"
	"github.com/romerox3/volra/internal/controlplane"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchCapabilities_Success(t *testing.T) {
	caps := []controlplane.FederatedCapability{
		{
			Server: "staging",
			Agent:  "analyst",
			URL:    "http://staging:8000",
			Status: "ok",
			Card: &a2a.AgentCard{
				Name: "analyst",
				Skills: []a2a.Skill{
					{ID: "summarize", Name: "summarize", Description: "Summarize text"},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/federation/capabilities", r.URL.Path)
		json.NewEncoder(w).Encode(caps)
	}))
	defer srv.Close()

	result, err := fetchCapabilities(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "staging", result[0].Server)
	assert.Equal(t, "analyst", result[0].Agent)
}

func TestLoadFederatedTools_Success(t *testing.T) {
	caps := []controlplane.FederatedCapability{
		{
			Server: "staging",
			Agent:  "analyst",
			URL:    "http://staging:8000",
			Status: "ok",
			Card: &a2a.AgentCard{
				Name: "analyst",
				Skills: []a2a.Skill{
					{ID: "summarize", Name: "summarize", Description: "Summarize text"},
					{ID: "classify", Name: "classify", Description: "Classify text"},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(caps)
	}))
	defer srv.Close()

	tools := LoadFederatedTools(context.Background(), srv.URL)
	require.Len(t, tools, 2)
	assert.Equal(t, "staging/analyst/summarize", tools[0].Tool.Name)
	assert.Equal(t, "staging/analyst/classify", tools[1].Tool.Name)
	assert.True(t, tools[0].Remote)
}

func TestLoadFederatedTools_Unavailable(t *testing.T) {
	tools := LoadFederatedTools(context.Background(), "http://localhost:1")
	assert.Nil(t, tools)
}

func TestLoadFederatedTools_EmptyURL(t *testing.T) {
	tools := LoadFederatedTools(context.Background(), "")
	assert.Nil(t, tools)
}

func TestLoadFederatedTools_SkipsLocalAndErrorAgents(t *testing.T) {
	caps := []controlplane.FederatedCapability{
		{Server: "local", Agent: "my-agent", Status: "ok", Card: &a2a.AgentCard{
			Skills: []a2a.Skill{{ID: "tool1"}},
		}},
		{Server: "staging", Agent: "broken", Status: "card_error"},
		{Server: "staging", Agent: "good", Status: "ok", URL: "http://staging:8000", Card: &a2a.AgentCard{
			Skills: []a2a.Skill{{ID: "tool2", Description: "Good tool"}},
		}},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(caps)
	}))
	defer srv.Close()

	tools := LoadFederatedTools(context.Background(), srv.URL)
	require.Len(t, tools, 1)
	assert.Equal(t, "staging/good/tool2", tools[0].Tool.Name)
}

func TestGetRefreshInterval_Default(t *testing.T) {
	interval := GetRefreshInterval()
	assert.Equal(t, DefaultRefreshInterval, interval)
}

func TestRefreshFederatedTools(t *testing.T) {
	caps := []controlplane.FederatedCapability{
		{
			Server: "prod",
			Agent:  "reporter",
			URL:    "http://prod:8000",
			Status: "ok",
			Card: &a2a.AgentCard{
				Name:   "reporter",
				Skills: []a2a.Skill{{ID: "report", Description: "Generate report"}},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(caps)
	}))
	defer srv.Close()

	cat := &Catalog{}
	refreshFederatedTools(context.Background(), cat, srv.URL)

	assert.Equal(t, 1, cat.RemoteToolCount())
	tools := cat.Tools()
	assert.Equal(t, "prod/reporter/report", tools[0].Tool.Name)
}
