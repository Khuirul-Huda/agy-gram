package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLatestAssistantResponse(t *testing.T) {
	// 1. Test empty conversation ID
	resp, err := GetLatestAssistantResponse("")
	if err != nil {
		t.Fatalf("expected nil error for empty convID, got: %v", err)
	}
	if resp != nil {
		t.Fatalf("expected nil resp for empty convID, got: %+v", resp)
	}

	// 2. Test mock transcript directory
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get user home: %v", err)
	}

	testConvID := "test-recovery-conv-999"
	logDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain", testConvID, ".system_generated", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("failed to create mock log dir: %v", err)
	}
	defer os.RemoveAll(filepath.Join(home, ".gemini", "antigravity-cli", "brain", testConvID))

	mockTranscript := `{"step_index":1,"source":"USER_EXPLICIT","type":"USER_INPUT","content":"halo agy"}
{"step_index":2,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","content":"Halo! Ada yang bisa saya bantu?"}
{"step_index":3,"source":"USER_EXPLICIT","type":"USER_INPUT","content":"buatkan tabel"}
{"step_index":4,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","content":"Ini tabel yang Anda minta:\n| A | B |\n|---|---|\n| 1 | 2 |"}
`
	if err := os.WriteFile(filepath.Join(logDir, "transcript.jsonl"), []byte(mockTranscript), 0644); err != nil {
		t.Fatalf("failed to write mock transcript: %v", err)
	}

	latest, err := GetLatestAssistantResponse(testConvID)
	if err != nil {
		t.Fatalf("GetLatestAssistantResponse failed: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest response, got nil")
	}
	if latest.StepIndex != 4 {
		t.Errorf("expected latest step index 4, got %d", latest.StepIndex)
	}
	if latest.Content != "Ini tabel yang Anda minta:\n| A | B |\n|---|---|\n| 1 | 2 |" {
		t.Errorf("unexpected content: %q", latest.Content)
	}
}
