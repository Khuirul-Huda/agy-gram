package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
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

func TestHandleToolStep_RunCommand(t *testing.T) {
	step := &streamStep{
		StepIndex: 1,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "run_command",
		ToolInfo: &streamToolInfo{
			Name: "run_command",
			Parameters: map[string]interface{}{
				"CommandLine": "echo hello",
				"toolSummary": "Say hello",
			},
		},
	}

	var calledCmd string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnCommand: func(cmd string) {
			calledCmd = cmd
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledCmd != "echo hello" {
		t.Errorf("expected command 'echo hello', got %q", calledCmd)
	}
	if receivedEvent.Kind != domain.ToolKindCommand {
		t.Errorf("expected ToolKindCommand, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "run_command" {
		t.Errorf("expected ToolName 'run_command', got %q", receivedEvent.ToolName)
	}
	if receivedEvent.Target != "echo hello" {
		t.Errorf("expected Target 'echo hello', got %q", receivedEvent.Target)
	}
	if receivedEvent.Detail != "Say hello" {
		t.Errorf("expected Detail 'Say hello', got %q", receivedEvent.Detail)
	}
}

func TestHandleToolStep_EditFile(t *testing.T) {
	step := &streamStep{
		StepIndex: 2,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "edit_file",
		ToolInfo: &streamToolInfo{
			Name: "edit_file",
			Parameters: map[string]interface{}{
				"Path":        "main.go",
				"Target":      "fmt.Println(\"before\")",
				"Replacement": "fmt.Println(\"after\")",
			},
		},
	}

	var calledTool string
	var calledFile string
	var calledDiff string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileEdit: func(toolName, targetFile, diff string) {
			calledTool = toolName
			calledFile = targetFile
			calledDiff = diff
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledTool != "edit_file" {
		t.Errorf("expected OnFileEdit tool 'edit_file', got %q", calledTool)
	}
	if calledFile != "main.go" {
		t.Errorf("expected file 'main.go', got %q", calledFile)
	}
	if !strings.Contains(calledDiff, "fmt.Println(\"before\")") {
		t.Errorf("expected diff to contain target content, got %q", calledDiff)
	}
	if receivedEvent.Kind != domain.ToolKindEdit {
		t.Errorf("expected ToolKindEdit, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "edit_file" {
		t.Errorf("expected ToolName 'edit_file', got %q", receivedEvent.ToolName)
	}
	if receivedEvent.Target != "main.go" {
		t.Errorf("expected Target 'main.go', got %q", receivedEvent.Target)
	}
	if !strings.Contains(receivedEvent.Detail, "fmt.Println(\"after\")") {
		t.Errorf("expected Detail to contain replacement content, got %q", receivedEvent.Detail)
	}
}

func TestHandleToolStep_LegacyReplaceFileContent(t *testing.T) {
	step := &streamStep{
		StepIndex: 3,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "replace_file_content",
		ToolInfo: &streamToolInfo{
			Name: "replace_file_content",
			Parameters: map[string]interface{}{
				"TargetFile":         "config.json",
				"Instruction":        "update port",
				"TargetContent":      `"port": 8080`,
				"ReplacementContent": `"port": 9090`,
			},
		},
	}

	var calledTool string
	var calledFile string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileEdit: func(toolName, targetFile, diff string) {
			calledTool = toolName
			calledFile = targetFile
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledTool != "edit_file" {
		t.Errorf("expected OnFileEdit tool 'edit_file', got %q", calledTool)
	}
	if calledFile != "config.json" {
		t.Errorf("expected targetFile 'config.json', got %q", calledFile)
	}
	if receivedEvent.Kind != domain.ToolKindEdit {
		t.Errorf("expected ToolKindEdit, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "replace_file_content" {
		t.Errorf("expected ToolName 'replace_file_content', got %q", receivedEvent.ToolName)
	}
	if !strings.Contains(receivedEvent.Detail, "Instruction: update port") {
		t.Errorf("expected instruction in detail, got %q", receivedEvent.Detail)
	}
}

func TestHandleToolStep_WriteFile(t *testing.T) {
	step := &streamStep{
		StepIndex: 4,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "write_file",
		ToolInfo: &streamToolInfo{
			Name: "write_file",
			Parameters: map[string]interface{}{
				"Path":    "output.txt",
				"Content": "hello world",
			},
		},
	}

	var calledTool string
	var calledFile string
	var calledDiff string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileEdit: func(toolName, targetFile, diff string) {
			calledTool = toolName
			calledFile = targetFile
			calledDiff = diff
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledTool != "write_file" {
		t.Errorf("expected OnFileEdit tool 'write_file', got %q", calledTool)
	}
	if calledFile != "output.txt" {
		t.Errorf("expected file 'output.txt', got %q", calledFile)
	}
	if !strings.Contains(calledDiff, "+ hello world") {
		t.Errorf("expected diff to contain '+ hello world', got %q", calledDiff)
	}
	if receivedEvent.Kind != domain.ToolKindWrite {
		t.Errorf("expected ToolKindWrite, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "write_file" {
		t.Errorf("expected ToolName 'write_file', got %q", receivedEvent.ToolName)
	}
	if receivedEvent.Target != "output.txt" {
		t.Errorf("expected Target 'output.txt', got %q", receivedEvent.Target)
	}
}

func TestHandleToolStep_LegacyWriteToFile(t *testing.T) {
	step := &streamStep{
		StepIndex: 5,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "write_to_file",
		ToolInfo: &streamToolInfo{
			Name: "write_to_file",
			Parameters: map[string]interface{}{
				"TargetFile":  "notes.md",
				"Description": "add notes",
				"CodeContent": "# Notes",
			},
		},
	}

	var calledTool string
	var calledFile string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileEdit: func(toolName, targetFile, diff string) {
			calledTool = toolName
			calledFile = targetFile
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledTool != "write_file" {
		t.Errorf("expected OnFileEdit tool 'write_file', got %q", calledTool)
	}
	if calledFile != "notes.md" {
		t.Errorf("expected targetFile 'notes.md', got %q", calledFile)
	}
	if receivedEvent.Kind != domain.ToolKindWrite {
		t.Errorf("expected ToolKindWrite, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "write_to_file" {
		t.Errorf("expected ToolName 'write_to_file', got %q", receivedEvent.ToolName)
	}
	if !strings.Contains(receivedEvent.Detail, "Description: add notes") {
		t.Errorf("expected description in detail, got %q", receivedEvent.Detail)
	}
}

func TestHandleToolStep_ReadFile(t *testing.T) {
	step := &streamStep{
		StepIndex: 6,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "read_file",
		ToolInfo: &streamToolInfo{
			Name: "read_file",
			Parameters: map[string]interface{}{
				"Path":      "/home/huda/test.txt",
				"StartLine": 10,
				"EndLine":   25,
			},
		},
	}

	var calledReadFile string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileRead: func(targetFile string) {
			calledReadFile = targetFile
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledReadFile != "/home/huda/test.txt" {
		t.Errorf("expected OnFileRead '/home/huda/test.txt', got %q", calledReadFile)
	}
	if receivedEvent.Kind != domain.ToolKindRead {
		t.Errorf("expected ToolKindRead, got %q", receivedEvent.Kind)
	}
	if receivedEvent.ToolName != "read_file" {
		t.Errorf("expected ToolName 'read_file', got %q", receivedEvent.ToolName)
	}
	if receivedEvent.Target != "/home/huda/test.txt" {
		t.Errorf("expected Target '/home/huda/test.txt', got %q", receivedEvent.Target)
	}
	if !strings.Contains(receivedEvent.Detail, "StartLine: 10") || !strings.Contains(receivedEvent.Detail, "EndLine: 25") {
		t.Errorf("expected detail to contain line range, got %q", receivedEvent.Detail)
	}
}

func TestHandleToolStep_ReadFile_LegacyTargetFile(t *testing.T) {
	step := &streamStep{
		StepIndex: 7,
		StepType:  "tool",
		State:     "ACTIVE",
		ToolName:  "read_file",
		ToolInfo: &streamToolInfo{
			Name: "read_file",
			Parameters: map[string]interface{}{
				"TargetFile": "/var/log/syslog",
			},
		},
	}

	var calledReadFile string
	var receivedEvent domain.ToolEvent

	opts := domain.PromptOptions{
		OnFileRead: func(targetFile string) {
			calledReadFile = targetFile
		},
		OnToolEvent: func(ev domain.ToolEvent) {
			receivedEvent = ev
		},
	}

	handleToolStep(step, "", opts)

	if calledReadFile != "/var/log/syslog" {
		t.Errorf("expected OnFileRead '/var/log/syslog', got %q", calledReadFile)
	}
	if receivedEvent.Kind != domain.ToolKindRead {
		t.Errorf("expected ToolKindRead, got %q", receivedEvent.Kind)
	}
	if receivedEvent.Target != "/var/log/syslog" {
		t.Errorf("expected Target '/var/log/syslog', got %q", receivedEvent.Target)
	}
}
