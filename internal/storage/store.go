package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	bbolt "go.etcd.io/bbolt"
)

var (
	bucketState       = []byte("state")
	bucketSessions    = []byte("sessions")
	keySession        = []byte("session")
	keyActiveWorkspace = []byte("active_workspace")
)

// SessionState stores persistent session information across bot restarts.
type SessionState struct {
	ConversationID  string    `json:"conversation_id"`
	Title           string    `json:"title"`
	Workspace       string    `json:"workspace"`
	ContinueSession bool      `json:"continue_session"`
	Mode            string    `json:"mode"`
	Model           string    `json:"model"`
	Effort          string    `json:"effort"`
	Language        string    `json:"language"`
	ThinkingEnabled        bool      `json:"thinking_enabled"`
	LastUserID             int64     `json:"last_user_id"`
	LastDeliveredStepIndex int       `json:"last_delivered_step_index"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// BoltStore provides lightning-fast embedded key-value storage using bbolt.
type BoltStore struct {
	db *bbolt.DB
}

// Open initializes or opens a bbolt database at the given file path.
func Open(dbPath string) (*BoltStore, error) {
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for bolt db: %w", err)
		}
	}

	opts := &bbolt.Options{
		Timeout: 2 * time.Second,
	}

	db, err := bbolt.Open(dbPath, 0600, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open bolt db at %s: %w", dbPath, err)
	}

	// Ensure standard buckets exist
	err = db.Update(func(tx *bbolt.Tx) error {
		if _, bErr := tx.CreateBucketIfNotExists(bucketState); bErr != nil {
			return bErr
		}
		if _, bErr := tx.CreateBucketIfNotExists(bucketSessions); bErr != nil {
			return bErr
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize buckets in bolt db: %w", err)
	}

	return &BoltStore{db: db}, nil
}

// Close closes the database cleanly.
func (s *BoltStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveSession persists the active session state and archives it into sessions history.
func (s *BoltStore) SaveSession(state SessionState) error {
	state.UpdatedAt = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		bState := tx.Bucket(bucketState)
		if bState == nil {
			return fmt.Errorf("bucket %s not found", bucketState)
		}
		if err := bState.Put(keySession, data); err != nil {
			return err
		}

		if state.ConversationID != "" {
			bSessions := tx.Bucket(bucketSessions)
			if bSessions != nil {
				_ = bSessions.Put([]byte(state.ConversationID), data)
			}
		}
		return nil
	})
}

// LoadSession retrieves the persistent session state. Returns nil if none exists.
func (s *BoltStore) LoadSession() (*SessionState, error) {
	var state SessionState
	var found bool

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketState)
		if b == nil {
			return nil
		}
		data := b.Get(keySession)
		if len(data) == 0 {
			return nil
		}
		found = true
		return json.Unmarshal(data, &state)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load session state: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &state, nil
}

// ListSessions returns recent conversation sessions sorted by UpdatedAt descending.
func (s *BoltStore) ListSessions(limit int) ([]SessionState, error) {
	var sessions []SessionState

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var st SessionState
			if err := json.Unmarshal(v, &st); err == nil {
				sessions = append(sessions, st)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

// GetSession retrieves a specific session by conversation ID.
func (s *BoltStore) GetSession(convID string) (*SessionState, error) {
	var state SessionState
	var found bool

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		if b == nil {
			return nil
		}
		data := b.Get([]byte(convID))
		if len(data) == 0 {
			return nil
		}
		found = true
		return json.Unmarshal(data, &state)
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &state, nil
}

// DeleteSession removes a session from history.
func (s *BoltStore) DeleteSession(convID string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketSessions)
		if b != nil {
			return b.Delete([]byte(convID))
		}
		return nil
	})
}
