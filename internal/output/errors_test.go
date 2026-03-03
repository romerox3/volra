package output

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserError_Error(t *testing.T) {
	ue := &UserError{
		Code: "E101",
		What: "Docker is not installed",
		Fix:  "Install Docker Desktop from https://docker.com",
	}

	assert.Equal(t, "[E101] Docker is not installed", ue.Error())
}

func TestUserError_ImplementsError(t *testing.T) {
	var err error = &UserError{Code: "E101", What: "test", Fix: "fix"}
	assert.Error(t, err)
}

func TestUserError_ErrorsAs(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &UserError{Code: "E102", What: "Docker is not running", Fix: "Start Docker"})

	var ue *UserError
	require.True(t, errors.As(err, &ue))
	assert.Equal(t, "E102", ue.Code)
	assert.Equal(t, "Docker is not running", ue.What)
	assert.Equal(t, "Start Docker", ue.Fix)
}

func TestUserWarning_Fields(t *testing.T) {
	w := &UserWarning{
		What:     "Port not detected",
		Assumed:  "8000",
		Override: "Set port in Agentfile",
	}

	assert.Equal(t, "Port not detected", w.What)
	assert.Equal(t, "8000", w.Assumed)
	assert.Equal(t, "Set port in Agentfile", w.Override)
}
