package observability

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestAuditLog_Creation(t *testing.T) {
	actor := &Actor{
		Type:  "user",
		ID:    "user-123",
		Name:  "Test User",
		Email: "test@example.com",
	}

	resource := &Resource{
		Type: "provider",
		ID:   "provider-openai",
		Name: "OpenAI",
	}

	log := NewAuditLog(actor, "created", resource)

	t.Run("creates audit log with ID", func(t *testing.T) {
		if log.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("sets timestamp", func(t *testing.T) {
		if log.Timestamp.IsZero() {
			t.Error("Expected timestamp to be set")
		}
	})

	t.Run("sets actor", func(t *testing.T) {
		if log.Actor.ID != "user-123" {
			t.Errorf("Expected actor ID user-123, got %s", log.Actor.ID)
		}
	})

	t.Run("sets action", func(t *testing.T) {
		if log.Action != "created" {
			t.Errorf("Expected action created, got %s", log.Action)
		}
	})

	t.Run("sets resource", func(t *testing.T) {
		if log.Resource.Type != "provider" {
			t.Errorf("Expected resource type provider, got %s", log.Resource.Type)
		}
	})

	t.Run("initializes changes map", func(t *testing.T) {
		if log.Changes == nil {
			t.Error("Expected changes map to be initialized")
		}
	})

	t.Run("initializes context map", func(t *testing.T) {
		if log.Context == nil {
			t.Error("Expected context map to be initialized")
		}
	})

	t.Run("sets default status", func(t *testing.T) {
		if log.Status != "success" {
			t.Errorf("Expected status success, got %s", log.Status)
		}
	})
}

func TestAuditLog_AddChange(t *testing.T) {
	log := NewAuditLog(nil, "updated", &Resource{Type: "budget", ID: "budget-1"})

	log.AddChange("balance", 100.0, 150.0)

	if len(log.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(log.Changes))
	}

	change := log.Changes["balance"]
	if change == nil {
		t.Fatal("Expected change to exist")
	}

	if change.OldValue != 100.0 {
		t.Errorf("Expected old value 100.0, got %v", change.OldValue)
	}

	if change.NewValue != 150.0 {
		t.Errorf("Expected new value 150.0, got %v", change.NewValue)
	}
}

func TestAuditLog_SetError(t *testing.T) {
	log := NewAuditLog(nil, "executed", &Resource{Type: "test", ID: "1"})

	if log.Status != "success" {
		t.Errorf("Expected initial status success, got %s", log.Status)
	}

	log.SetError("something went wrong")

	if log.Status != "failure" {
		t.Errorf("Expected status failure, got %s", log.Status)
	}

	if log.Error != "something went wrong" {
		t.Errorf("Expected error message, got %s", log.Error)
	}
}

func TestAuditLog_SetContext(t *testing.T) {
	log := NewAuditLog(nil, "test", &Resource{Type: "test", ID: "1"})

	log.SetContext("key1", "value1")
	log.SetContext("key2", 42)

	if log.Context["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", log.Context["key1"])
	}

	if log.Context["key2"] != 42 {
		t.Errorf("Expected key2=42, got %v", log.Context["key2"])
	}
}

func TestAuditQuery_Cursor(t *testing.T) {
	query := &AuditQuery{
		StartTime: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC),
		Limit:     50,
	}

	cursor := query.Cursor()
	if cursor == "" {
		t.Error("Expected cursor to be generated")
	}

	t.Run("cursor is base64 encoded", func(t *testing.T) {
		decoded, err := base64.URLEncoding.DecodeString(cursor)
		if err != nil {
			t.Fatalf("Expected valid base64, got error: %v", err)
		}

		var data map[string]interface{}
		if err := json.Unmarshal(decoded, &data); err != nil {
			t.Fatalf("Expected valid JSON, got error: %v", err)
		}
	})

	t.Run("cursor contains timestamp", func(t *testing.T) {
		parsed, err := ParseCursor(cursor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if parsed == nil {
			t.Fatal("Expected parsed cursor to be non-nil")
		}

		if parsed.StartTime.IsZero() {
			t.Error("Expected StartTime to be set from cursor")
		}
	})
}

func TestParseCursor(t *testing.T) {
	t.Run("parses valid cursor", func(t *testing.T) {
		cursorData := map[string]interface{}{
			"timestamp": float64(time.Now().UnixNano()),
			"id":        "test-id",
		}
		data, _ := json.Marshal(cursorData)
		cursor := base64.URLEncoding.EncodeToString(data)

		parsed, err := ParseCursor(cursor)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if parsed == nil {
			t.Fatal("Expected parsed cursor to be non-nil")
		}

		if parsed.StartTime.IsZero() {
			t.Error("Expected StartTime to be set")
		}
	})

	t.Run("returns nil for empty cursor", func(t *testing.T) {
		parsed, err := ParseCursor("")
		if err != nil {
			t.Fatalf("Expected no error for empty cursor, got %v", err)
		}

		if parsed != nil {
			t.Error("Expected nil for empty cursor")
		}
	})

	t.Run("returns error for invalid base64", func(t *testing.T) {
		_, err := ParseCursor("not-valid-base64!!!")
		if err == nil {
			t.Error("Expected error for invalid base64")
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		invalid := base64.URLEncoding.EncodeToString([]byte("not json"))
		_, err := ParseCursor(invalid)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

func TestInMemoryAuditStore_Query(t *testing.T) {
	store := NewInMemoryAuditStore()

	// Add test logs
	for i := 0; i < 10; i++ {
		log := NewAuditLog(&Actor{Type: "user", ID: "user-1"}, "action", &Resource{Type: "test", ID: "1"})
		log.Timestamp = time.Now().Add(-time.Duration(i) * time.Minute)
		store.Write(context.Background(), log)
	}

	t.Run("returns all logs without filter", func(t *testing.T) {
		query := &AuditQuery{Limit: 100}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(page.Items) != 10 {
			t.Errorf("Expected 10 items, got %d", len(page.Items))
		}
	})

	t.Run("limits results", func(t *testing.T) {
		query := &AuditQuery{Limit: 5}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(page.Items) != 5 {
			t.Errorf("Expected 5 items, got %d", len(page.Items))
		}

		if !page.HasMore {
			t.Error("Expected HasMore to be true")
		}
	})

	t.Run("filters by actor ID", func(t *testing.T) {
		// Add log with different actor
		log := NewAuditLog(&Actor{Type: "user", ID: "user-2"}, "action", &Resource{Type: "test", ID: "1"})
		store.Write(context.Background(), log)

		query := &AuditQuery{ActorID: "user-1", Limit: 100}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		for _, item := range page.Items {
			if item.Actor.ID != "user-1" {
				t.Errorf("Expected actor ID user-1, got %s", item.Actor.ID)
			}
		}
	})

	t.Run("filters by action", func(t *testing.T) {
		log := NewAuditLog(nil, "special-action", &Resource{Type: "test", ID: "1"})
		store.Write(context.Background(), log)

		query := &AuditQuery{Action: "special-action", Limit: 100}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(page.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(page.Items))
		}
	})

	t.Run("generates next cursor", func(t *testing.T) {
		query := &AuditQuery{Limit: 3}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if page.NextCursor == "" {
			t.Error("Expected NextCursor to be set when HasMore is true")
		}
	})

	t.Run("does not generate cursor when no more results", func(t *testing.T) {
		query := &AuditQuery{Limit: 100}
		page, err := store.Query(context.Background(), query)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if page.NextCursor != "" {
			t.Error("Expected NextCursor to be empty when no more results")
		}
	})
}

func TestInMemoryAuditStore_GetByID(t *testing.T) {
	store := NewInMemoryAuditStore()

	log := NewAuditLog(nil, "test", &Resource{Type: "test", ID: "1"})
	store.Write(context.Background(), log)

	t.Run("retrieves existing log", func(t *testing.T) {
		retrieved, err := store.GetByID(context.Background(), log.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if retrieved.ID != log.ID {
			t.Errorf("Expected ID %s, got %s", log.ID, retrieved.ID)
		}
	})

	t.Run("returns error for non-existent log", func(t *testing.T) {
		_, err := store.GetByID(context.Background(), "non-existent")
		if err == nil {
			t.Error("Expected error for non-existent log")
		}
	})
}

func TestInMemoryAuditStore_GetByRequestID(t *testing.T) {
	store := NewInMemoryAuditStore()

	requestID := "req-123"

	// Add logs with same request ID
	for i := 0; i < 3; i++ {
		log := NewAuditLog(nil, "step", &Resource{Type: "test", ID: "1"})
		log.RequestID = requestID
		store.Write(context.Background(), log)
	}

	// Add log with different request ID
	log := NewAuditLog(nil, "other", &Resource{Type: "test", ID: "1"})
	log.RequestID = "req-456"
	store.Write(context.Background(), log)

	t.Run("retrieves logs by request ID", func(t *testing.T) {
		logs, err := store.GetByRequestID(context.Background(), requestID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(logs) != 3 {
			t.Errorf("Expected 3 logs, got %d", len(logs))
		}

		for _, l := range logs {
			if l.RequestID != requestID {
				t.Errorf("Expected request ID %s, got %s", requestID, l.RequestID)
			}
		}
	})
}

func TestAuditLogger_Log(t *testing.T) {
	store := NewInMemoryAuditStore()
	logger := NewAuditLogger(store)

	actor := &Actor{Type: "user", ID: "user-1", Name: "Test User"}
	resource := &Resource{Type: "provider", ID: "openai", Name: "OpenAI"}

	log := logger.Log(context.Background(), actor, "created", resource)

	t.Run("creates audit log", func(t *testing.T) {
		if log == nil {
			t.Fatal("Expected log to be returned")
		}

		if log.ID == "" {
			t.Error("Expected ID to be set")
		}
	})

	t.Run("stores log in store", func(t *testing.T) {
		retrieved, err := store.GetByID(context.Background(), log.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if retrieved.ID != log.ID {
			t.Errorf("Expected ID %s, got %s", log.ID, retrieved.ID)
		}
	})
}

func TestAuditLogger_LogProviderCall(t *testing.T) {
	store := NewInMemoryAuditStore()
	logger := NewAuditLogger(store)

	actor := &Actor{Type: "user", ID: "user-1"}
	log := logger.LogProviderCall(context.Background(), actor, "openai", "gpt-4", "req-123", 150, "success")

	t.Run("creates provider call log", func(t *testing.T) {
		if log == nil {
			t.Fatal("Expected log to be returned")
		}

		if log.Resource.Type != "provider" {
			t.Errorf("Expected resource type provider, got %s", log.Resource.Type)
		}
	})

	t.Run("sets request ID", func(t *testing.T) {
		if log.RequestID != "req-123" {
			t.Errorf("Expected request ID req-123, got %s", log.RequestID)
		}
	})

	t.Run("sets latency context", func(t *testing.T) {
		if log.Context["latency_ms"] != int64(150) {
			t.Errorf("Expected latency_ms=150, got %v", log.Context["latency_ms"])
		}
	})
}

func TestAuditLogger_LogBudgetChange(t *testing.T) {
	store := NewInMemoryAuditStore()
	logger := NewAuditLogger(store)

	actor := &Actor{Type: "user", ID: "user-1"}
	log := logger.LogBudgetChange(context.Background(), actor, "budget-1", 100.0, 150.0)

	t.Run("creates budget change log", func(t *testing.T) {
		if log == nil {
			t.Fatal("Expected log to be returned")
		}

		if log.Action != "updated" {
			t.Errorf("Expected action updated, got %s", log.Action)
		}
	})

	t.Run("records change", func(t *testing.T) {
		change := log.Changes["balance"]
		if change == nil {
			t.Fatal("Expected balance change")
		}

		if change.OldValue != 100.0 {
			t.Errorf("Expected old value 100.0, got %v", change.OldValue)
		}

		if change.NewValue != 150.0 {
			t.Errorf("Expected new value 150.0, got %v", change.NewValue)
		}
	})
}

func TestAuditLogger_LogEvalRun(t *testing.T) {
	store := NewInMemoryAuditStore()
	logger := NewAuditLogger(store)

	t.Run("logs successful eval", func(t *testing.T) {
		log := logger.LogEvalRun(context.Background(), "suite-1", "coding-suite", true, 0.95)

		if log.Status != "success" {
			t.Errorf("Expected status success, got %s", log.Status)
		}

		if log.Context["score"] != 0.95 {
			t.Errorf("Expected score 0.95, got %v", log.Context["score"])
		}
	})

	t.Run("logs failed eval", func(t *testing.T) {
		log := logger.LogEvalRun(context.Background(), "suite-1", "coding-suite", false, 0.3)

		if log.Status != "failure" {
			t.Errorf("Expected status failure, got %s", log.Status)
		}

		if log.Error == "" {
			t.Error("Expected error to be set for failed eval")
		}
	})
}

func TestGlobalAuditStore(t *testing.T) {
	store := NewInMemoryAuditStore()
	InitGlobalAuditStore(store)

	t.Run("global store is initialized", func(t *testing.T) {
		retrieved := GetGlobalAuditStore()
		if retrieved == nil {
			t.Fatal("Expected global store to be initialized")
		}
	})

	t.Run("audit logger uses global store", func(t *testing.T) {
		logger := GetAuditLogger()
		if logger == nil {
			t.Fatal("Expected audit logger to be returned")
		}

		log := logger.Log(context.Background(), &Actor{Type: "system", ID: "test"}, "test", &Resource{Type: "test", ID: "1"})

		// Verify it was stored
		retrieved, err := store.GetByID(context.Background(), log.ID)
		if err != nil {
			t.Fatalf("Expected log to be stored: %v", err)
		}

		if retrieved.ID != log.ID {
			t.Errorf("Expected ID %s, got %s", log.ID, retrieved.ID)
		}
	})
}

func TestAuditPage(t *testing.T) {
	page := &AuditPage{
		Items:       make([]*AuditLog, 5),
		NextCursor:  "abc123",
		HasMore:     true,
		TotalCount:  100,
	}

	if len(page.Items) != 5 {
		t.Errorf("Expected 5 items, got %d", len(page.Items))
	}

	if !page.HasMore {
		t.Error("Expected HasMore to be true")
	}

	if page.TotalCount != 100 {
		t.Errorf("Expected TotalCount=100, got %d", page.TotalCount)
	}
}
