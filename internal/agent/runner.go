package agent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
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

// Execute runs the agy prompt command with context and timeout.
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

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	r.currentCmd = cmd
	r.mu.Unlock()

	defer r.cleanupProcess()

	err := cmd.Run()
	output := stdoutBuf.String()
	stderrOutput := stderrBuf.String()

	if err != nil {
		if errors.Is(execCtx.Err(), context.Canceled) {
			return "", domain.ErrExecutionCancelled
		}
		if errors.Is(execCtx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("agent execution timed out after %v", r.timeout)
		}
		if output == "" && stderrOutput != "" {
			return "", fmt.Errorf("agent error: %s", strings.TrimSpace(stderrOutput))
		}
	}

	combined := strings.TrimSpace(output)
	if combined == "" && stderrOutput != "" {
		combined = strings.TrimSpace(stderrOutput)
	}
	if combined == "" {
		combined = "(No output returned by agent)"
	}

	return combined, nil
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
	args := []string{"-p", opts.Prompt, "--dangerously-skip-permissions"}

	if opts.ContinueSession {
		args = append(args, "-c")
	}
	if opts.Mode != "" {
		args = append(args, "--mode", opts.Mode)
	}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}

	return args
}

func (r *Runner) cleanupProcess() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentCmd = nil
	r.cancelFn = nil
}
