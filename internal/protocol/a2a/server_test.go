package a2a

import (
	"encoding/json"
	"testing"
	"time"
)

func TestA2ATask(t *testing.T) {
	t.Run("TaskStatusString", func(t *testing.T) {
		tests := []struct {
			status   TaskStatus
			expected string
		}{
			{TaskSubmitted, "submitted"},
			{TaskWorking, "working"},
			{TaskCompleted, "completed"},
			{TaskFailed, "failed"},
			{TaskCancelled, "cancelled"},
		}

		for _, tt := range tests {
			if tt.status.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.status.String())
			}
		}
	})
}

func TestA2AServer(t *testing.T) {
	server := NewServer()

	t.Run("CreateTask", func(t *testing.T) {
		input := map[string]interface{}{
			"command": "test_command",
		}
		task, err := server.CreateTask("test_agent", input)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		if task.AgentID != "test_agent" {
			t.Errorf("Expected agent ID 'test_agent', got '%s'", task.AgentID)
		}
		if task.Status != TaskSubmitted {
			t.Errorf("Expected status 'submitted', got '%s'", task.Status.String())
		}
	})

	t.Run("GetTask", func(t *testing.T) {
		input := map[string]interface{}{"test": "data"}
		task, _ := server.CreateTask("test_agent", input)

		gotTask, exists := server.GetTask(task.ID)
		if !exists {
			t.Fatal("Task not found")
		}
		if gotTask.ID != task.ID {
			t.Errorf("Expected task ID %s, got %s", task.ID, gotTask.ID)
		}
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		input := map[string]interface{}{}
		task, _ := server.CreateTask("test_agent", input)

		err := server.UpdateTaskStatus(task.ID, TaskWorking)
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		updatedTask, _ := server.GetTask(task.ID)
		if updatedTask.Status != TaskWorking {
			t.Errorf("Expected status 'working', got '%s'", updatedTask.Status.String())
		}
	})

	t.Run("SetTaskOutput", func(t *testing.T) {
		input := map[string]interface{}{}
		task, _ := server.CreateTask("test_agent", input)

		output := map[string]interface{}{"result": "success"}
		err := server.SetTaskOutput(task.ID, output)
		if err != nil {
			t.Fatalf("Failed to set output: %v", err)
		}

		updatedTask, _ := server.GetTask(task.ID)
		if updatedTask.Status != TaskCompleted {
			t.Errorf("Expected status 'completed', got '%s'", updatedTask.Status.String())
		}
		if updatedTask.Output["result"] != "success" {
			t.Errorf("Expected output result 'success', got '%v'", updatedTask.Output["result"])
		}
	})

	t.Run("SetTaskError", func(t *testing.T) {
		input := map[string]interface{}{}
		task, _ := server.CreateTask("test_agent", input)

		err := server.SetTaskError(task.ID, "test error")
		if err != nil {
			t.Fatalf("Failed to set error: %v", err)
		}

		updatedTask, _ := server.GetTask(task.ID)
		if updatedTask.Status != TaskFailed {
			t.Errorf("Expected status 'failed', got '%s'", updatedTask.Status.String())
		}
		if updatedTask.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", updatedTask.Error)
		}
	})

	t.Run("ListTasks", func(t *testing.T) {
		// Create tasks with different statuses
		server.CreateTask("agent1", map[string]interface{}{})
		server.CreateTask("agent2", map[string]interface{}{})

		allTasks := server.ListTasks(nil)
		if len(allTasks) < 2 {
			t.Errorf("Expected at least 2 tasks, got %d", len(allTasks))
		}
	})

	t.Run("CancelTask", func(t *testing.T) {
		input := map[string]interface{}{}
		task, _ := server.CreateTask("test_agent", input)

		err := server.CancelTask(task.ID)
		if err != nil {
			t.Fatalf("Failed to cancel task: %v", err)
		}

		cancelledTask, _ := server.GetTask(task.ID)
		if cancelledTask.Status != TaskCancelled {
			t.Errorf("Expected status 'cancelled', got '%s'", cancelledTask.Status.String())
		}
	})

	t.Run("DeleteTask", func(t *testing.T) {
		input := map[string]interface{}{}
		task, _ := server.CreateTask("test_agent", input)

		err := server.DeleteTask(task.ID)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, exists := server.GetTask(task.ID)
		if exists {
			t.Error("Task should not exist after deletion")
		}
	})
}

func TestA2ASkills(t *testing.T) {
	server := NewServer()

	t.Run("ListSkills", func(t *testing.T) {
		skills := server.ListSkills()
		if len(skills) == 0 {
			t.Error("Expected skills to be registered")
		}
	})

	t.Run("GetSkill", func(t *testing.T) {
		skill, exists := server.GetSkill("smartRouting")
		if !exists {
			t.Fatal("Skill not found")
		}
		if skill.Name != "smartRouting" {
			t.Errorf("Expected skill name 'smartRouting', got '%s'", skill.Name)
		}
	})

	t.Run("BuiltInSkills", func(t *testing.T) {
		expectedSkills := []string{
			"smartRouting",
			"quotaManagement",
			"providerDiscovery",
			"costAnalysis",
			"healthReport",
		}

		for _, name := range expectedSkills {
			skill, exists := server.GetSkill(name)
			if !exists {
				t.Errorf("Expected skill %s to exist", name)
				continue
			}
			if skill.Name != name {
				t.Errorf("Expected name '%s', got '%s'", name, skill.Name)
			}
		}
	})
}

func TestA2ATaskJSON(t *testing.T) {
	task := &A2ATask{
		ID:        "task-123",
		Status:    TaskCompleted,
		AgentID:  "test_agent",
		Input:    map[string]interface{}{"key": "value"},
		Output:   map[string]interface{}{"result": "success"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test ToJSON
	data, err := task.ToJSON()
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	// Test TaskFromJSON
	parsedTask, err := TaskFromJSON(data)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if parsedTask.ID != task.ID {
		t.Errorf("Expected ID '%s', got '%s'", task.ID, parsedTask.ID)
	}
	if parsedTask.AgentID != task.AgentID {
		t.Errorf("Expected AgentID '%s', got '%s'", task.AgentID, parsedTask.AgentID)
	}
}

func TestStreamingServer(t *testing.T) {
	server := NewStreamingServer()

	t.Run("CreateTaskWithStream", func(t *testing.T) {
		input := map[string]interface{}{
			"command": "stream_test",
		}
		task, ch, unsub := server.CreateTaskWithStream("test_agent", input)
		defer unsub()

		if task == nil {
			t.Fatal("Expected task to be created")
		}
		if ch == nil {
			t.Fatal("Expected channel to be created")
		}
		if task.AgentID != "test_agent" {
			t.Errorf("Expected agent ID 'test_agent', got '%s'", task.AgentID)
		}
	})
}

func TestSSEHandler(t *testing.T) {
	server := NewServer()
	sseHandler := NewSSEHandler(server)

	t.Run("Subscribe", func(t *testing.T) {
		ch, unsub := sseHandler.Subscribe("task-123")
		if ch == nil {
			t.Fatal("Expected channel to be created")
		}
		unsub()
	})

	t.Run("SubscribeAll", func(t *testing.T) {
		ch, unsub := sseHandler.SubscribeAll()
		if ch == nil {
			t.Fatal("Expected channel to be created")
		}
		unsub()
	})

	t.Run("FormatSSE", func(t *testing.T) {
		data := map[string]interface{}{"test": "value"}
		formatted, err := FormatSSE("test_event", data)
		if err != nil {
			t.Fatalf("Failed to format SSE: %v", err)
		}

		expected := "event: test_event\ndata: {\"test\":\"value\"}\n\n"
		if formatted != expected {
			t.Errorf("Expected '%s', got '%s'", expected, formatted)
		}
	})
}

func TestTaskJSONSerialization(t *testing.T) {
	now := time.Now()
	task := &A2ATask{
		ID:        "task-456",
		Status:    TaskWorking,
		AgentID:  "another_agent",
		Input:    map[string]interface{}{"data": 123},
		Metadata: map[string]interface{}{"priority": "high"},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed A2ATask
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.ID != task.ID {
		t.Errorf("ID mismatch")
	}
	if parsed.Status != task.Status {
		t.Errorf("Status mismatch")
	}
}
