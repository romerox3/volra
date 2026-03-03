package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/antonioromero/volra/internal/output"
	"github.com/antonioromero/volra/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForHealth_ImmediateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	p := &testutil.MockPresenter{}

	err := WaitForHealth(context.Background(), port, "/", "test", p)
	require.NoError(t, err)
	assert.NotEmpty(t, p.ProgressCalls)
}

func TestWaitForHealth_EventualSuccess(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	p := &testutil.MockPresenter{}

	err := WaitForHealth(context.Background(), port, "/", "test", p)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, attempts, 3)
}

func TestWaitForHealth_Timeout(t *testing.T) {
	// Server that never returns 200
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	p := &testutil.MockPresenter{}

	// Use a short context deadline to avoid waiting 60s
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := WaitForHealth(ctx, port, "/health", "my-agent", p)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeHealthCheckFailed, ue.Code)
	assert.Contains(t, ue.What, "timed out")
	assert.Contains(t, ue.Fix, "docker logs my-agent-agent")
}

func TestWaitForHealth_ConnectionRefused(t *testing.T) {
	// Use a port that's not listening
	p := &testutil.MockPresenter{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := WaitForHealth(ctx, 19999, "/health", "test", p)
	require.Error(t, err)

	var ue *output.UserError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, output.CodeHealthCheckFailed, ue.Code)
}

func TestWaitForHealth_CorrectPath(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	p := &testutil.MockPresenter{}

	err := WaitForHealth(context.Background(), port, "/healthz", "test", p)
	require.NoError(t, err)
	assert.Equal(t, "/healthz", receivedPath)
}

func TestWaitForHealth_ProgressCalled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	port := extractPort(t, srv.URL)
	p := &testutil.MockPresenter{}

	err := WaitForHealth(context.Background(), port, "/", "test", p)
	require.NoError(t, err)
	assert.Contains(t, p.ProgressCalls, "Waiting for agent health...")
}

func extractPort(t *testing.T, url string) int {
	t.Helper()
	parts := strings.Split(url, ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	require.NoError(t, err)
	return port
}
