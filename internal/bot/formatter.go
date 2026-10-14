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
	linkRegex        = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\)]+)\)`)
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

	// 9. Links ([title](url) -> <a href="url">title</a>)
	processed = linkRegex.ReplaceAllString(processed, `<a href="$2">$1</a>`)

	// 10. Restore inline code blocks
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
