package agent

import (
	"log/slog"
	"sync"

	"github.com/Khuirul-Huda/agy-gram/internal/domain"
	"github.com/Khuirul-Huda/agy-gram/internal/storage"
)

// Session manages conversation state and active agent settings with bbolt persistence.
type Session struct {
	mu              sync.RWMutex
	store           *storage.BoltStore
	conversationID  string
	workspace       string
	continueSession bool
	mode            string
	model           string
	effort          string
	language        string
	thinkingEnabled        bool
	lastUserID             int64
	lastDeliveredStepIndex int
}

// NewSession creates a new Session instance with default mode, model, thinking preference, and optional bbolt store.
func NewSession(defaultMode, defaultModel string, defaultThinking bool, store ...*storage.BoltStore) *Session {
	var sStore *storage.BoltStore
	if len(store) > 0 {
		sStore = store[0]
	}

	s := &Session{
		store:           sStore,
		conversationID:  "",
		workspace:       "",
		continueSession: false,
		mode:            defaultMode,
		model:           defaultModel,
		effort:          "",
		language:        "id",
		thinkingEnabled: defaultThinking,
		lastUserID:      0,
	}

	if s.store != nil {
		s.load()
	}

	return s
}

func (s *Session) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.store.LoadSession()
	if err != nil {
		slog.Warn("Failed to load session from bbolt store", "error", err)
		return
	}
	if state == nil {
		return
	}

	if state.ConversationID != "" {
		s.conversationID = state.ConversationID
		s.continueSession = state.ContinueSession
	}
	if state.Workspace != "" {
		s.workspace = state.Workspace
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
	s.lastDeliveredStepIndex = state.LastDeliveredStepIndex
}

func (s *Session) saveLocked() {
	if s.store == nil {
		return
	}

	state := storage.SessionState{
		ConversationID:         s.conversationID,
		Workspace:              s.workspace,
		ContinueSession:        s.continueSession,
		Mode:                   s.mode,
		Model:                  s.model,
		Effort:                 s.effort,
		Language:               s.language,
		ThinkingEnabled:        s.thinkingEnabled,
		LastUserID:             s.lastUserID,
		LastDeliveredStepIndex: s.lastDeliveredStepIndex,
	}

	if err := s.store.SaveSession(state); err != nil {
		slog.Warn("Failed to save session state to bbolt store", "error", err)
	}
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

// GetWorkspace returns the active workspace path for AGY CLI.
func (s *Session) GetWorkspace() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

// SetWorkspace updates the active working directory for AGY CLI.
func (s *Session) SetWorkspace(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspace = dir
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

// GetLastDeliveredStepIndex returns the latest step index delivered to Telegram.
func (s *Session) GetLastDeliveredStepIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastDeliveredStepIndex
}

// SetLastDeliveredStepIndex updates the latest step index delivered to Telegram.
func (s *Session) SetLastDeliveredStepIndex(stepIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastDeliveredStepIndex = stepIndex
	s.saveLocked()
}

// Reset resets the conversation session state to start a fresh conversation.
func (s *Session) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversationID = ""
	s.continueSession = false
	s.lastDeliveredStepIndex = 0
	s.saveLocked()
}

// ListSessions returns recent conversation sessions from history.
func (s *Session) ListSessions(limit int) ([]domain.SessionRecord, error) {
	if s.store == nil {
		return nil, nil
	}

	sessions, err := s.store.ListSessions(limit)
	if err != nil {
		return nil, err
	}

	var records []domain.SessionRecord
	for _, st := range sessions {
		records = append(records, domain.SessionRecord{
			ConversationID: st.ConversationID,
			Title:          st.Title,
			Workspace:      st.Workspace,
			Mode:           st.Mode,
			Model:          st.Model,
			Effort:         st.Effort,
			UpdatedAt:      st.UpdatedAt,
		})
	}

	return records, nil
}

// SwitchSession activates a historical session.
func (s *Session) SwitchSession(convID string) error {
	if s.store == nil {
		return nil
	}

	st, err := s.store.GetSession(convID)
	if err != nil || st == nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversationID = st.ConversationID
	s.continueSession = true
	if st.Workspace != "" {
		s.workspace = st.Workspace
	}
	if st.Mode != "" {
		s.mode = st.Mode
	}
	if st.Model != "" {
		s.model = st.Model
	}
	if st.Effort != "" {
		s.effort = st.Effort
	}
	s.saveLocked()

	return nil
}

// GetStore returns the underlying BoltStore instance.
func (s *Session) GetStore() *storage.BoltStore {
	return s.store
}

var _ domain.SessionManager = (*Session)(nil)
