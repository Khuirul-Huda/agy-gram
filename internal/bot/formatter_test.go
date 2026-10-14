package bot

import (
	"strings"
	"testing"
)

func TestSplitMessage_ShortMessage(t *testing.T) {
	text := "Hello world, this is a short response."
	chunks := SplitMessage(text)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != text {
		t.Fatalf("expected '%s', got '%s'", text, chunks[0])
	}
}

func TestSplitMessage_EmptyMessage(t *testing.T) {
	chunks := SplitMessage("   ")
	if len(chunks) != 1 || chunks[0] != "(Empty response)" {
		t.Fatalf("expected '(Empty response)', got %v", chunks)
	}
}

func TestSplitMessage_LongMessage(t *testing.T) {
	// Construct a message longer than 4000 characters
	part1 := strings.Repeat("A", 3000)
	part2 := strings.Repeat("B", 3000)
	longText := part1 + "\n\n" + part2

	chunks := SplitMessage(longText)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if len(chunk) > 4000 {
			t.Errorf("chunk %d exceeded limit: length %d", i, len(chunk))
		}
	}
}
