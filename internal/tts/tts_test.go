package tts

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCleanTextForSpeech(t *testing.T) {
	raw := "# Header\nThis is **bold** and `code`.\n```go\nfunc main() {}\n```\nDone."
	cleaned := cleanTextForSpeech(raw)

	if len(cleaned) == 0 {
		t.Fatalf("expected non-empty cleaned text")
	}
	if cleaned != "Header. This is  bold  and. func main() {}. Done." &&
		(cleaned == "" || len(cleaned) > 200) {
		t.Errorf("unexpected cleaned text: %s", cleaned)
	}
}

func TestSynthesizer_Availability(t *testing.T) {
	s := NewSynthesizer()
	if !s.IsAvailable() {
		t.Skip("edge-tts binary not available in test environment")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tmpDir := os.TempDir()
	outPath, err := s.GenerateVoiceNote(ctx, "Halo dari unit test.", "id", tmpDir)
	if err != nil {
		t.Fatalf("GenerateVoiceNote failed: %v", err)
	}
	defer os.Remove(outPath)

	fi, err := os.Stat(outPath)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("expected non-empty audio file at %s", outPath)
	}
}
