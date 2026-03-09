package update

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient returns predefined responses.
type mockHTTPClient struct {
	responses map[string]*http.Response
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if resp, ok := m.responses[req.URL.String()]; ok {
		return resp, nil
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}, nil
}

func makeReleaseJSON(tag string, assets []githubAsset) []byte {
	r := githubRelease{
		TagName: tag,
		Assets:  assets,
	}
	b, _ := json.Marshal(r)
	return b
}

func TestCheckLatest_Success(t *testing.T) {
	releaseJSON := makeReleaseJSON("v0.4.1", []githubAsset{
		{Name: "volra-0.4.1-darwin-arm64.tar.gz", BrowserDownloadURL: "https://example.com/volra-0.4.1-darwin-arm64.tar.gz"},
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
	})

	checksumContent := "abc123  volra-0.4.1-darwin-arm64.tar.gz\ndef456  volra-0.4.1-linux-amd64.tar.gz\n"

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			releasesURL: {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(releaseJSON)),
			},
			"https://example.com/checksums.txt": {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(checksumContent))),
			},
		},
	}

	release, err := CheckLatest(context.Background(), client)
	require.NoError(t, err)
	assert.Equal(t, "v0.4.1", release.Version)
	assert.NotEmpty(t, release.AssetURL)
}

func TestCheckLatest_APIError(t *testing.T) {
	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			releasesURL: {
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			},
		},
	}

	_, err := CheckLatest(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCheckLatest_NoMatchingAsset(t *testing.T) {
	releaseJSON := makeReleaseJSON("v0.4.1", []githubAsset{
		{Name: "volra-0.4.1-windows-amd64.tar.gz", BrowserDownloadURL: "https://example.com/windows.tar.gz"},
	})

	client := &mockHTTPClient{
		responses: map[string]*http.Response{
			releasesURL: {
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(releaseJSON)),
			},
		},
	}

	_, err := CheckLatest(context.Background(), client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no binary found")
}

func TestIsUpToDate(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"v0.4.0", "v0.4.0", true},
		{"0.4.0", "v0.4.0", true},
		{"v0.4.0", "0.4.0", true},
		{"v0.4.0", "v0.4.1", false},
		{"v0.3.0", "v0.4.0", false},
		{"dev", "v0.4.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.want, IsUpToDate(tt.current, tt.latest))
		})
	}
}

func TestIsHomebrew(t *testing.T) {
	// This test just verifies it doesn't panic — actual behavior depends on binary path
	_ = IsHomebrew()
}
