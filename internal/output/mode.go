package output

import (
	"os"
	"strconv"
	"strings"
)

// Mode represents the output formatting mode.
type Mode int

const (
	// ModeColor uses ANSI colors and emoji (default for TTY).
	ModeColor Mode = iota
	// ModeNoColor disables colors but keeps emoji (NO_COLOR set).
	ModeNoColor
	// ModePlain uses no colors or emoji (TERM=dumb or narrow terminal).
	ModePlain
)

// DetectMode determines the output mode based on environment variables
// and terminal capabilities.
func DetectMode() Mode {
	if os.Getenv("NO_COLOR") != "" {
		return ModeNoColor
	}

	term := os.Getenv("TERM")
	if strings.EqualFold(term, "dumb") {
		return ModePlain
	}

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n < 60 {
			return ModePlain
		}
	}

	return ModeColor
}
