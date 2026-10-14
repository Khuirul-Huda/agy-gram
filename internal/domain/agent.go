package domain

import (
	"context"
)

// PromptOptions holds options for sending a prompt to Antigravity CLI.
type PromptOptions struct {
	Prompt          string
	ContinueSession bool
	Mode            string
	Model           string
}

// SystemStatus represents host hardware and resource utilization metrics.
type SystemStatus struct {
	Uptime      string
	CPUModel    string
	MemoryUsage string
	SwapUsage   string
	DiskUsage   string
	AgentStatus string
}

// AgentRunner executes Antigravity CLI commands.
type AgentRunner interface {
	Execute(ctx context.Context, opts PromptOptions) (string, error)
	CancelCurrent() bool
	IsRunning() bool
}

// SessionManager manages conversation state and active agent configuration.
type SessionManager interface {
	ShouldContinue() bool
	SetContinue(shouldContinue bool)
	GetMode() string
	SetMode(mode string)
	GetModel() string
	SetModel(model string)
	Reset()
}

// MetricsCollector collects host system resource metrics.
type MetricsCollector interface {
	Collect() (*SystemStatus, error)
}
