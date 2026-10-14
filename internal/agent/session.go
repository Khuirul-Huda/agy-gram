package agent

import (
	"sync"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
)

// Session manages conversation state and active agent settings.
type Session struct {
	mu              sync.RWMutex
	continueSession bool
	mode            string
	model           string
	effort          string
	thinkingEnabled bool
}

// NewSession creates a new Session instance with default mode, model, and thinking preference.
func NewSession(defaultMode, defaultModel string, defaultThinking bool) *Session {
	return &Session{
		continueSession: false,
		mode:            defaultMode,
		model:           defaultModel,
		effort:          "",
		thinkingEnabled: defaultThinking,
	}
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
}

// Reset resets the session state to start a fresh conversation.
func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.continueSession = false
}

var _ domain.SessionManager = (*Session)(nil)
