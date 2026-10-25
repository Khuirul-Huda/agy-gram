package bot

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

const maxMessageLength = 3800

var (
	headerRegex      = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
	boldRegex        = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRegex      = regexp.MustCompile(`\*([^\*\s\n](?:[^\*\n]*?[^\*\s\n])?)\*`)
	italicUnderRegex = regexp.MustCompile(`(^|[\s(])_([^_\s\n](?:[^_\n]*?[^_\s\n])?)_($|[\s).,!?])`)
	blockquoteRegex  = regexp.MustCompile(`(?m)^&gt;\s*(.+)$`)
	strikeRegex      = regexp.MustCompile(`~~(.+?)~~`)
	fileLinkRegex    = regexp.MustCompile(`\[([^\]]+)\]\(file://[^\)]*\)`)
	linkRegex        = regexp.MustCompile(`\[([^\]]+)\]\(((?:https?|tg)://[^\)]+)\)`)
	inlineRegex      = regexp.MustCompile("`([^`\n]+)`")
)

// SplitMessage chunks a long text message into safe sizes for Telegram (limit: 4096).
// It also balances open code blocks across chunks to avoid unclosed formatting.
func SplitMessage(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{"(Empty response)"}
	}

	if len(text) <= maxMessageLength {
		return []string{text}
	}

	var chunks []string
	remaining := text
	inCodeBlock := false
	codeBlockLang := ""

	for len(remaining) > maxMessageLength {
		cutIndex := findOptimalCutPoint(remaining[:maxMessageLength])
		if cutIndex <= 0 {
			cutIndex = maxMessageLength
		}

		part := remaining[:cutIndex]

		// Check code fence parity in this part
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "```") {
				if inCodeBlock {
					inCodeBlock = false
					codeBlockLang = ""
				} else {
					inCodeBlock = true
					codeBlockLang = strings.TrimPrefix(trimmed, "```")
				}
			}
		}

		chunk := strings.TrimSpace(part)
		if inCodeBlock {
			chunk += "\n```"
		}

		if chunk != "" {
			chunks = append(chunks, chunk)
		}

		rest := strings.TrimSpace(remaining[cutIndex:])
		if inCodeBlock && rest != "" {
			rest = "```" + codeBlockLang + "\n" + rest
		}
		remaining = rest
	}

	if remaining != "" {
		chunks = append(chunks, remaining)
	}

	return chunks
}

func findOptimalCutPoint(chunk string) int {
	if idx := strings.LastIndex(chunk, "\n\n"); idx > maxMessageLength/2 {
		return idx + 2
	}
	if idx := strings.LastIndex(chunk, "\n"); idx > maxMessageLength/2 {
		return idx + 1
	}
	if idx := strings.LastIndex(chunk, " "); idx > maxMessageLength/2 {
		return idx + 1
	}
	return -1
}

// MarkdownToTelegramHTML converts CommonMark/GitHub-flavored markdown into Telegram-compatible HTML.
func MarkdownToTelegramHTML(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	input = FormatMarkdownTables(input)

	// 1. Separate code blocks to protect them from inline formatting
	var codeBlocks []string
	codeBlockPlaceholder := "\x00CB%d\x00"

	fencedCodeRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\n?(.*?)```")
	processed := fencedCodeRegex.ReplaceAllStringFunc(input, func(m string) string {
		match := fencedCodeRegex.FindStringSubmatch(m)
		lang := ""
		code := ""
		if len(match) > 1 {
			lang = strings.TrimSpace(match[1])
		}
		if len(match) > 2 {
			code = match[2]
		}

		escapedCode := html.EscapeString(code)
		var blockHTML string
		if lang != "" {
			blockHTML = fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, html.EscapeString(lang), escapedCode)
		} else {
			blockHTML = fmt.Sprintf("<pre><code>%s</code></pre>", escapedCode)
		}

		idx := len(codeBlocks)
		codeBlocks = append(codeBlocks, blockHTML)
		return fmt.Sprintf(codeBlockPlaceholder, idx)
	})

	// 2. Separate inline code to protect from inline formatting
	var inlineCodes []string
	inlinePlaceholder := "\x00IC%d\x00"

	processed = inlineRegex.ReplaceAllStringFunc(processed, func(m string) string {
		match := inlineRegex.FindStringSubmatch(m)
		code := match[1]
		escaped := html.EscapeString(code)
		idx := len(inlineCodes)
		inlineCodes = append(inlineCodes, fmt.Sprintf("<code>%s</code>", escaped))
		return fmt.Sprintf(inlinePlaceholder, idx)
	})

	// 3. Escape raw HTML entities in regular text
	processed = html.EscapeString(processed)

	// 4. Headers (# Title -> <b>Title</b>)
	processed = headerRegex.ReplaceAllString(processed, "<b>$1</b>")

	// 5. Bold (**text** -> <b>text</b>)
	processed = boldRegex.ReplaceAllString(processed, "<b>$1</b>")

	// 6. Italic (*text* or _text_ -> <i>text</i>)
	processed = italicRegex.ReplaceAllString(processed, "<i>$1</i>")
	processed = italicUnderRegex.ReplaceAllString(processed, "$1<i>$2</i>$3")

	// 7. Blockquotes (> quote -> <blockquote>quote</blockquote>)
	processed = blockquoteRegex.ReplaceAllString(processed, "<blockquote>$1</blockquote>")

	// 8. Strikethrough (~~text~~ -> <s>text</s>)
	processed = strikeRegex.ReplaceAllString(processed, "<s>$1</s>")

	// 9. Links
	// Convert file:// links to code elements since Telegram does not support the file:// protocol
	processed = fileLinkRegex.ReplaceAllStringFunc(processed, func(m string) string {
		match := fileLinkRegex.FindStringSubmatch(m)
		if len(match) < 2 {
			return m
		}
		title := match[1]
		if strings.Contains(title, "\x00IC") || strings.HasPrefix(title, "&lt;code&gt;") || strings.HasPrefix(title, "<code>") {
			return title
		}
		return fmt.Sprintf("<code>%s</code>", title)
	})

	// Web links ([title](url) -> <a href="url">title</a>)
	processed = linkRegex.ReplaceAllString(processed, `<a href="$2">$1</a>`)

	// 10. Restore standard safe Telegram HTML tags
	safeTagsReplacer := strings.NewReplacer(
		"&lt;i&gt;", "<i>",
		"&lt;/i&gt;", "</i>",
		"&lt;b&gt;", "<b>",
		"&lt;/b&gt;", "</b>",
		"&lt;s&gt;", "<s>",
		"&lt;/s&gt;", "</s>",
		"&lt;u&gt;", "<u>",
		"&lt;/u&gt;", "</u>",
		"&lt;blockquote&gt;", "<blockquote>",
		"&lt;/blockquote&gt;", "</blockquote>",
		"&lt;blockquote expandable&gt;", "<blockquote expandable>",
	)
	processed = safeTagsReplacer.Replace(processed)

	// 11. Restore inline code blocks
	for i, ic := range inlineCodes {
		placeholder := html.EscapeString(fmt.Sprintf(inlinePlaceholder, i))
		processed = strings.ReplaceAll(processed, placeholder, ic)
		// in case it wasn't escaped
		processed = strings.ReplaceAll(processed, fmt.Sprintf(inlinePlaceholder, i), ic)
	}

	// 9. Restore fenced code blocks
	for i, cb := range codeBlocks {
		placeholder := html.EscapeString(fmt.Sprintf(codeBlockPlaceholder, i))
		processed = strings.ReplaceAll(processed, placeholder, cb)
		// in case it wasn't escaped
		processed = strings.ReplaceAll(processed, fmt.Sprintf(codeBlockPlaceholder, i), cb)
	}

	return processed
}

var buttonsRegex = regexp.MustCompile(`(?i)\[(?:BUTTONS|CONFIRM):\s*([^\]]+)\]`)

// ExtractInteractiveButtons parses [BUTTONS: opt1 | opt2] or [CONFIRM: opt1 | opt2] from text.
// It returns the cleaned text without the tag, and the list of button labels.
func ExtractInteractiveButtons(text string) (string, []string) {
	match := buttonsRegex.FindStringSubmatch(text)
	if match == nil {
		return text, nil
	}

	rawButtons := match[1]
	parts := strings.Split(rawButtons, "|")
	var buttons []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			buttons = append(buttons, trimmed)
		}
	}

	cleanText := strings.TrimSpace(buttonsRegex.ReplaceAllString(text, ""))
	return cleanText, buttons
}

var (
	stripBoldRegex   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	stripItalicRegex = regexp.MustCompile(`\*([^\*\s\n](?:[^\*\n]*?[^\*\s\n])?)\*`)
	stripStrikeRegex = regexp.MustCompile(`~~(.+?)~~`)
	stripLinkRegex   = regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
	stripCodeRegex   = regexp.MustCompile("`([^`\n]+)`")
)

const escapedPipePlaceholder = "\x00PIPE\x00"

// cleanCellMarkdown removes markdown styling inside table cells so they render neatly inside monospace blocks.
func cleanCellMarkdown(s string) string {
	s = stripBoldRegex.ReplaceAllString(s, "$1")
	s = stripItalicRegex.ReplaceAllString(s, "$1")
	s = stripStrikeRegex.ReplaceAllString(s, "$1")
	s = stripLinkRegex.ReplaceAllString(s, "$1")
	s = stripCodeRegex.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "<br>", " ")
	s = strings.ReplaceAll(s, "<br/>", " ")
	s = strings.ReplaceAll(s, "<br />", " ")
	return strings.TrimSpace(s)
}

func splitRowCells(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, `\|`, escapedPipePlaceholder)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	parts := strings.Split(line, "|")
	cells := make([]string, len(parts))
	for i, p := range parts {
		p = strings.ReplaceAll(p, escapedPipePlaceholder, "|")
		cells[i] = cleanCellMarkdown(p)
	}
	return cells
}

func isSeparatorRow(line string) bool {
	line = strings.TrimSpace(line)
	if !strings.Contains(line, "-") {
		return false
	}
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	parts := strings.Split(line, "|")
	if len(parts) == 0 {
		return false
	}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if len(trimmed) == 0 {
			return false
		}
		for _, r := range trimmed {
			if r != '-' && r != ':' && r != ' ' {
				return false
			}
		}
		if !strings.Contains(trimmed, "-") {
			return false
		}
	}
	return true
}

func parseAlignments(line string) []int {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = line[1:]
	}
	if strings.HasSuffix(line, "|") {
		line = line[:len(line)-1]
	}
	parts := strings.Split(line, "|")
	aligns := make([]int, len(parts))
	for i, p := range parts {
		trimmed := strings.TrimSpace(p)
		left := strings.HasPrefix(trimmed, ":")
		right := strings.HasSuffix(trimmed, ":")
		if left && right {
			aligns[i] = 1 // Center
		} else if right {
			aligns[i] = 2 // Right
		} else {
			aligns[i] = 0 // Left
		}
	}
	return aligns
}

func runeWidth(r rune) int {
	// Zero-width characters & variation selectors
	if r == 0x200B || r == 0x200C || r == 0x200D || r == 0xFEFF || (r >= 0xFE00 && r <= 0xFE0F) {
		return 0
	}
	// Combining characters
	if (r >= 0x0300 && r <= 0x036F) || (r >= 0x1AB0 && r <= 0x1AFF) || (r >= 0x1DC0 && r <= 0x1DFF) || (r >= 0x20D0 && r <= 0x20FF) {
		return 0
	}
	// Standard ASCII
	if r >= 0x20 && r <= 0x7E {
		return 1
	}
	// Wide characters / Emojis / CJK
	if (r >= 0x1100 && r <= 0x115F) ||
		(r >= 0x231A && r <= 0x231B) ||
		(r >= 0x23E9 && r <= 0x23EC) ||
		(r >= 0x23F0 && r <= 0x23F3) ||
		(r >= 0x25FD && r <= 0x25FE) ||
		(r >= 0x2600 && r <= 0x27BF) || // Dingbats & Misc symbols (✅, ⚠️, ❌, etc.)
		(r >= 0x2B50 && r <= 0x2B55) ||
		(r >= 0x2E80 && r <= 0xA4CF) || // CJK
		(r >= 0xAC00 && r <= 0xD7A3) || // Hangul
		(r >= 0xF900 && r <= 0xFAFF) ||
		(r >= 0xFE10 && r <= 0xFE19) ||
		(r >= 0xFE30 && r <= 0xFE6F) ||
		(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth
		(r >= 0xFFE0 && r <= 0xFFE6) ||
		(r >= 0x1F000 && r <= 0x1FAFF) { // Emojis
		return 2
	}
	return 1
}

func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func padString(s string, targetWidth int, align int) string {
	curW := stringWidth(s)
	if curW >= targetWidth {
		return s
	}
	diff := targetWidth - curW
	switch align {
	case 1: // Center
		left := diff / 2
		right := diff - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	case 2: // Right
		return strings.Repeat(" ", diff) + s
	default: // Left
		return s + strings.Repeat(" ", diff)
	}
}

// RenderTable renders table headers, alignments, and rows into a unicode box table.
func RenderTable(headers []string, aligns []int, rows [][]string) string {
	numCols := len(headers)
	for _, row := range rows {
		if len(row) > numCols {
			numCols = len(row)
		}
	}
	for len(headers) < numCols {
		headers = append(headers, "")
	}

	colWidths := make([]int, numCols)

	for i, h := range headers {
		w := stringWidth(h)
		if w > colWidths[i] {
			colWidths[i] = w
		}
	}

	for _, row := range rows {
		for i := 0; i < numCols && i < len(row); i++ {
			w := stringWidth(row[i])
			if w > colWidths[i] {
				colWidths[i] = w
			}
		}
	}

	var sb strings.Builder

	// Top border: ┌───┬───┐
	sb.WriteString("┌")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┬")
		}
	}
	sb.WriteString("┐\n")

	// Header row: │ Col1 │ Col2 │
	sb.WriteString("│")
	for i, h := range headers {
		sb.WriteString(" ")
		align := 0
		if i < len(aligns) {
			align = aligns[i]
		}
		sb.WriteString(padString(h, colWidths[i], align))
		sb.WriteString(" │")
	}
	sb.WriteString("\n")

	// Header separator: ├───┼───┤
	sb.WriteString("├")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┼")
		}
	}
	sb.WriteString("┤\n")

	// Data rows
	for _, row := range rows {
		sb.WriteString("│")
		for i := 0; i < numCols; i++ {
			sb.WriteString(" ")
			val := ""
			if i < len(row) {
				val = row[i]
			}
			align := 0
			if i < len(aligns) {
				align = aligns[i]
			}
			sb.WriteString(padString(val, colWidths[i], align))
			sb.WriteString(" │")
		}
		sb.WriteString("\n")
	}

	// Bottom border: └───┴───┘
	sb.WriteString("└")
	for i, w := range colWidths {
		sb.WriteString(strings.Repeat("─", w+2))
		if i < numCols-1 {
			sb.WriteString("┴")
		}
	}
	sb.WriteString("┘")

	return sb.String()
}

// RenderTableAsCard converts a markdown table into a beautifully formatted, mobile-friendly structured list/cards.
func RenderTableAsCard(headers []string, rows [][]string) string {
	if len(headers) == 0 || len(rows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Case 1: 1 Column
	if len(headers) == 1 {
		for i, row := range rows {
			val := ""
			if len(row) > 0 {
				val = row[0]
			}
			sb.WriteString(fmt.Sprintf("• %s", val))
			if i < len(rows)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	}

	// Case 2: 2 Columns (Key-Value)
	if len(headers) == 2 {
		for i, row := range rows {
			key := ""
			val := ""
			if len(row) > 0 {
				key = cleanCellMarkdown(row[0])
			}
			if len(row) > 1 {
				val = strings.TrimSpace(row[1])
			}
			if val != "" {
				sb.WriteString(fmt.Sprintf("• **%s:** %s", key, val))
			} else {
				sb.WriteString(fmt.Sprintf("• **%s**", key))
			}
			if i < len(rows)-1 {
				sb.WriteString("\n")
			}
		}
		return sb.String()
	}

	// Case 3: 3+ Columns (Structured Card)
	for i, row := range rows {
		title := ""
		if len(row) > 0 {
			title = cleanCellMarkdown(row[0])
		}
		if title != "" {
			sb.WriteString(fmt.Sprintf("🔹 **%s**\n", title))
		}

		for col := 1; col < len(headers); col++ {
			headerName := cleanCellMarkdown(headers[col])
			val := ""
			if col < len(row) {
				val = strings.TrimSpace(row[col])
			}
			if val == "" {
				val = "-"
			}
			sb.WriteString(fmt.Sprintf("  • **%s:** %s\n", headerName, val))
		}

		if i < len(rows)-1 {
			sb.WriteString("\n")
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatMarkdownTables detects markdown tables in text and converts them into mobile-friendly structured cards.
func FormatMarkdownTables(input string) string {
	if strings.TrimSpace(input) == "" {
		return input
	}

	// 1. Protect existing code blocks so tables inside them are not touched
	var codeBlocks []string
	codeBlockPlaceholder := "\x00TB_CB%d\x00"

	fencedCodeRegex := regexp.MustCompile("(?s)```([a-zA-Z0-9_-]*)\n?(.*?)```")
	processed := fencedCodeRegex.ReplaceAllStringFunc(input, func(m string) string {
		idx := len(codeBlocks)
		codeBlocks = append(codeBlocks, m)
		return fmt.Sprintf(codeBlockPlaceholder, idx)
	})

	lines := strings.Split(processed, "\n")
	var newLines []string

	for i := 0; i < len(lines); {
		// Check if line i could be a header and line i+1 a separator
		if i+1 < len(lines) && strings.Contains(lines[i], "|") && isSeparatorRow(lines[i+1]) {
			headerLine := lines[i]
			headers := splitRowCells(headerLine)

			// Collect data rows
			var rows [][]string
			j := i + 2
			for j < len(lines) {
				curLine := strings.TrimSpace(lines[j])
				if curLine == "" || !strings.Contains(curLine, "|") || isSeparatorRow(curLine) {
					break
				}
				rows = append(rows, splitRowCells(lines[j]))
				j++
			}

			if len(headers) > 0 && len(rows) > 0 {
				renderedCards := RenderTableAsCard(headers, rows)
				newLines = append(newLines, renderedCards)
				i = j
				continue
			}
		}

		newLines = append(newLines, lines[i])
		i++
	}

	result := strings.Join(newLines, "\n")

	// Restore protected code blocks
	for idx, cb := range codeBlocks {
		placeholder := fmt.Sprintf(codeBlockPlaceholder, idx)
		result = strings.ReplaceAll(result, placeholder, cb)
	}

	return result
}

