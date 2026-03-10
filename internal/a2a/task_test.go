package a2a

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTask(t *testing.T) {
	msg := Message{
		Role:  "user",
		Parts: []Part{{Type: "text", Text: "hello"}},
	}

	task := NewTask(msg)

	assert.NotEmpty(t, task.ID)
	assert.Len(t, task.ID, 32) // 16 bytes hex
	assert.Equal(t, TaskStateSubmitted, task.Status.State)
	require.Len(t, task.History, 1)
	assert.Equal(t, "user", task.History[0].Role)
	assert.False(t, task.CreatedAt.IsZero())
}

func TestTask_Transition(t *testing.T) {
	task := NewTask(Message{Role: "user", Parts: []Part{{Type: "text", Text: "do something"}}})

	task.Transition(TaskStateWorking, nil)
	assert.Equal(t, TaskStateWorking, task.Status.State)
	assert.Len(t, task.History, 1) // no new message

	agentMsg := &Message{Role: "agent", Parts: []Part{{Type: "text", Text: "processing..."}}}
	task.Transition(TaskStateWorking, agentMsg)
	assert.Len(t, task.History, 2)
	assert.Equal(t, "processing...", task.History[1].Parts[0].Text)
}

func TestTask_Complete(t *testing.T) {
	task := NewTask(Message{Role: "user", Parts: []Part{{Type: "text", Text: "summarize"}}})
	task.Transition(TaskStateWorking, nil)

	artifacts := []Artifact{
		{Parts: []Part{{Type: "text", Text: "Summary result"}}},
	}
	task.Complete(artifacts)

	assert.Equal(t, TaskStateCompleted, task.Status.State)
	require.Len(t, task.Artifacts, 1)
	assert.Equal(t, "Summary result", task.Artifacts[0].Parts[0].Text)
}

func TestTask_Fail(t *testing.T) {
	task := NewTask(Message{Role: "user", Parts: []Part{{Type: "text", Text: "do"}}})
	task.Transition(TaskStateWorking, nil)

	task.Fail("something went wrong")

	assert.Equal(t, TaskStateFailed, task.Status.State)
	require.NotNil(t, task.Status.Message)
	assert.Equal(t, "something went wrong", task.Status.Message.Parts[0].Text)
	assert.Len(t, task.History, 2) // original + error
}

func TestTask_Cancel(t *testing.T) {
	task := NewTask(Message{Role: "user", Parts: []Part{{Type: "text", Text: "do"}}})
	task.Cancel()
	assert.Equal(t, TaskStateCanceled, task.Status.State)
}

func TestTask_JSONRoundtrip(t *testing.T) {
	task := NewTask(Message{Role: "user", Parts: []Part{{Type: "text", Text: "hello"}}})
	task.Transition(TaskStateWorking, nil)
	task.Complete([]Artifact{{Parts: []Part{{Type: "text", Text: "result"}}}})

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var parsed Task
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, task.ID, parsed.ID)
	assert.Equal(t, TaskStateCompleted, parsed.Status.State)
	assert.Len(t, parsed.Artifacts, 1)
	assert.Equal(t, "result", parsed.Artifacts[0].Parts[0].Text)
}

func TestGenerateID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateID()
		assert.False(t, ids[id], "duplicate ID generated")
		ids[id] = true
	}
}
