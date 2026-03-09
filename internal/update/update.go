// Package update implements self-update functionality for the Volra CLI.
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Release holds information about a GitHub release.
type Release struct {
	Version   string
	AssetURL  string // Download URL for the correct OS/arch archive
	Checksum  string // Expected SHA256 checksum
	Published time.Time
}

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// githubRelease is the API response structure (subset).
type githubRelease struct {
	TagName     string        `json:"tag_name"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

const (
	releasesURL = "https://api.github.com/repos/romerox3/volra/releases/latest"
)

// CheckLatest queries GitHub Releases API for the latest version.
func CheckLatest(ctx context.Context, client HTTPClient) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("contacting GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	// Find the correct asset for current OS/arch
	archiveName := fmt.Sprintf("volra-%s-%s-%s.tar.gz",
		strings.TrimPrefix(release.TagName, "v"),
		runtime.GOOS,
		runtime.GOARCH,
	)

	var assetURL string
	var checksumURL string
	for _, a := range release.Assets {
		if a.Name == archiveName {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}

	if assetURL == "" {
		return nil, fmt.Errorf("no binary found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	// Fetch checksum
	checksum := ""
	if checksumURL != "" {
		checksum, err = fetchChecksum(ctx, client, checksumURL, archiveName)
		if err != nil {
			return nil, fmt.Errorf("fetching checksum: %w", err)
		}
	}

	return &Release{
		Version:   release.TagName,
		AssetURL:  assetURL,
		Checksum:  checksum,
		Published: release.PublishedAt,
	}, nil
}

// fetchChecksum downloads checksums.txt and extracts the checksum for the given archive.
func fetchChecksum(ctx context.Context, client HTTPClient, url, archiveName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == archiveName {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("checksum not found for %s", archiveName)
}

// Download fetches the release archive, verifies its checksum, extracts the binary,
// and replaces the current binary.
func Download(ctx context.Context, client HTTPClient, release *Release) error {
	// 1. Download archive to temp file
	req, err := http.NewRequestWithContext(ctx, "GET", release.AssetURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading release: %w", err)
	}
	defer resp.Body.Close()

	tmpArchive, err := os.CreateTemp("", "volra-update-*.tar.gz")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpArchive.Name())

	// Write and compute SHA256 simultaneously
	hasher := sha256.New()
	writer := io.MultiWriter(tmpArchive, hasher)
	if _, err := io.Copy(writer, resp.Body); err != nil {
		tmpArchive.Close()
		return fmt.Errorf("downloading: %w", err)
	}
	tmpArchive.Close()

	// 2. Verify checksum
	if release.Checksum != "" {
		got := hex.EncodeToString(hasher.Sum(nil))
		if got != release.Checksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", release.Checksum, got)
		}
	}

	// 3. Extract binary from tar.gz
	tmpBinary, err := extractBinaryFromArchive(tmpArchive.Name())
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}
	defer os.Remove(tmpBinary)

	// 4. Replace current binary
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	// Check write permission
	if err := checkWritable(currentBinary); err != nil {
		return err
	}

	// Atomic replace: rename temp to current
	if err := os.Rename(tmpBinary, currentBinary); err != nil {
		// Cross-device: copy instead
		return copyFile(tmpBinary, currentBinary)
	}

	return nil
}

// extractBinaryFromArchive extracts the "volra" binary from a tar.gz archive.
func extractBinaryFromArchive(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if filepath.Base(header.Name) == "volra" && header.Typeflag == tar.TypeReg {
			tmpFile, err := os.CreateTemp("", "volra-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmpFile, tr); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				return "", err
			}
			tmpFile.Close()
			if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
				os.Remove(tmpFile.Name())
				return "", err
			}
			return tmpFile.Name(), nil
		}
	}

	return "", fmt.Errorf("volra binary not found in archive")
}

// IsHomebrew checks if the current binary was installed via Homebrew.
func IsHomebrew() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return false
	}
	return strings.Contains(exe, "/Cellar/") || strings.Contains(exe, "/homebrew/")
}

// IsUpToDate compares current version with latest release version.
func IsUpToDate(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	return current == latest
}

func checkWritable(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("binary at %s is not writable: try `sudo volra update` or download manually from GitHub", path)
	}
	f.Close()
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}
