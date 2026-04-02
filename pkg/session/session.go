// Package session defines session management types for Techne Code.
// Sessions represent conversations with the AI agent, including message
// history and metadata.
package session

import (
	"encoding/json"
	"time"
)

// Session represents a conversation session with the AI agent.
type Session struct {
	// ID is the unique identifier for this session.
	ID string `json:"id"`
	// Title is a human-readable title for the session (often auto-generated from first message).
	Title string `json:"title"`
	// Model is the LLM model used for this session.
	Model string `json:"model"`
	// Provider is the LLM provider used for this session (e.g., "anthropic", "openai").
	Provider string `json:"provider"`
	// SummaryMessageID references the message containing the conversation summary (for context compression).
	SummaryMessageID *string `json:"summary_message_id,omitempty"`
	// CreatedAt is the timestamp when the session was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the timestamp when the session was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// StoredMessage represents a message persisted in the session history.
type StoredMessage struct {
	// ID is the unique identifier for this message.
	ID string `json:"id"`
	// SessionID references the session this message belongs to.
	SessionID string `json:"session_id"`
	// Role indicates who sent the message ("user", "assistant", "system", "tool").
	Role string `json:"role"`
	// Content contains the message content as raw JSON (supports complex content blocks).
	Content json.RawMessage `json:"content"`
	// Model is the model that generated this message (for assistant messages).
	Model string `json:"model,omitempty"`
	// CreatedAt is the timestamp when the message was created.
	CreatedAt time.Time `json:"created_at"`
}

// SessionStore defines the interface for session persistence.
// Implementations handle storing and retrieving sessions and their messages.
type SessionStore interface {
	// CreateSession creates a new session in the store.
	CreateSession(s *Session) error
	// GetSession retrieves a session by ID.
	GetSession(id string) (*Session, error)
	// ListSessions returns all sessions, typically ordered by most recent first.
	ListSessions() ([]Session, error)
	// UpdateSessionTitle updates the title of a session.
	UpdateSessionTitle(id, title string) error
	// UpdateSessionSummary updates the summary message reference for context compression.
	UpdateSessionSummary(id, summaryMessageID string) error
	// DeleteSession removes a session and all its messages from the store.
	DeleteSession(id string) error

	// SaveMessage stores a message in the session history.
	SaveMessage(m *StoredMessage) error
	// GetMessages retrieves all messages for a session, ordered chronologically.
	GetMessages(sessionID string) ([]StoredMessage, error)
	// GetMessagesAfter retrieves messages created after the specified time.
	// Useful for fetching new messages after a summary point.
	GetMessagesAfter(sessionID string, after time.Time) ([]StoredMessage, error)
	// DeleteMessages removes all messages for a session.
	DeleteMessages(sessionID string) error

	// TrackReadFile records that a file was read in a session.
	// Used for context tracking and permission decisions.
	TrackReadFile(sessionID, path string) error
	// HasReadFile checks if a file was previously read in a session.
	HasReadFile(sessionID, path string) (bool, error)
}
