package bot

import (
	"strings"
)

const maxMessageLength = 4000

// SplitMessage chunks a long text message so it conforms to Telegram's 4096 character limit.
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

	for len(remaining) > maxMessageLength {
		cutIndex := findOptimalCutPoint(remaining[:maxMessageLength])
		if cutIndex <= 0 {
			cutIndex = maxMessageLength
		}

		chunk := strings.TrimSpace(remaining[:cutIndex])
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		remaining = strings.TrimSpace(remaining[cutIndex:])
	}

	if remaining != "" {
		chunks = append(chunks, remaining)
	}

	return chunks
}

func findOptimalCutPoint(chunk string) int {
	// Prefer splitting at paragraph breaks
	if idx := strings.LastIndex(chunk, "\n\n"); idx > maxMessageLength/2 {
		return idx + 2
	}
	// Next prefer splitting at newline
	if idx := strings.LastIndex(chunk, "\n"); idx > maxMessageLength/2 {
		return idx + 1
	}
	// Otherwise split at whitespace
	if idx := strings.LastIndex(chunk, " "); idx > maxMessageLength/2 {
		return idx + 1
	}
	return -1
}
