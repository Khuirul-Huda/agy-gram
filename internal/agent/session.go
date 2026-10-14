package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

type sessionData struct {
	ConversationID  string `json:"conversation_id"`
	ContinueSession bool   `json:"continue_session"`
	Mode            string `json:"mode"`
	Model           string `json:"model"`
	Effort          string `json:"effort"`
	Language        string `json:"language"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	LastUserID      int64  `json:"last_user_id"`
}

// Session manages conversation state and active agent settings.
type Session struct {
	mu              sync.RWMutex
	persistPath     string
	conversationID  string
	continueSession bool
	mode            string
	model           string
	effort          string
	language        string
	thinkingEnabled bool
	lastUserID      int64
}

// NewSession creates a new Session instance with default mode, model, thinking preference, and optional persistence file.
func NewSession(defaultMode, defaultModel string, defaultThinking bool, persistPath ...string) *Session {
	path := ""
	if len(persistPath) > 0 {
		path = persistPath[0]
	}

	s := &Session{
		persistPath:     path,
		conversationID:  "",
		continueSession: false,
		mode:            defaultMode,
		model:           defaultModel,
		effort:          "",
		language:        "id",
		thinkingEnabled: defaultThinking,
		lastUserID:      0,
	}

	if path != "" {
		s.load()
	}

	return s
}

func (s *Session) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		return
	}

	var state sessionData
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}

	if state.ConversationID != "" {
		s.conversationID = state.ConversationID
		s.continueSession = state.ContinueSession
	}
	if state.Mode != "" {
		s.mode = state.Mode
	}
	if state.Model != "" {
		s.model = state.Model
	}
	if state.Effort != "" {
		s.effort = state.Effort
	}
	if state.Language != "" {
		s.language = state.Language
	}
	s.thinkingEnabled = state.ThinkingEnabled
	if state.LastUserID != 0 {
		s.lastUserID = state.LastUserID
	}
}

func (s *Session) saveLocked() {
	if s.persistPath == "" {
		return
	}

	state := sessionData{
		ConversationID:  s.conversationID,
		ContinueSession: s.continueSession,
		Mode:            s.mode,
		Model:           s.model,
		Effort:          s.effort,
		Language:        s.language,
		ThinkingEnabled: s.thinkingEnabled,
		LastUserID:      s.lastUserID,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}

	dir := filepath.Dir(s.persistPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	_ = os.WriteFile(s.persistPath, data, 0644)
}

// GetConversationID returns the active Antigravity CLI conversation ID.
func (s *Session) GetConversationID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conversationID
}

// SetConversationID updates the tracked Antigravity CLI conversation ID.
func (s *Session) SetConversationID(convID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationID = convID
	s.saveLocked()
}

// GetLastUserID returns the Telegram user ID that last interacted with the bot.
func (s *Session) GetLastUserID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUserID
}

// SetLastUserID updates the Telegram user ID that last interacted with the bot.
func (s *Session) SetLastUserID(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUserID = userID
	s.saveLocked()
}

// ShouldContinue returns whether the next prompt should continue the last conversation.
func (s *Session) ShouldContinue() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.continueSession
}

// SetContinue sets the continuation flag.
func (s *Session) SetContinue(shouldContinue bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.continueSession = shouldContinue
	s.saveLocked()
}

// GetMode returns the current execution mode.
func (s *Session) GetMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// SetMode sets the execution mode.
func (s *Session) SetMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
	s.saveLocked()
}

// GetModel returns the current model override.
func (s *Session) GetModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.model
}

// SetModel sets the model override.
func (s *Session) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
	s.saveLocked()
}

// GetEffort returns the current reasoning effort override.
func (s *Session) GetEffort() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.effort
}

// SetEffort sets the reasoning effort override.
func (s *Session) SetEffort(effort string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.effort = effort
	s.saveLocked()
}

// GetLanguage returns the current UI language (id/en).
func (s *Session) GetLanguage() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.language == "" {
		return "id"
	}
	return s.language
}

// SetLanguage sets the UI language.
func (s *Session) SetLanguage(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.language = lang
	s.saveLocked()
}

// IsThinkingEnabled returns whether verbose thinking is enabled.
func (s *Session) IsThinkingEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.thinkingEnabled
}

// SetThinkingEnabled sets the verbose thinking preference.
func (s *Session) SetThinkingEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.thinkingEnabled = enabled
	s.saveLocked()
}

// Reset resets the conversation session state to start a fresh conversation.
func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationID = ""
	s.continueSession = false
	s.saveLocked()
}

var _ domain.SessionManager = (*Session)(nil)
