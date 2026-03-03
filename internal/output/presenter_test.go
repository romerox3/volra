package output

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testPresenter creates a presenter with captured stdout/stderr.
func testPresenter(mode Mode) (Presenter, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	switch mode {
	case ModeNoColor:
		return &noColorPresenter{stdout: stdout, stderr: stderr}, stdout, stderr
	case ModePlain:
		return &plainPresenter{stdout: stdout, stderr: stderr}, stdout, stderr
	default:
		return &colorPresenter{stdout: stdout, stderr: stderr}, stdout, stderr
	}
}

func TestPresenter_ProgressWritesToStderr(t *testing.T) {
	for _, mode := range []Mode{ModeColor, ModeNoColor, ModePlain} {
		p, stdout, stderr := testPresenter(mode)
		p.Progress("checking docker...")

		assert.Empty(t, stdout.String(), "Progress should not write to stdout")
		assert.Contains(t, stderr.String(), "checking docker...")
	}
}

func TestPresenter_ResultWritesToStdout(t *testing.T) {
	for _, mode := range []Mode{ModeColor, ModeNoColor, ModePlain} {
		p, stdout, stderr := testPresenter(mode)
		p.Result("deploy complete")

		assert.Contains(t, stdout.String(), "deploy complete")
		assert.Empty(t, stderr.String(), "Result should not write to stderr")
	}
}

func TestPresenter_ErrorWithUserError(t *testing.T) {
	ue := &UserError{
		Code: "E101",
		What: "Docker is not installed",
		Fix:  "Install Docker Desktop from https://docker.com",
	}

	tests := []struct {
		name     string
		mode     Mode
		wantCode string
		wantWhat string
		wantFix  string
	}{
		{"color", ModeColor, "[E101]", "Docker is not installed", "Install Docker Desktop from https://docker.com"},
		{"nocolor", ModeNoColor, "[E101]", "Docker is not installed", "Install Docker Desktop from https://docker.com"},
		{"plain", ModePlain, "[E101]", "Docker is not installed", "Install Docker Desktop from https://docker.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, stdout, stderr := testPresenter(tt.mode)
			p.Error(ue)

			assert.Empty(t, stdout.String())
			assert.Contains(t, stderr.String(), tt.wantCode)
			assert.Contains(t, stderr.String(), tt.wantWhat)
			assert.Contains(t, stderr.String(), tt.wantFix)
		})
	}
}

func TestPresenter_ErrorWithGenericError(t *testing.T) {
	err := errors.New("unexpected failure")

	tests := []struct {
		name string
		mode Mode
	}{
		{"color", ModeColor},
		{"nocolor", ModeNoColor},
		{"plain", ModePlain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, stdout, stderr := testPresenter(tt.mode)
			p.Error(err)

			assert.Empty(t, stdout.String())
			assert.Contains(t, stderr.String(), "[INTERNAL]")
			assert.Contains(t, stderr.String(), "unexpected failure")
		})
	}
}

func TestPresenter_ErrorWithWrappedUserError(t *testing.T) {
	ue := &UserError{Code: "E302", What: "Docker build failed", Fix: "Check Dockerfile"}
	wrapped := errors.Join(errors.New("deploy step failed"), ue)

	p, _, stderr := testPresenter(ModeNoColor)
	p.Error(wrapped)

	assert.Contains(t, stderr.String(), "[E302]")
	assert.Contains(t, stderr.String(), "Docker build failed")
}

func TestPresenter_WarnWithAssumed(t *testing.T) {
	w := &UserWarning{
		What:     "Port not detected",
		Assumed:  "8000",
		Override: "Set port in Agentfile",
	}

	tests := []struct {
		name string
		mode Mode
	}{
		{"color", ModeColor},
		{"nocolor", ModeNoColor},
		{"plain", ModePlain},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, stdout, stderr := testPresenter(tt.mode)
			p.Warn(w)

			assert.Empty(t, stdout.String())
			assert.Contains(t, stderr.String(), "Port not detected")
			assert.Contains(t, stderr.String(), "8000")
			assert.Contains(t, stderr.String(), "Set port in Agentfile")
		})
	}
}

func TestPresenter_WarnWithoutAssumed(t *testing.T) {
	w := &UserWarning{
		What:     "Config file missing",
		Override: "Create a config file",
	}

	p, _, stderr := testPresenter(ModeNoColor)
	p.Warn(w)

	assert.Contains(t, stderr.String(), "Config file missing")
	assert.Contains(t, stderr.String(), "Create a config file")
	assert.NotContains(t, stderr.String(), "assumed")
}

func TestPlainPresenter_UsesTextPrefixes(t *testing.T) {
	p, _, stderr := testPresenter(ModePlain)

	p.Error(&UserError{Code: "E101", What: "test error", Fix: "fix it"})
	assert.Contains(t, stderr.String(), "->")

	stderr.Reset()
	p.Warn(&UserWarning{What: "test warn", Override: "override it"})
	assert.Contains(t, stderr.String(), "WARNING:")
}

func TestNewPresenter_ReturnsModes(t *testing.T) {
	p1 := NewPresenter(ModeColor)
	assert.IsType(t, &colorPresenter{}, p1)

	p2 := NewPresenter(ModeNoColor)
	assert.IsType(t, &noColorPresenter{}, p2)

	p3 := NewPresenter(ModePlain)
	assert.IsType(t, &plainPresenter{}, p3)
}
