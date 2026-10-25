package storage

import (
	"path/filepath"
	"testing"
)

func TestBoltStore_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer store.Close()

	// Initial load should be nil
	state, err := store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state, got %+v", state)
	}

	// Save session
	testState := SessionState{
		ConversationID:  "conv-12345",
		ContinueSession: true,
		Mode:            "plan",
		Model:           "gemini-2.5-pro",
		Effort:          "high",
		Language:        "id",
		ThinkingEnabled:        true,
		LastUserID:             987654321,
		LastDeliveredStepIndex: 123,
	}

	if err := store.SaveSession(testState); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	// Load session and verify
	loaded, err := store.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession after save failed: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected loaded session, got nil")
	}

	if loaded.ConversationID != testState.ConversationID {
		t.Errorf("ConversationID mismatch: got %s, want %s", loaded.ConversationID, testState.ConversationID)
	}
	if loaded.ContinueSession != testState.ContinueSession {
		t.Errorf("ContinueSession mismatch: got %v, want %v", loaded.ContinueSession, testState.ContinueSession)
	}
	if loaded.Mode != testState.Mode {
		t.Errorf("Mode mismatch: got %s, want %s", loaded.Mode, testState.Mode)
	}
	if loaded.Model != testState.Model {
		t.Errorf("Model mismatch: got %s, want %s", loaded.Model, testState.Model)
	}
	if loaded.Effort != testState.Effort {
		t.Errorf("Effort mismatch: got %s, want %s", loaded.Effort, testState.Effort)
	}
	if loaded.Language != testState.Language {
		t.Errorf("Language mismatch: got %s, want %s", loaded.Language, testState.Language)
	}
	if loaded.ThinkingEnabled != testState.ThinkingEnabled {
		t.Errorf("ThinkingEnabled mismatch: got %v, want %v", loaded.ThinkingEnabled, testState.ThinkingEnabled)
	}
	if loaded.LastUserID != testState.LastUserID {
		t.Errorf("LastUserID mismatch: got %d, want %d", loaded.LastUserID, testState.LastUserID)
	}
	if loaded.LastDeliveredStepIndex != testState.LastDeliveredStepIndex {
		t.Errorf("LastDeliveredStepIndex mismatch: got %d, want %d", loaded.LastDeliveredStepIndex, testState.LastDeliveredStepIndex)
	}

	// Verify active turn lifecycle
	activeTurn := ActiveTurn{
		ChatID:         123,
		ThreadID:       45,
		MessageID:      678,
		ConversationID: "conv-12345",
		Prompt:         "Test turn",
	}
	if err := store.SaveActiveTurn(activeTurn); err != nil {
		t.Fatalf("SaveActiveTurn failed: %v", err)
	}
	loadedTurn, err := store.LoadActiveTurn()
	if err != nil || loadedTurn == nil {
		t.Fatalf("LoadActiveTurn failed: %v", err)
	}
	if loadedTurn.MessageID != 678 || loadedTurn.ConversationID != "conv-12345" {
		t.Errorf("unexpected loaded turn: %+v", loadedTurn)
	}
	if err := store.ClearActiveTurn(); err != nil {
		t.Fatalf("ClearActiveTurn failed: %v", err)
	}
	clearedTurn, err := store.LoadActiveTurn()
	if err != nil || clearedTurn != nil {
		t.Errorf("expected nil turn after clear, got %+v", clearedTurn)
	}
}
