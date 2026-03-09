package marketplace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var sampleIndex = Index{
	Templates: []Template{
		{Name: "langchain-rag", Description: "RAG agent with LangChain", Framework: "langchain", Author: "volra", Tags: []string{"rag", "langchain"}},
		{Name: "crewai-research", Description: "Research crew with CrewAI", Framework: "crewai", Author: "community", Tags: []string{"research", "multi-agent"}},
		{Name: "fastapi-basic", Description: "Basic FastAPI agent", Framework: "generic", Author: "volra", Tags: []string{"fastapi", "basic"}},
	},
}

func TestSearch_MatchesName(t *testing.T) {
	results := Search(&sampleIndex, "langchain")
	require.Len(t, results, 1)
	assert.Equal(t, "langchain-rag", results[0].Name)
}

func TestSearch_MatchesDescription(t *testing.T) {
	results := Search(&sampleIndex, "research")
	require.Len(t, results, 1)
	assert.Equal(t, "crewai-research", results[0].Name)
}

func TestSearch_MatchesTags(t *testing.T) {
	results := Search(&sampleIndex, "rag")
	require.Len(t, results, 1)
	assert.Equal(t, "langchain-rag", results[0].Name)
}

func TestSearch_CaseInsensitive(t *testing.T) {
	results := Search(&sampleIndex, "RAG")
	require.Len(t, results, 1)
}

func TestSearch_EmptyQueryReturnsAll(t *testing.T) {
	results := Search(&sampleIndex, "")
	assert.Len(t, results, 3)
}

func TestSearch_NoMatchReturnsNil(t *testing.T) {
	results := Search(&sampleIndex, "nonexistent")
	assert.Nil(t, results)
}

func TestLookup_ExactMatch(t *testing.T) {
	tmpl, ok := Lookup(&sampleIndex, "langchain-rag")
	require.True(t, ok)
	assert.Equal(t, "langchain-rag", tmpl.Name)
}

func TestLookup_CaseInsensitive(t *testing.T) {
	tmpl, ok := Lookup(&sampleIndex, "LANGCHAIN-RAG")
	require.True(t, ok)
	assert.Equal(t, "langchain-rag", tmpl.Name)
}

func TestLookup_NotFound(t *testing.T) {
	_, ok := Lookup(&sampleIndex, "nonexistent")
	assert.False(t, ok)
}

func TestFetchIndex_RemoteFetch(t *testing.T) {
	data, _ := json.Marshal(sampleIndex)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	client := &Client{
		IndexURL:   srv.URL,
		CacheDir:   t.TempDir(),
		HTTPClient: srv.Client(),
	}

	idx, err := client.FetchIndex()
	require.NoError(t, err)
	assert.Len(t, idx.Templates, 3)
}

func TestFetchIndex_UsesCache(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(sampleIndex)

	// Write fresh cache.
	cacheDir := dir
	require.NoError(t, os.WriteFile(filepath.Join(cacheDir, "marketplace-cache.json"), data, 0o644))

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Write(data)
	}))
	defer srv.Close()

	client := &Client{
		IndexURL:   srv.URL,
		CacheDir:   cacheDir,
		HTTPClient: srv.Client(),
	}

	idx, err := client.FetchIndex()
	require.NoError(t, err)
	assert.Len(t, idx.Templates, 3)
	assert.Equal(t, 0, callCount, "should use cache, not fetch remotely")
}

func TestFetchIndex_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := &Client{
		IndexURL:   srv.URL,
		CacheDir:   t.TempDir(),
		HTTPClient: srv.Client(),
	}

	_, err := client.FetchIndex()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestIsEmpty_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	empty, err := IsEmpty(dir)
	require.NoError(t, err)
	assert.True(t, empty)
}

func TestIsEmpty_NonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0o644))

	empty, err := IsEmpty(dir)
	require.NoError(t, err)
	assert.False(t, empty)
}

func TestIsEmpty_NonExistentDir(t *testing.T) {
	empty, err := IsEmpty("/tmp/nonexistent-volra-test-dir")
	require.NoError(t, err)
	assert.True(t, empty)
}
