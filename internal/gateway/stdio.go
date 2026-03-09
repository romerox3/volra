package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/romerox3/volra/internal/mcp"
)

// newLineScanner creates a buffered scanner for reading JSON-RPC lines.
func newLineScanner(r io.Reader) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	return s
}

// readJSONRPCResponse reads a single JSON-RPC response line with context cancellation.
func readJSONRPCResponse(ctx context.Context, scanner *bufio.Scanner) (*mcp.Response, error) {
	done := make(chan struct{})
	var resp mcp.Response
	var scanErr error

	go func() {
		defer close(done)
		if !scanner.Scan() {
			scanErr = fmt.Errorf("EOF from MCP server")
			if scanner.Err() != nil {
				scanErr = scanner.Err()
			}
			return
		}
		scanErr = json.Unmarshal(scanner.Bytes(), &resp)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		if scanErr != nil {
			return nil, scanErr
		}
		return &resp, nil
	}
}
