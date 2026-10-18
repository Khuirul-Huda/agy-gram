package agent

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TranscriptResponse represents a completed assistant reply parsed from transcript logs.
type TranscriptResponse struct {
	StepIndex int
	Content   string
}

// GetLatestAssistantResponse finds the most recent PLANNER_RESPONSE with user-facing content from the conversation transcript.
// If the content was truncated in transcript.jsonl, it retrieves the full text from transcript_full.jsonl.
func GetLatestAssistantResponse(convID string) (*TranscriptResponse, error) {
	if convID == "" {
		return nil, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs")
	transcriptPath := filepath.Join(baseDir, "transcript.jsonl")

	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var (
		latestStepIndex    int
		latestContent      string
		isContentTruncated bool
	)

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var step struct {
			StepIndex       int      `json:"step_index"`
			Source          string   `json:"source"`
			Type            string   `json:"type"`
			Status          string   `json:"status"`
			Content         string   `json:"content"`
			TruncatedFields []string `json:"truncated_fields"`
		}
		if err := json.Unmarshal(line, &step); err == nil {
			if step.Type == "PLANNER_RESPONSE" && step.Source == "MODEL" && strings.TrimSpace(step.Content) != "" {
				latestStepIndex = step.StepIndex
				latestContent = step.Content
				isContentTruncated = false
				for _, tf := range step.TruncatedFields {
					if tf == "content" {
						isContentTruncated = true
						break
					}
				}
			}
		}
	}

	if latestStepIndex == 0 || strings.TrimSpace(latestContent) == "" {
		return nil, nil
	}

	// If content was truncated in transcript.jsonl, read the complete content from transcript_full.jsonl
	if isContentTruncated {
		fullPath := filepath.Join(baseDir, "transcript_full.jsonl")
		if fullContent := readFullContentByStep(fullPath, latestStepIndex); fullContent != "" {
			latestContent = fullContent
		}
	}

	return &TranscriptResponse{
		StepIndex: latestStepIndex,
		Content:   latestContent,
	}, nil
}

func readFullContentByStep(filePath string, targetStep int) string {
	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var step struct {
			StepIndex int    `json:"step_index"`
			Content   string `json:"content"`
		}
		if err := json.Unmarshal(line, &step); err == nil {
			if step.StepIndex == targetStep && strings.TrimSpace(step.Content) != "" {
				return step.Content
			}
		}
	}
	return ""
}
