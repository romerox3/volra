package output

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Presenter defines the interface for all CLI output.
// Progress, Error, and Warn write to stderr. Result writes to stdout.
type Presenter interface {
	Progress(msg string)
	Result(msg string)
	Error(err error)
	Warn(w *UserWarning)
}

// NewPresenter creates a Presenter for the given mode.
func NewPresenter(mode Mode) Presenter {
	switch mode {
	case ModeNoColor:
		return &noColorPresenter{stdout: os.Stdout, stderr: os.Stderr}
	case ModePlain:
		return &plainPresenter{stdout: os.Stdout, stderr: os.Stderr}
	default:
		return &colorPresenter{stdout: os.Stdout, stderr: os.Stderr}
	}
}

// writef writes formatted output, discarding write errors (non-actionable for terminal I/O).
func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

// writeln writes a line, discarding write errors (non-actionable for terminal I/O).
func writeln(w io.Writer, msg string) {
	_, _ = fmt.Fprintln(w, msg)
}

// --- Color Presenter (default) ---

type colorPresenter struct {
	stdout io.Writer
	stderr io.Writer
}

func (p *colorPresenter) Progress(msg string) {
	writeln(p.stderr, msg)
}

func (p *colorPresenter) Result(msg string) {
	writeln(p.stdout, msg)
}

func (p *colorPresenter) Error(err error) {
	var ue *UserError
	if errors.As(err, &ue) {
		writef(p.stderr, "\033[31m[%s] %s\033[0m\n  \u2192 %s\n", ue.Code, ue.What, ue.Fix)
		return
	}
	writef(p.stderr, "\033[31m[INTERNAL] %s\033[0m\n", err.Error())
}

func (p *colorPresenter) Warn(w *UserWarning) {
	if w.Assumed != "" {
		writef(p.stderr, "\033[33m\u26a0\ufe0f  %s (assumed %s)\033[0m\n  \u2192 %s\n", w.What, w.Assumed, w.Override)
		return
	}
	writef(p.stderr, "\033[33m\u26a0\ufe0f  %s\033[0m\n  \u2192 %s\n", w.What, w.Override)
}

// --- NoColor Presenter ---

type noColorPresenter struct {
	stdout io.Writer
	stderr io.Writer
}

func (p *noColorPresenter) Progress(msg string) {
	writeln(p.stderr, msg)
}

func (p *noColorPresenter) Result(msg string) {
	writeln(p.stdout, msg)
}

func (p *noColorPresenter) Error(err error) {
	var ue *UserError
	if errors.As(err, &ue) {
		writef(p.stderr, "[%s] %s\n  \u2192 %s\n", ue.Code, ue.What, ue.Fix)
		return
	}
	writef(p.stderr, "[INTERNAL] %s\n", err.Error())
}

func (p *noColorPresenter) Warn(w *UserWarning) {
	if w.Assumed != "" {
		writef(p.stderr, "\u26a0\ufe0f  %s (assumed %s)\n  \u2192 %s\n", w.What, w.Assumed, w.Override)
		return
	}
	writef(p.stderr, "\u26a0\ufe0f  %s\n  \u2192 %s\n", w.What, w.Override)
}

// --- Plain Presenter ---

type plainPresenter struct {
	stdout io.Writer
	stderr io.Writer
}

func (p *plainPresenter) Progress(msg string) {
	writeln(p.stderr, msg)
}

func (p *plainPresenter) Result(msg string) {
	writeln(p.stdout, msg)
}

func (p *plainPresenter) Error(err error) {
	var ue *UserError
	if errors.As(err, &ue) {
		writef(p.stderr, "[%s] %s\n  -> %s\n", ue.Code, ue.What, ue.Fix)
		return
	}
	writef(p.stderr, "[INTERNAL] %s\n", err.Error())
}

func (p *plainPresenter) Warn(w *UserWarning) {
	if w.Assumed != "" {
		writef(p.stderr, "WARNING: %s (assumed %s)\n  -> %s\n", w.What, w.Assumed, w.Override)
		return
	}
	writef(p.stderr, "WARNING: %s\n  -> %s\n", w.What, w.Override)
}
