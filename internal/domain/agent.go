package domain

import (
	"context"
	"time"
)

// ToolKind classifies the type of tool operation performed by an agent.
type ToolKind string

const (
	ToolKindCommand ToolKind = "command"
	ToolKindRead    ToolKind = "read"
	ToolKindEdit    ToolKind = "edit"
	ToolKindWrite   ToolKind = "write"
)

// ToolEvent represents a structured agent tool execution event.
type ToolEvent struct {
	Kind     ToolKind
	ToolName string
	Target   string // file path or command string
	Detail   string // diff, description, or parameter details
}

// PromptOptions holds options for sending a prompt to Antigravity CLI.
type PromptOptions struct {
	Prompt          string
	ContinueSession bool
	ConversationID  string
	Workspace       string
	Mode            string
	Model           string
	Effort          string
	VerboseThinking bool
	OnCommand       func(cmd string)
	OnFileEdit      func(toolName, targetFile, diff string)
	OnFileRead      func(targetFile string)
	OnToolEvent     func(event ToolEvent)
	OnThinking      func(thinking string)
	OnTextDelta     func(delta string)
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

// SessionRecord represents a saved session in history.
type SessionRecord struct {
	ConversationID  string
	Title           string
	Workspace       string
	Mode            string
	Model           string
	Effort          string
	UpdatedAt       time.Time
}

// UsageBucket represents an individual quota limit window (e.g. 5-hour or weekly).
type UsageBucket struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Window            string    `json:"window"`
	RemainingFraction float64   `json:"remaining_fraction"`
	ResetTime         time.Time `json:"reset_time"`
}

// UsageGroup represents a group of models sharing a quota bucket (e.g. Gemini, Claude & GPT).
type UsageGroup struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Buckets     []UsageBucket `json:"buckets"`
}

// QuotaInfo represents the parsed quota and rate limit status from agy.
type QuotaInfo struct {
	Description string       `json:"description"`
	Groups      []UsageGroup `json:"groups"`
}

// AgentRunner executes Antigravity CLI commands.
type AgentRunner interface {
	Execute(ctx context.Context, opts PromptOptions) (string, error)
	GetUsage(ctx context.Context) (*QuotaInfo, error)
	CancelCurrent() bool
	IsRunning() bool
}

// SessionManager manages conversation state, workspaces, and active agent configuration.
type SessionManager interface {
	ShouldContinue() bool
	SetContinue(shouldContinue bool)
	GetConversationID() string
	SetConversationID(convID string)
	GetWorkspace() string
	SetWorkspace(dir string)
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
	GetLastDeliveredStepIndex() int
	SetLastDeliveredStepIndex(stepIndex int)
	Reset()
	ListSessions(limit int) ([]SessionRecord, error)
	SwitchSession(convID string) error
	ForTopic(chatID int64, threadID int) SessionManager
}

// MetricsCollector collects host system resource metrics.
type MetricsCollector interface {
	Collect() (*SystemStatus, error)
}
