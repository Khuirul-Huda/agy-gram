package agent

import (
	"path/filepath"
	"testing"

	"github.com/Khuirul-Huda/agy-gram/internal/storage"
)

func TestSessionLifecycle(t *testing.T) {
	s := NewSession("auto", "", false)

	if s.ShouldContinue() {
		t.Errorf("expected initial continueSession to be false")
	}
	if s.GetMode() != "auto" {
		t.Errorf("expected default mode 'auto', got '%s'", s.GetMode())
	}
	if s.IsThinkingEnabled() {
		t.Errorf("expected default thinkingEnabled to be false")
	}

	s.SetContinue(true)
	if !s.ShouldContinue() {
		t.Errorf("expected continueSession to be true")
	}

	s.SetConversationID("test-conv-123")
	if s.GetConversationID() != "test-conv-123" {
		t.Errorf("expected conversationID 'test-conv-123', got '%s'", s.GetConversationID())
	}

	s.SetLastUserID(123456)
	if s.GetLastUserID() != 123456 {
		t.Errorf("expected lastUserID 123456, got %d", s.GetLastUserID())
	}

	s.SetMode("plan")
	if s.GetMode() != "plan" {
		t.Errorf("expected mode 'plan', got '%s'", s.GetMode())
	}

	s.SetModel("gemini-2.5-pro")
	if s.GetModel() != "gemini-2.5-pro" {
		t.Errorf("expected model 'gemini-2.5-pro', got '%s'", s.GetModel())
	}

	s.SetEffort("high")
	if s.GetEffort() != "high" {
		t.Errorf("expected effort 'high', got '%s'", s.GetEffort())
	}

	s.SetThinkingEnabled(true)
	if !s.IsThinkingEnabled() {
		t.Errorf("expected thinkingEnabled to be true")
	}

	if s.GetWorkspace() != "" {
		t.Errorf("expected empty initial workspace, got '%s'", s.GetWorkspace())
	}
	s.SetWorkspace("/home/huda/repo/Mocci")
	if s.GetWorkspace() != "/home/huda/repo/Mocci" {
		t.Errorf("expected workspace '/home/huda/repo/Mocci', got '%s'", s.GetWorkspace())
	}

	if s.GetLanguage() != "id" {
		t.Errorf("expected default language 'id', got '%s'", s.GetLanguage())
	}
	s.SetLanguage("en")
	if s.GetLanguage() != "en" {
		t.Errorf("expected language 'en', got '%s'", s.GetLanguage())
	}

	s.Reset()
	if s.ShouldContinue() {
		t.Errorf("expected continueSession to be reset to false")
	}
	if s.GetConversationID() != "" {
		t.Errorf("expected conversationID to be empty after reset")
	}
}

func TestSessionPersistence(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_session.db")

	store1, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Session 1 writes state
	s1 := NewSession("auto", "default-model", false, store1)
	s1.SetConversationID("session-uuid-456")
	s1.SetContinue(true)
	s1.SetMode("accept-edits")
	s1.SetModel("claude-3-7-sonnet")
	s1.SetEffort("medium")
	s1.SetLanguage("en")
	s1.SetThinkingEnabled(true)
	s1.SetLastUserID(999888)

	_ = store1.Close()

	// Reopen store from same DB file
	store2, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer store2.Close()

	// Session 2 restores state from bbolt store
	s2 := NewSession("auto", "", false, store2)
	if s2.GetConversationID() != "session-uuid-456" {
		t.Errorf("expected restored convID 'session-uuid-456', got '%s'", s2.GetConversationID())
	}
	if !s2.ShouldContinue() {
		t.Errorf("expected restored continue to be true")
	}
	if s2.GetMode() != "accept-edits" {
		t.Errorf("expected restored mode 'accept-edits', got '%s'", s2.GetMode())
	}
	if s2.GetModel() != "claude-3-7-sonnet" {
		t.Errorf("expected restored model 'claude-3-7-sonnet', got '%s'", s2.GetModel())
	}
	if s2.GetEffort() != "medium" {
		t.Errorf("expected restored effort 'medium', got '%s'", s2.GetEffort())
	}
	if s2.GetLanguage() != "en" {
		t.Errorf("expected restored language 'en', got '%s'", s2.GetLanguage())
	}
	if !s2.IsThinkingEnabled() {
		t.Errorf("expected restored thinkingEnabled to be true")
	}
	if s2.GetLastUserID() != 999888 {
		t.Errorf("expected restored lastUserID 999888, got %d", s2.GetLastUserID())
	}
}
