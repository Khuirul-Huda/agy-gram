package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatDiff(t *testing.T) {
	before := "func hello() {\n\tprintln(\"old\")\n}"
	after := "func hello() {\n\tprintln(\"new\")\n}"

	diff := formatDiff(before, after)

	if !strings.Contains(diff, "--- Sebelum") {
		t.Errorf("expected diff to contain '--- Sebelum', got:\n%s", diff)
	}
	if !strings.Contains(diff, "- func hello()") {
		t.Errorf("expected diff to contain '- func hello()', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+++ Sesudah") {
		t.Errorf("expected diff to contain '+++ Sesudah', got:\n%s", diff)
	}
	if !strings.Contains(diff, "+ func hello()") {
		t.Errorf("expected diff to contain '+ func hello()', got:\n%s", diff)
	}
}

func TestFormatWriteContent(t *testing.T) {
	content := "line 1\nline 2\nline 3"
	formatted := formatWriteContent(content)

	if !strings.HasPrefix(formatted, "+++ File Baru") {
		t.Errorf("expected formatted write to start with '+++ File Baru', got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "+ line 1") {
		t.Errorf("expected formatted write to contain '+ line 1', got:\n%s", formatted)
	}
}

func TestReadToolCallFromLog(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "transcript_full.jsonl")

	sampleLog := `{"step_index":10,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","tool_calls":[{"name":"replace_file_content","args":{"AllowMultiple":false,"TargetFile":"/path/to/test.go","Instruction":"Fix bug","TargetContent":"oldCode()","ReplacementContent":"newCode()"}}]}` + "\n"

	if err := os.WriteFile(logPath, []byte(sampleLog), 0644); err != nil {
		t.Fatalf("failed to write test log file: %v", err)
	}

	args := readToolCallFromLog(logPath, 11, "replace_file_content")
	if args == nil {
		t.Fatalf("expected non-nil tool args")
	}

	targetFile := getToolArgString(args, "TargetFile")
	if targetFile != "/path/to/test.go" {
		t.Errorf("expected '/path/to/test.go', got '%s'", targetFile)
	}

	targetContent := getToolArgString(args, "TargetContent")
	if targetContent != "oldCode()" {
		t.Errorf("expected 'oldCode()', got '%s'", targetContent)
	}

	replacementContent := getToolArgString(args, "ReplacementContent")
	if replacementContent != "newCode()" {
		t.Errorf("expected 'newCode()', got '%s'", replacementContent)
	}
}

func TestParseQuotaJSON(t *testing.T) {
	sampleJSON := `{
		"conversation_id":"",
		"status":"SUCCESS",
		"command":{
			"name":"usage",
			"data":{
				"description":"Quota description test",
				"groups":[
					{
						"name":"Gemini Models",
						"description":"Gemini models group",
						"buckets":[
							{
								"id":"gemini-weekly",
								"name":"Weekly Limit Remaining",
								"description":"Weekly bucket",
								"window":"weekly",
								"remaining_fraction":0.95,
								"reset_time":"2026-10-24T03:51:51Z"
							}
						]
					}
				]
			}
		}
	}`

	var raw struct {
		Command *struct {
			Data *struct {
				Description string `json:"description"`
				Groups      []struct {
					Name    string `json:"name"`
					Buckets []struct {
						ID                string  `json:"id"`
						Name              string  `json:"name"`
						RemainingFraction float64 `json:"remaining_fraction"`
					} `json:"buckets"`
				} `json:"groups"`
			} `json:"data"`
		} `json:"command"`
	}

	if err := json.Unmarshal([]byte(sampleJSON), &raw); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}

	if raw.Command == nil || len(raw.Command.Data.Groups) != 1 {
		t.Fatalf("unexpected parsed groups count")
	}

	group := raw.Command.Data.Groups[0]
	if group.Name != "Gemini Models" {
		t.Errorf("expected 'Gemini Models', got '%s'", group.Name)
	}
	if group.Buckets[0].RemainingFraction != 0.95 {
		t.Errorf("expected 0.95, got %v", group.Buckets[0].RemainingFraction)
	}
}
