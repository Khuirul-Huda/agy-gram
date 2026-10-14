package domain

import (
	"context"
)

// PromptOptions holds options for sending a prompt to Antigravity CLI.
type PromptOptions struct {
	Prompt          string
	ContinueSession bool
	ConversationID  string
	Mode            string
	Model           string
	Effort          string
	VerboseThinking bool
	OnCommand       func(cmd string)
	OnFileEdit      func(toolName, targetFile, diff string)
	OnThinking      func(thinking string)
	OnConversation  func(conversationID string)
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
	GetConversationID() string
	SetConversationID(convID string)
	GetLastUserID() int64
	SetLastUserID(userID int64)
	GetMode() string
	SetMode(mode string)
	GetModel() string
	SetModel(model string)
	GetEffort() string
	SetEffort(effort string)
	GetLanguage() string
	SetLanguage(lang string)
	IsThinkingEnabled() bool
	SetThinkingEnabled(enabled bool)
	Reset()
}

// MetricsCollector collects host system resource metrics.
type MetricsCollector interface {
	Collect() (*SystemStatus, error)
}
