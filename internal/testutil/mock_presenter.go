package testutil

import "github.com/antonioromero/volra/internal/output"

// MockPresenter implements output.Presenter and records all calls for test assertions.
type MockPresenter struct {
	ProgressCalls []string
	ResultCalls   []string
	ErrorCalls    []error
	WarnCalls     []*output.UserWarning
}

func (m *MockPresenter) Progress(msg string) {
	m.ProgressCalls = append(m.ProgressCalls, msg)
}

func (m *MockPresenter) Result(msg string) {
	m.ResultCalls = append(m.ResultCalls, msg)
}

func (m *MockPresenter) Error(err error) {
	m.ErrorCalls = append(m.ErrorCalls, err)
}

func (m *MockPresenter) Warn(w *output.UserWarning) {
	m.WarnCalls = append(m.WarnCalls, w)
}
