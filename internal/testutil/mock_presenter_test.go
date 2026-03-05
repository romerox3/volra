package testutil

import (
	"errors"
	"testing"

	"github.com/romerox3/volra/internal/output"
	"github.com/stretchr/testify/assert"
)

// Compile-time interface check.
var _ output.Presenter = (*MockPresenter)(nil)

func TestMockPresenter_RecordsCalls(t *testing.T) {
	m := &MockPresenter{}

	m.Progress("step 1")
	m.Progress("step 2")
	m.Result("done")
	m.Error(errors.New("oops"))
	m.Warn(&output.UserWarning{What: "heads up", Override: "fix it"})

	assert.Equal(t, []string{"step 1", "step 2"}, m.ProgressCalls)
	assert.Equal(t, []string{"done"}, m.ResultCalls)
	assert.Len(t, m.ErrorCalls, 1)
	assert.Len(t, m.WarnCalls, 1)
	assert.Equal(t, "heads up", m.WarnCalls[0].What)
}
