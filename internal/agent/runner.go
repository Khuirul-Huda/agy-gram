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

	var execCtx context.Context
	var cancel context.CancelFunc
	if r.timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, r.timeout)
	} else {
		execCtx, cancel = context.WithCancel(ctx)
	}
	r.cancelFn = cancel

	args := r.buildArgs(opts)
	cmd := exec.CommandContext(execCtx, r.binPath, args...)
	if opts.Workspace != "" {
		cmd.Dir = opts.Workspace
	} else {
		cmd.Dir = r.workingDir
	}

	// Ensure agy CLI process NEVER inherits any proxy; it must run 100% direct without proxy
	var cleanEnv []string
	for _, env := range os.Environ() {
		upper := strings.ToUpper(env)
		if strings.HasPrefix(upper, "HTTP_PROXY=") ||
			strings.HasPrefix(upper, "HTTPS_PROXY=") ||
			strings.HasPrefix(upper, "ALL_PROXY=") ||
			strings.HasPrefix(upper, "TELEGRAM_PROXY=") {
			continue
		}
		cleanEnv = append(cleanEnv, env)
	}
	cmd.Env = cleanEnv

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
			if opts.OnConversation != nil {
				opts.OnConversation(conversationID)
			}
		}

		if ev.StepUpdate != nil {
			step := ev.StepUpdate
			if step.ConversationID != "" && conversationID == "" {
				conversationID = step.ConversationID
				if opts.OnConversation != nil {
					opts.OnConversation(conversationID)
				}
			}

			// 1. Real-time thinking polling: stream thinking as soon as it appears in transcript
			if opts.VerboseThinking && opts.OnThinking != nil && conversationID != "" {
				for _, th := range pollNewThinking(conversationID, seenThinkingSteps) {
					opts.OnThinking(th)
				}
			}

			// 2. Capture tool executions (run_command, edit_file, write_file, read_file, legacy replace_file_content, write_to_file)
			if step.StepType == "tool" && step.State == "ACTIVE" {
				if !seenToolSteps[step.StepIndex] {
					seenToolSteps[step.StepIndex] = true
					handleToolStep(step, conversationID, opts)
				}
			}

			// Accumulate response text deltas if any
			if step.StepType == "agent_response" && step.TextDelta != "" {
				textDeltas.WriteString(step.TextDelta)
				if opts.OnTextDelta != nil {
					opts.OnTextDelta(step.TextDelta)
				}
			}
		}

		if ev.Result != nil {
			if ev.Result.ConversationID != "" {
				conversationID = ev.Result.ConversationID
				if opts.OnConversation != nil {
					opts.OnConversation(conversationID)
				}
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

	if opts.ConversationID != "" {
		args = append(args, "--conversation", opts.ConversationID)
	} else if opts.ContinueSession {
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

// GetUsage executes /usage command against agy CLI with json output and parses quota information.
func (r *Runner) GetUsage(ctx context.Context) (*domain.QuotaInfo, error) {
	// agy -p "/usage" --output-format json
	cmdCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, r.binPath, "-p", "/usage", "--output-format", "json")
	cmd.Dir = r.workingDir

	var cleanEnv []string
	for _, env := range os.Environ() {
		upper := strings.ToUpper(env)
		if strings.HasPrefix(upper, "HTTP_PROXY=") ||
			strings.HasPrefix(upper, "HTTPS_PROXY=") ||
			strings.HasPrefix(upper, "ALL_PROXY=") ||
			strings.HasPrefix(upper, "TELEGRAM_PROXY=") {
			continue
		}
		cleanEnv = append(cleanEnv, env)
	}
	cmd.Env = cleanEnv

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run agy /usage: %w", err)
	}

	var raw struct {
		Status  string `json:"status"`
		Command *struct {
			Name string `json:"name"`
			Data *struct {
				Description string `json:"description"`
				Groups      []struct {
					Name        string `json:"name"`
					Description string `json:"description"`
					Buckets     []struct {
						ID                string  `json:"id"`
						Name              string  `json:"name"`
						Description       string  `json:"description"`
						Window            string  `json:"window"`
						RemainingFraction float64 `json:"remaining_fraction"`
						ResetTime         string  `json:"reset_time"`
					} `json:"buckets"`
				} `json:"groups"`
			} `json:"data"`
		} `json:"command"`
	}

	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse /usage json output: %w", err)
	}

	if raw.Command == nil || raw.Command.Data == nil {
		return nil, errors.New("usage command data not found in agy output")
	}

	info := &domain.QuotaInfo{
		Description: raw.Command.Data.Description,
	}

	for _, g := range raw.Command.Data.Groups {
		group := domain.UsageGroup{
			Name:        g.Name,
			Description: g.Description,
		}
		for _, b := range g.Buckets {
			var resetT time.Time
			if b.ResetTime != "" {
				resetT, _ = time.Parse(time.RFC3339, b.ResetTime)
			}
			group.Buckets = append(group.Buckets, domain.UsageBucket{
				ID:                b.ID,
				Name:              b.Name,
				Description:       b.Description,
				Window:            b.Window,
				RemainingFraction: b.RemainingFraction,
				ResetTime:         resetT,
			})
		}
		info.Groups = append(info.Groups, group)
	}

	return info, nil
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

// fetchToolCallDetails retrieves tool parameters from the conversation logs.
// It prioritizes transcript_full.jsonl which contains untruncated content, with fallback to transcript.jsonl.
func fetchToolCallDetails(convID string, stepIndex int, toolName string) map[string]interface{} {
	if convID == "" {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	baseDir := filepath.Join(home, ".gemini", "antigravity-cli", "brain", convID, ".system_generated", "logs")
	logPaths := []string{
		filepath.Join(baseDir, "transcript_full.jsonl"),
		filepath.Join(baseDir, "transcript.jsonl"),
	}

	for _, p := range logPaths {
		if args := readToolCallFromLog(p, stepIndex, toolName); len(args) > 0 {
			return args
		}
	}

	// Brief pause if log write is still buffering
	time.Sleep(60 * time.Millisecond)
	for _, p := range logPaths {
		if args := readToolCallFromLog(p, stepIndex, toolName); len(args) > 0 {
			return args
		}
	}

	return nil
}

func readToolCallFromLog(filePath string, targetStep int, toolName string) map[string]interface{} {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	var latestMatch map[string]interface{}

	for scanner.Scan() {
		line := scanner.Bytes()
		var entry struct {
			StepIndex int `json:"step_index"`
			Type      string `json:"type"`
			ToolCalls []struct {
				Name string                 `json:"name"`
				Args map[string]interface{} `json:"args"`
			} `json:"tool_calls"`
		}
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		isNearStep := entry.StepIndex == targetStep || entry.StepIndex == targetStep-1 || entry.StepIndex == targetStep+1
		for _, tc := range entry.ToolCalls {
			if tc.Name == toolName {
				if isNearStep {
					return tc.Args
				}
				latestMatch = tc.Args
			}
		}
	}

	return latestMatch
}

func getToolArgString(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	val, ok := args[key]
	if !ok || val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return unescapeTranscriptArg(s)
	}
	return fmt.Sprintf("%v", val)
}

func formatDiff(targetContent, replacementContent string) string {
	var sb strings.Builder
	if targetContent != "" {
		sb.WriteString("--- Sebelum\n")
		lines := strings.Split(targetContent, "\n")
		if len(lines) > 20 {
			for _, l := range lines[:10] {
				sb.WriteString("- " + l + "\n")
			}
			sb.WriteString("- ... (dipotong)\n")
			for _, l := range lines[len(lines)-10:] {
				sb.WriteString("- " + l + "\n")
			}
		} else {
			for _, l := range lines {
				sb.WriteString("- " + l + "\n")
			}
		}
	}
	if replacementContent != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("+++ Sesudah\n")
		lines := strings.Split(replacementContent, "\n")
		if len(lines) > 20 {
			for _, l := range lines[:10] {
				sb.WriteString("+ " + l + "\n")
			}
			sb.WriteString("+ ... (dipotong)\n")
			for _, l := range lines[len(lines)-10:] {
				sb.WriteString("+ " + l + "\n")
			}
		} else {
			for _, l := range lines {
				sb.WriteString("+ " + l + "\n")
			}
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatWriteContent(codeContent string) string {
	var sb strings.Builder
	sb.WriteString("+++ File Baru\n")
	lines := strings.Split(codeContent, "\n")
	if len(lines) > 25 {
		for _, l := range lines[:15] {
			sb.WriteString("+ " + l + "\n")
		}
		sb.WriteString("+ ... (dipotong)\n")
		for _, l := range lines[len(lines)-10:] {
			sb.WriteString("+ " + l + "\n")
		}
	} else {
		for _, l := range lines {
			sb.WriteString("+ " + l + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// unescapeTranscriptArg unescapes JSON string quoting used in transcript logs.
func unescapeTranscriptArg(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		var decoded string
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			return decoded
		}
	}
	return raw
}

