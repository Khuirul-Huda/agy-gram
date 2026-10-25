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

	s.SetLastDeliveredStepIndex(42)
	if s.GetLastDeliveredStepIndex() != 42 {
		t.Errorf("expected lastDeliveredStepIndex 42, got %d", s.GetLastDeliveredStepIndex())
	}

	s.Reset()
	if s.ShouldContinue() {
		t.Errorf("expected continueSession to be reset to false")
	}
	if s.GetConversationID() != "" {
		t.Errorf("expected conversationID to be empty after reset")
	}
	if s.GetLastDeliveredStepIndex() != 0 {
		t.Errorf("expected lastDeliveredStepIndex to be reset to 0, got %d", s.GetLastDeliveredStepIndex())
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
	s1.SetLastDeliveredStepIndex(77)

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
	if s2.GetLastDeliveredStepIndex() != 77 {
		t.Errorf("expected restored lastDeliveredStepIndex 77, got %d", s2.GetLastDeliveredStepIndex())
	}
}

func TestSessionForTopic(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "topic_test.db")
	store, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open bolt store: %v", err)
	}
	defer store.Close()

	root := NewSession("auto", "default-model", false, store)
	root.SetConversationID("root-conv")
	root.SetWorkspace("/home/root")

	// Thread 0 returns root
	if root.ForTopic(100, 0) != root {
		t.Errorf("expected thread 0 to return root session")
	}

	// Topic 42 has its own isolated conversation and workspace
	topic42 := root.ForTopic(100, 42)
	if topic42 == root {
		t.Errorf("expected topic 42 to be distinct from root")
	}
	topic42.SetConversationID("topic42-conv")
	topic42.SetWorkspace("/home/topic42")

	// Check isolation
	if root.GetConversationID() != "root-conv" {
		t.Errorf("expected root conv to remain 'root-conv', got '%s'", root.GetConversationID())
	}
	if topic42.GetConversationID() != "topic42-conv" {
		t.Errorf("expected topic42 conv to be 'topic42-conv', got '%s'", topic42.GetConversationID())
	}
	if root.GetWorkspace() != "/home/root" {
		t.Errorf("expected root workspace to remain '/home/root', got '%s'", root.GetWorkspace())
	}
	if topic42.GetWorkspace() != "/home/topic42" {
		t.Errorf("expected topic42 workspace to be '/home/topic42', got '%s'", topic42.GetWorkspace())
	}

	// Fetching same topic returns cached instance
	topic42Cached := root.ForTopic(100, 42)
	if topic42Cached != topic42 {
		t.Errorf("expected same instance from cache")
	}
}
