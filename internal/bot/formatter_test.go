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
	part1 := strings.Repeat("A", 2500)
	part2 := strings.Repeat("B", 2500)
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

func TestMarkdownToTelegramHTML_Basic(t *testing.T) {
	input := "## Header 2\nThis is **bold** and `code`."
	expected := "<b>Header 2</b>\nThis is <b>bold</b> and <code>code</code>."
	output := MarkdownToTelegramHTML(input)

	if output != expected {
		t.Fatalf("expected %q, got %q", expected, output)
	}
}

func TestMarkdownToTelegramHTML_CodeBlockWithSpecialChars(t *testing.T) {
	input := "Here is code:\n```go\nfunc main() {\n    if a < b && b > 0 {\n        fmt.Println(\"hi\")\n    }\n}\n```"
	output := MarkdownToTelegramHTML(input)

	if !strings.Contains(output, `<pre><code class="language-go">`) {
		t.Errorf("missing pre code tag: %s", output)
	}
	if !strings.Contains(output, `a &lt; b &amp;&amp; b &gt; 0`) {
		t.Errorf("special characters not escaped inside code block: %s", output)
	}
}

func TestMarkdownToTelegramHTML_LinksAndHeaders(t *testing.T) {
	input := "# Main Title\nCheck [Google](https://google.com) now."
	output := MarkdownToTelegramHTML(input)

	if !strings.Contains(output, "<b>Main Title</b>") {
		t.Errorf("header not converted to bold: %s", output)
	}
	if !strings.Contains(output, `<a href="https://google.com">Google</a>`) {
		t.Errorf("link not converted to anchor: %s", output)
	}
}

func TestMarkdownToTelegramHTML_ItalicsAndQuotes(t *testing.T) {
	input := "*italic star* and _italic under_\n> This is a quote.\nAlso variable_name_test should remain intact."
	output := MarkdownToTelegramHTML(input)

	if !strings.Contains(output, "<i>italic star</i>") {
		t.Errorf("asterisk italic failed: %s", output)
	}
	if !strings.Contains(output, "<i>italic under</i>") {
		t.Errorf("underscore italic failed: %s", output)
	}
	if !strings.Contains(output, "<blockquote>This is a quote.</blockquote>") {
		t.Errorf("blockquote conversion failed: %s", output)
	}
	if strings.Contains(output, "variable<i>name</i>test") {
		t.Errorf("snake_case incorrectly converted to italic: %s", output)
	}
}

func TestExtractInteractiveButtons(t *testing.T) {
	input := "Apakah Anda ingin melanjutkan penginstalan?\n\n[CONFIRM: Ya, Lanjutkan | Batalkan]"
	cleanText, buttons := ExtractInteractiveButtons(input)

	if len(buttons) != 2 {
		t.Fatalf("expected 2 buttons, got %d", len(buttons))
	}
	if buttons[0] != "Ya, Lanjutkan" || buttons[1] != "Batalkan" {
		t.Errorf("unexpected buttons parsed: %v", buttons)
	}
	if strings.Contains(cleanText, "[CONFIRM:") {
		t.Errorf("tag was not stripped from cleanText: %s", cleanText)
	}
	if cleanText != "Apakah Anda ingin melanjutkan penginstalan?" {
		t.Errorf("unexpected cleanText: %q", cleanText)
	}

	// Test no buttons
	noBtnInput := "Halo ini pesan biasa."
	text, emptyBtns := ExtractInteractiveButtons(noBtnInput)
	if len(emptyBtns) != 0 || text != noBtnInput {
		t.Errorf("expected no buttons on regular message, got %v", emptyBtns)
	}
}
