package output

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectMode_Default(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "")
	t.Setenv("COLUMNS", "")

	assert.Equal(t, ModeColor, DetectMode())
}

func TestDetectMode_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	assert.Equal(t, ModeNoColor, DetectMode())
}

func TestDetectMode_TermDumb(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")

	assert.Equal(t, ModePlain, DetectMode())
}

func TestDetectMode_NarrowTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	t.Setenv("COLUMNS", "40")

	assert.Equal(t, ModePlain, DetectMode())
}

func TestDetectMode_WideTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	t.Setenv("COLUMNS", "120")

	assert.Equal(t, ModeColor, DetectMode())
}

func TestDetectMode_NoColorTakesPrecedence(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("TERM", "dumb")

	assert.Equal(t, ModeNoColor, DetectMode())
}
