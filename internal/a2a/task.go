package a2a

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// TaskState represents the state of an A2A task.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCanceled      TaskState = "canceled"
)

// Task represents an A2A task per the v0.3 spec.
type Task struct {
	ID        string     `json:"id"`
	ContextID string     `json:"contextId,omitempty"`
	Status    TaskStatus `json:"status"`
	History   []Message  `json:"history,omitempty"`
	Artifacts []Artifact `json:"artifacts,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// TaskStatus holds the current state of a task.
type TaskStatus struct {
	State   TaskState `json:"state"`
	Message *Message  `json:"message,omitempty"`
}

// Message represents a message in the A2A protocol.
type Message struct {
	Role  string `json:"role"` // "user" or "agent"
	Parts []Part `json:"parts"`
}

// Part is a content part in a message or artifact.
type Part struct {
	Type string `json:"type"` // "text", "data", "file"
	Text string `json:"text,omitempty"`
}

// Artifact is a result artifact attached to a task.
type Artifact struct {
	Name  string `json:"name,omitempty"`
	Parts []Part `json:"parts"`
}

// NewTask creates a new task with the given initial message.
func NewTask(message Message) *Task {
	now := time.Now().UTC()
	return &Task{
		ID:      generateID(),
		Status:  TaskStatus{State: TaskStateSubmitted},
		History: []Message{message},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Transition moves the task to a new state and records an optional message.
func (t *Task) Transition(state TaskState, msg *Message) {
	t.Status.State = state
	t.Status.Message = msg
	if msg != nil {
		t.History = append(t.History, *msg)
	}
	t.UpdatedAt = time.Now().UTC()
}

// Complete marks the task as completed with the given artifacts.
func (t *Task) Complete(artifacts []Artifact) {
	t.Status.State = TaskStateCompleted
	t.Artifacts = artifacts
	t.UpdatedAt = time.Now().UTC()
}

// Fail marks the task as failed with an error message.
func (t *Task) Fail(errMsg string) {
	t.Status.State = TaskStateFailed
	t.Status.Message = &Message{
		Role:  "agent",
		Parts: []Part{{Type: "text", Text: errMsg}},
	}
	t.History = append(t.History, *t.Status.Message)
	t.UpdatedAt = time.Now().UTC()
}

// Cancel marks the task as canceled.
func (t *Task) Cancel() {
	t.Status.State = TaskStateCanceled
	t.UpdatedAt = time.Now().UTC()
}

// generateID creates a random hex ID for tasks.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
