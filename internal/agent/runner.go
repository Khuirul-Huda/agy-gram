package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

// Runner implements domain.AgentRunner for Antigravity CLI.
type Runner struct {
	binPath    string
	workingDir string
	timeout    time.Duration

	mu         sync.Mutex
	currentCmd *exec.Cmd
	cancelFn   context.CancelFunc
}

// NewRunner creates a new Runner instance.
func NewRunner(binPath, workingDir string, timeout time.Duration) *Runner {
	return &Runner{
		binPath:    binPath,
		workingDir: workingDir,
		timeout:    timeout,
	}
}

// Stream JSON structures emitted by agy CLI with --output-format stream-json
type streamEvent struct {
	Event          string          `json:"event"`
	ConversationID string          `json:"conversation_id"`
	Init           *streamInit     `json:"init,omitempty"`
	StepUpdate     *streamStep     `json:"step_update,omitempty"`
	Result         *streamResult   `json:"result,omitempty"`
}

type streamInit struct {
	Cwd string `json:"cwd"`
}

type streamStep struct {
	ConversationID string          `json:"conversation_id"`
	StepIndex      int             `json:"step_index"`
	State          string          `json:"state"`
	StepType       string          `json:"step_type"`
	ToolName       string          `json:"tool_name"`
	ToolInfo       *streamToolInfo `json:"tool_info,omitempty"`
	TextDelta      string          `json:"text_delta,omitempty"`
}

type streamToolInfo struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Output     string                 `json:"output"`
}

type streamResult struct {
	ConversationID string `json:"conversation_id"`
	Status         string `json:"status"`
	Response       string `json:"response"`
}

// Execute runs the agy prompt command with real-time stream parsing.
func (r *Runner) Execute(ctx context.Context, opts domain.PromptOptions) (string, error) {
	if strings.TrimSpace(opts.Prompt) == "" {
		return "", domain.ErrEmptyPrompt
	}

	// Ensure only one execution runs at a time to prevent server overload
	r.mu.Lock()
	if r.currentCmd != nil {
		r.mu.Unlock()
		return "", domain.ErrAgentBusy
	}

	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	r.cancelFn = cancel

	args := r.buildArgs(opts)
	cmd := exec.CommandContext(execCtx, r.binPath, args...)
	cmd.Dir = r.workingDir

	// Put process into its own process group to ensure clean termination of children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		r.mu.Unlock()
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	r.currentCmd = cmd
	r.mu.Unlock()

	defer r.cleanupProcess()

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start agent process: %w", err)
	}

	var (
		conversationID string
		finalResponse  string
		textDeltas        strings.Builder
		rawStdout         strings.Builder
		seenToolSteps     = make(map[int]bool)
		seenThinkingSteps = make(map[int]bool)
	)

	// Stream read stdout line by line
	scanner := bufio.NewScanner(stdoutPipe)
	// Support up to 1MB per JSON line
	scanBuf := make([]byte, 64*1024)
	scanner.Buffer(scanBuf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		rawStdout.WriteString(line)
		rawStdout.WriteString("\n")

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
			continue
		}

		var ev streamEvent
		if err := json.Unmarshal([]byte(trimmed), &ev); err != nil {
			continue
		}

		if ev.ConversationID != "" && conversationID == "" {
			conversationID = ev.ConversationID
		}

		if ev.StepUpdate != nil {
			step := ev.StepUpdate
			if step.ConversationID != "" && conversationID == "" {
				conversationID = step.ConversationID
			}

			// 1. Real-time thinking polling: stream thinking as soon as it appears in transcript
			if opts.VerboseThinking && opts.OnThinking != nil && conversationID != "" {
				for _, th := range pollNewThinking(conversationID, seenThinkingSteps) {
					opts.OnThinking(th)
				}
			}

			// 2. Capture tool executions (run_command, replace_file_content, write_to_file)
			if step.StepType == "tool" && step.State == "ACTIVE" {
				if !seenToolSteps[step.StepIndex] {
					seenToolSteps[step.StepIndex] = true
					if step.ToolInfo != nil && step.ToolInfo.Parameters != nil {
						params := step.ToolInfo.Parameters
						switch step.ToolName {
						case "run_command":
							if opts.OnCommand != nil {
								if cmdVal, ok := params["CommandLine"]; ok {
									if cmdStr, okStr := cmdVal.(string); okStr && strings.TrimSpace(cmdStr) != "" {
										opts.OnCommand(strings.TrimSpace(cmdStr))
									}
								}
							}
						case "replace_file_content":
							if opts.OnFileEdit != nil {
								targetFile, _ := params["TargetFile"].(string)
								instruction, _ := params["Instruction"].(string)
								targetContent, _ := params["TargetContent"].(string)
								replacementContent, _ := params["ReplacementContent"].(string)

								var diffBuilder strings.Builder
								if instruction != "" {
									diffBuilder.WriteString(fmt.Sprintf("Instruction: %s\n\n", instruction))
								}
								if targetContent != "" || replacementContent != "" {
									diffBuilder.WriteString("--- Target:\n" + truncateSnippet(targetContent, 400) + "\n\n")
									diffBuilder.WriteString("+++ Replacement:\n" + truncateSnippet(replacementContent, 400))
								}
								opts.OnFileEdit("replace_file_content", targetFile, diffBuilder.String())
							}
						case "write_to_file":
							if opts.OnFileEdit != nil {
								targetFile, _ := params["TargetFile"].(string)
								desc, _ := params["Description"].(string)
								codeContent, _ := params["CodeContent"].(string)

								var diffBuilder strings.Builder
								if desc != "" {
									diffBuilder.WriteString(fmt.Sprintf("Description: %s\n\n", desc))
								}
								if codeContent != "" {
									diffBuilder.WriteString("Content:\n" + truncateSnippet(codeContent, 500))
								}
								opts.OnFileEdit("write_to_file", targetFile, diffBuilder.String())
							}
						}
					}
				}
			}

			// Accumulate response text deltas if any
			if step.StepType == "agent_response" && step.TextDelta != "" {
				textDeltas.WriteString(step.TextDelta)
			}
		}

		if ev.Result != nil {
			if ev.Result.ConversationID != "" {
				conversationID = ev.Result.ConversationID
			}
			if ev.Result.Response != "" {
				finalResponse = ev.Result.Response
			}
		}
	}

	// Final check for any remaining thinking blocks
	if opts.VerboseThinking && opts.OnThinking != nil && conversationID != "" {
		for _, th := range pollNewThinking(conversationID, seenThinkingSteps) {
			opts.OnThinking(th)
		}
	}

	waitErr := cmd.Wait()
	stderrOutput := stderrBuf.String()

	if waitErr != nil {
		if errors.Is(execCtx.Err(), context.Canceled) {
			return "", domain.ErrExecutionCancelled
		}
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("agent execution timed out after %v", r.timeout)
		}
		if finalResponse == "" && textDeltas.Len() == 0 && stderrOutput != "" {
			return "", fmt.Errorf("agent error: %s", strings.TrimSpace(stderrOutput))
		}
	}

	// Determine best output text
	output := finalResponse
	if strings.TrimSpace(output) == "" {
		output = textDeltas.String()
	}
	if strings.TrimSpace(output) == "" {
		output = rawStdout.String()
	}
	if strings.TrimSpace(output) == "" && stderrOutput != "" {
		output = strings.TrimSpace(stderrOutput)
	}
	if strings.TrimSpace(output) == "" {
		output = "(No output returned by agent)"
	}

	return strings.TrimSpace(output), nil
}

// CancelCurrent terminates the currently running agent process group.
func (r *Runner) CancelCurrent() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.currentCmd == nil || r.currentCmd.Process == nil {
		return false
	}

	// Kill entire process group by sending negative PID
	pgid := r.currentCmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGKILL)

	if r.cancelFn != nil {
		r.cancelFn()
	}

	return true
}

// IsRunning reports whether an agent execution is actively running.
func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentCmd != nil
}

func (r *Runner) buildArgs(opts domain.PromptOptions) []string {
	args := []string{"-p", opts.Prompt, "--dangerously-skip-permissions", "--output-format", "stream-json"}

	if opts.ContinueSession {
		args = append(args, "-c")
	}
	if opts.Mode != "" {
		args = append(args, "--mode", opts.Mode)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if opts.Effort != "" {
		args = append(args, "--effort", opts.Effort)
	}

	return args
}

func (r *Runner) cleanupProcess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentCmd = nil
	r.cancelFn = nil
}

// pollNewThinking parses new thinking blocks from transcript that haven't been emitted yet.
func pollNewThinking(convID string, seenSteps map[int]bool) []string {
	if convID == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	baseDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs")
	transcriptPath := filepath.Join(baseDir, "transcript.jsonl")

	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var newThoughts []string
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var step struct {
			StepIndex int    `json:"step_index"`
			Type      string `json:"type"`
			Thinking  string `json:"thinking"`
		}
		if err := json.Unmarshal(line, &step); err == nil {
			if step.Type == "PLANNER_RESPONSE" && !seenSteps[step.StepIndex] {
				seenSteps[step.StepIndex] = true
				if th := strings.TrimSpace(step.Thinking); th != "" {
					newThoughts = append(newThoughts, th)
				}
			}
		}
	}

	return newThoughts
}

func truncateSnippet(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n...(dipotong karena panjang)"
}
