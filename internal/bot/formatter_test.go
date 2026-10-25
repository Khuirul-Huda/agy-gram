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
	input := "# Main Title\nCheck [Google](https://google.com) and [example](file://path/) and [`Session`](file:///path/to/session.go#L10) now."
	output := MarkdownToTelegramHTML(input)

	if !strings.Contains(output, "<b>Main Title</b>") {
		t.Errorf("header not converted to bold: %s", output)
	}
	if !strings.Contains(output, `<a href="https://google.com">Google</a>`) {
		t.Errorf("link not converted to anchor: %s", output)
	}
	if !strings.Contains(output, `<code>example</code>`) {
		t.Errorf("file link not converted to code: %s", output)
	}
	if !strings.Contains(output, `<code>Session</code>`) {
		t.Errorf("backticked file link not converted to code: %s", output)
	}
	if strings.Contains(output, `<code><code>`) {
		t.Errorf("found nested code tags in output: %s", output)
	}
	if strings.Contains(output, `file://`) {
		t.Errorf("file link protocol still present in output: %s", output)
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

func TestFormatMarkdownTables_Basic(t *testing.T) {
	input := `Tabel Spesifikasi:
| Parameter | Nilai |
| --- | --- |
| CPU | Celeron N2840 |
| RAM | 1.8 GiB |`

	output := FormatMarkdownTables(input)
	if !strings.Contains(output, "• **CPU:** Celeron N2840") {
		t.Errorf("expected 2-column list row 1, got: %s", output)
	}
	if !strings.Contains(output, "• **RAM:** 1.8 GiB") {
		t.Errorf("expected 2-column list row 2, got: %s", output)
	}
}

func TestFormatMarkdownTables_3ColumnsCard(t *testing.T) {
	input := `| Layanan | Status | Port |
| :--- | :---: | ---: |
| NextDNS | Aktif | 53 |
| MediaMTX | Aktif | 8554 |`

	output := FormatMarkdownTables(input)
	if !strings.Contains(output, "🔹 **NextDNS**") {
		t.Errorf("expected card 1 title, got: %s", output)
	}
	if !strings.Contains(output, "• **Status:** Aktif") || !strings.Contains(output, "• **Port:** 53") {
		t.Errorf("expected card 1 attributes, got: %s", output)
	}
	if !strings.Contains(output, "🔹 **MediaMTX**") {
		t.Errorf("expected card 2 title, got: %s", output)
	}
}

func TestFormatMarkdownTables_InsideCodeBlock(t *testing.T) {
	input := "```markdown\n| Kolom1 | Kolom2 |\n| --- | --- |\n| 1 | 2 |\n```"
	output := FormatMarkdownTables(input)

	if strings.Contains(output, "• **1:**") || strings.Contains(output, "🔹") {
		t.Errorf("table inside code block should not be converted, got: %s", output)
	}
	if output != input {
		t.Errorf("code block content was modified: %s", output)
	}
}

func TestFormatMarkdownTables_UnicodeAndEmojis(t *testing.T) {
	input := `| Status | Keterangan |
| :--- | :--- |
| ✅ Aktif | Berjalan normal |
| ❌ Mati | Layanan mati |`

	output := FormatMarkdownTables(input)
	if !strings.Contains(output, "• **✅ Aktif:** Berjalan normal") {
		t.Errorf("emoji row 1 mismatch: %s", output)
	}
	if !strings.Contains(output, "• **❌ Mati:** Layanan mati") {
		t.Errorf("emoji row 2 mismatch: %s", output)
	}
}

func TestMarkdownToTelegramHTML_WithTable(t *testing.T) {
	input := "Berikut hasil perbandingan:\n\n| Nama | Skor |\n| --- | --- |\n| Alpha | 90 |\n| Beta | 85 |\n\nSelesai."
	output := MarkdownToTelegramHTML(input)

	if !strings.Contains(output, "• <b>Alpha:</b> 90") || !strings.Contains(output, "• <b>Beta:</b> 85") {
		t.Errorf("expected bold html tags in converted table list, got: %s", output)
	}
	if !strings.Contains(output, "Berikut hasil perbandingan:") || !strings.Contains(output, "Selesai.") {
		t.Errorf("surrounding text was lost: %s", output)
	}
}

