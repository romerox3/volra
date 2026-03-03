package output

import "fmt"

// UserError represents a user-facing error with an error code and fix suggestion.
type UserError struct {
	Code string // E1xx=doctor, E2xx=setup, E3xx=deploy, E4xx=status, E5xx=shared
	What string // What happened
	Fix  string // How to fix it
}

func (e *UserError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.What)
}

// UserWarning represents a non-fatal warning with assumed defaults and override instructions.
type UserWarning struct {
	What     string // What happened
	Assumed  string // What was assumed (empty for non-default warnings)
	Override string // How to override
}
