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
		ThinkingEnabled: true,
		LastUserID:      987654321,
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
}
