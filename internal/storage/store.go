package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bbolt "go.etcd.io/bbolt"
)

var (
	bucketState = []byte("state")
	keySession  = []byte("session")
)

// SessionState stores persistent session information across bot restarts.
type SessionState struct {
	ConversationID  string    `json:"conversation_id"`
	ContinueSession bool      `json:"continue_session"`
	Mode            string    `json:"mode"`
	Model           string    `json:"model"`
	Effort          string    `json:"effort"`
	Language        string    `json:"language"`
	ThinkingEnabled bool      `json:"thinking_enabled"`
	LastUserID      int64     `json:"last_user_id"`
	UpdatedAt       time.Time `json:"updated_at"`
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
		_, bErr := tx.CreateBucketIfNotExists(bucketState)
		return bErr
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to initialize bucket in bolt db: %w", err)
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

// SaveSession persists the session state to bbolt.
func (s *BoltStore) SaveSession(state SessionState) error {
	state.UpdatedAt = time.Now()
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal session state: %w", err)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketState)
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketState)
		}
		return b.Put(keySession, data)
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
