package marketplace

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Install downloads and extracts a marketplace template to the target directory.
func (c *Client) Install(t *Template, targetDir string) error {
	// GitHub tarball URL pattern: {repo_url}/archive/refs/heads/main.tar.gz
	tarURL := strings.TrimSuffix(t.RepoURL, "/") + "/archive/refs/heads/main.tar.gz"

	resp, err := c.HTTPClient.Get(tarURL)
	if err != nil {
		return fmt.Errorf("downloading template %s: %w", t.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading template %s: HTTP %d", t.Name, resp.StatusCode)
	}

	return extractTarGz(resp.Body, targetDir)
}

const (
	// maxFileSize limits individual extracted files to 50MB.
	maxFileSize = 50 << 20
	// maxTotalSize limits total extracted content to 500MB.
	maxTotalSize = 500 << 20
)

// extractTarGz reads a gzip-compressed tar and extracts it to dir.
// Strips the top-level directory (GitHub archives wrap in repo-branch/).
func extractTarGz(r io.Reader, dir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("reading gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	stripPrefix := ""
	var totalBytes int64

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		// Skip symlinks and hard links (security: path traversal risk).
		if hdr.Typeflag == tar.TypeSymlink || hdr.Typeflag == tar.TypeLink {
			continue
		}

		// Determine the top-level prefix to strip (e.g., "repo-main/").
		if stripPrefix == "" {
			parts := strings.SplitN(hdr.Name, "/", 2)
			if len(parts) > 0 {
				stripPrefix = parts[0] + "/"
			}
		}

		name := strings.TrimPrefix(hdr.Name, stripPrefix)
		if name == "" {
			continue
		}

		target := filepath.Join(dir, name)

		// Path traversal protection.
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dir)) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if hdr.Size > maxFileSize {
				return fmt.Errorf("file %s exceeds max size (%d bytes)", name, maxFileSize)
			}
			totalBytes += hdr.Size
			if totalBytes > maxTotalSize {
				return fmt.Errorf("total extracted size exceeds limit (%d bytes)", maxTotalSize)
			}

			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating parent dir for %s: %w", target, err)
			}
			// Mask file mode to strip setuid/setgid/sticky bits.
			mode := os.FileMode(hdr.Mode) & 0o777
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(f, io.LimitReader(tr, maxFileSize)); err != nil {
				f.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			f.Close()
		}
	}

	return nil
}

// IsEmpty returns true if the directory is empty or doesn't exist.
func IsEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return len(entries) == 0, nil
}
