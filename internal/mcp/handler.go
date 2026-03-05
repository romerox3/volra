package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/romerox3/volra/internal/agentfile"
	"github.com/romerox3/volra/internal/deploy"
	"github.com/romerox3/volra/internal/docker"
	"github.com/romerox3/volra/internal/doctor"
	"github.com/romerox3/volra/internal/output"
	"github.com/romerox3/volra/internal/status"
)

// CallContext carries shared dependencies for tool handlers.
type CallContext struct {
	Ctx     context.Context
	Version string
	Runner  docker.DockerRunner
}

// capturePresenter collects Presenter output into a buffer.
type capturePresenter struct {
	buf bytes.Buffer
}

func (c *capturePresenter) Progress(msg string) { fmt.Fprintln(&c.buf, msg) }
func (c *capturePresenter) Result(msg string)   { fmt.Fprintln(&c.buf, msg) }
func (c *capturePresenter) Error(err error)     { fmt.Fprintln(&c.buf, "ERROR:", err) }
func (c *capturePresenter) Warn(w *output.UserWarning) {
	fmt.Fprintf(&c.buf, "WARNING: %s\n", w.What)
}

func (c *capturePresenter) String() string { return c.buf.String() }

// ---------------------------------------------------------------------------
// Tool handlers
// ---------------------------------------------------------------------------

type deployArgs struct {
	Path string `json:"path"`
}

func handleDeploy(cc *CallContext, raw json.RawMessage) *ToolCallResult {
	var args deployArgs
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &args)
	}
	dir := resolveDir(args.Path)

	p := &capturePresenter{}
	err := deploy.Run(cc.Ctx, dir, p, cc.Runner)
	if err != nil {
		return ErrorResult(fmt.Sprintf("%s\n%s", p.String(), err.Error()))
	}
	return SuccessResult(p.String())
}

type statusArgs struct {
	Path string `json:"path"`
}

func handleStatus(cc *CallContext, raw json.RawMessage) *ToolCallResult {
	var args statusArgs
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &args)
	}
	dir := resolveDir(args.Path)

	p := &capturePresenter{}
	err := status.Run(cc.Ctx, dir, p, cc.Runner)
	if err != nil {
		return ErrorResult(fmt.Sprintf("%s\n%s", p.String(), err.Error()))
	}
	return SuccessResult(p.String())
}

type logsArgs struct {
	Path    string `json:"path"`
	Lines   int    `json:"lines"`
	Service string `json:"service"`
}

func handleLogs(cc *CallContext, raw json.RawMessage) *ToolCallResult {
	var args logsArgs
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &args)
	}
	dir := resolveDir(args.Path)
	if args.Lines <= 0 {
		args.Lines = 50
	}

	af, err := agentfile.Load(filepath.Join(dir, "Agentfile"))
	if err != nil {
		return ErrorResult("No Agentfile found — are you in an agent project directory?")
	}

	composePath := filepath.Join(dir, ".volra", "docker-compose.yml")
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return ErrorResult("No deployment found — run 'volra deploy' first")
	}

	composeArgs := []string{"compose", "-f", composePath, "logs", "--tail", strconv.Itoa(args.Lines)}
	if args.Service != "" {
		composeArgs = append(composeArgs, af.Name+"-"+args.Service)
	} else {
		composeArgs = append(composeArgs, af.Name)
	}

	cmd := exec.CommandContext(cc.Ctx, "docker", composeArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ErrorResult(fmt.Sprintf("Failed to get logs: %s\n%s", err, string(out)))
	}
	return SuccessResult(string(out))
}

func handleDoctor(cc *CallContext, _ json.RawMessage) *ToolCallResult {
	p := &capturePresenter{}
	err := doctor.Run(cc.Ctx, cc.Version, p, cc.Runner, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("%s\n%s", p.String(), err.Error()))
	}
	return SuccessResult(p.String())
}

// resolveDir returns path if non-empty, otherwise current working directory.
func resolveDir(path string) string {
	if path != "" {
		return path
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
