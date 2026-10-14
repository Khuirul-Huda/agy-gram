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
		textDeltas     strings.Builder
		rawStdout      strings.Builder
		minStepIndex   = -1
		seenToolSteps  = make(map[int]bool)
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

			if minStepIndex == -1 {
				minStepIndex = step.StepIndex
			}

			// Capture tool execution (specifically run_command)
			if step.StepType == "tool" && step.ToolName == "run_command" && step.State == "ACTIVE" {
				if !seenToolSteps[step.StepIndex] {
					seenToolSteps[step.StepIndex] = true
					if opts.OnCommand != nil && step.ToolInfo != nil && step.ToolInfo.Parameters != nil {
						if cmdVal, ok := step.ToolInfo.Parameters["CommandLine"]; ok {
							if cmdStr, okStr := cmdVal.(string); okStr && strings.TrimSpace(cmdStr) != "" {
								opts.OnCommand(strings.TrimSpace(cmdStr))
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

	// If verbose thinking is requested, extract thinking trace from transcript
	if opts.VerboseThinking && opts.OnThinking != nil && conversationID != "" {
		if thinking := extractThinkingFromTranscript(conversationID, minStepIndex); thinking != "" {
			opts.OnThinking(thinking)
		}
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

// extractThinkingFromTranscript parses transcript files for thinking blocks.
func extractThinkingFromTranscript(convID string, minStepIndex int) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check transcript.jsonl first, then fallback to transcript_full.jsonl if needed
	baseDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs")
	transcriptPath := filepath.Join(baseDir, "transcript.jsonl")

	f, err := os.Open(transcriptPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	var thoughts []string
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
			if (minStepIndex < 0 || step.StepIndex >= minStepIndex) && step.Type == "PLANNER_RESPONSE" {
				if th := strings.TrimSpace(step.Thinking); th != "" {
					thoughts = append(thoughts, th)
				}
			}
		}
	}

	return strings.Join(thoughts, "\n\n")
}
