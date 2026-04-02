package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Edcko/techne-code/pkg/session"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormSessionStore implements session.SessionStore using GORM with SQLite.
type GormSessionStore struct {
	db *gorm.DB
}

// NewSessionStore creates a new GormSessionStore with the given database connection.
func NewSessionStore(db *gorm.DB) *GormSessionStore {
	return &GormSessionStore{db: db}
}

// CreateSession creates a new session in the store.
// If sess.ID is empty, a UUID v4 is generated.
// CreatedAt and UpdatedAt are set to the current time.
func (s *GormSessionStore) CreateSession(sess *session.Session) error {
	now := time.Now()

	// Generate UUID if not provided
	id := sess.ID
	if id == "" {
		id = uuid.New().String()
	}

	model := &SessionModel{
		ID:               id,
		Title:            sess.Title,
		Model:            sess.Model,
		Provider:         sess.Provider,
		SummaryMessageID: sess.SummaryMessageID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Update the input session with generated values
	sess.ID = id
	sess.CreatedAt = now
	sess.UpdatedAt = now

	return nil
}

// GetSession retrieves a session by ID.
// Returns nil if the session doesn't exist.
func (s *GormSessionStore) GetSession(id string) (*session.Session, error) {
	var model SessionModel
	if err := s.db.First(&model, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session.Session{
		ID:               model.ID,
		Title:            model.Title,
		Model:            model.Model,
		Provider:         model.Provider,
		SummaryMessageID: model.SummaryMessageID,
		CreatedAt:        model.CreatedAt,
		UpdatedAt:        model.UpdatedAt,
	}, nil
}

// ListSessions returns all sessions, ordered by UpdatedAt descending (most recent first).
func (s *GormSessionStore) ListSessions() ([]session.Session, error) {
	var models []SessionModel
	if err := s.db.Order("updated_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	sessions := make([]session.Session, len(models))
	for i, model := range models {
		sessions[i] = session.Session{
			ID:               model.ID,
			Title:            model.Title,
			Model:            model.Model,
			Provider:         model.Provider,
			SummaryMessageID: model.SummaryMessageID,
			CreatedAt:        model.CreatedAt,
			UpdatedAt:        model.UpdatedAt,
		}
	}

	return sessions, nil
}

// UpdateSessionTitle updates the title of a session.
func (s *GormSessionStore) UpdateSessionTitle(id, title string) error {
	result := s.db.Model(&SessionModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"title":      title,
			"updated_at": time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update session title: %w", result.Error)
	}

	return nil
}

// UpdateSessionSummary updates the summary message reference for context compression.
func (s *GormSessionStore) UpdateSessionSummary(id, summaryMessageID string) error {
	result := s.db.Model(&SessionModel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"summary_message_id": summaryMessageID,
			"updated_at":         time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update session summary: %w", result.Error)
	}

	return nil
}

// DeleteSession removes a session and all its messages from the store.
// Messages are cascade deleted via foreign key constraint.
func (s *GormSessionStore) DeleteSession(id string) error {
	if err := s.db.Select("Messages").Delete(&SessionModel{ID: id}).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// SaveMessage stores a message in the session history.
// If m.ID is empty, a UUID v4 is generated.
// CreatedAt is set to the current time if not provided.
func (s *GormSessionStore) SaveMessage(m *session.StoredMessage) error {
	// Generate UUID if not provided
	id := m.ID
	if id == "" {
		id = uuid.New().String()
	}

	// Use provided CreatedAt or current time
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Convert json.RawMessage to string for storage
	content := "[]"
	if m.Content != nil {
		content = string(m.Content)
	}

	model := &MessageModel{
		ID:        id,
		SessionID: m.SessionID,
		Role:      m.Role,
		Content:   content,
		Model:     m.Model,
		CreatedAt: createdAt,
	}

	if err := s.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Update the input message with generated values
	m.ID = id
	m.CreatedAt = createdAt

	return nil
}

// GetMessages retrieves all messages for a session, ordered chronologically (CreatedAt ASC).
func (s *GormSessionStore) GetMessages(sessionID string) ([]session.StoredMessage, error) {
	var models []MessageModel
	if err := s.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	messages := make([]session.StoredMessage, len(models))
	for i, model := range models {
		messages[i] = session.StoredMessage{
			ID:        model.ID,
			SessionID: model.SessionID,
			Role:      model.Role,
			Content:   json.RawMessage(model.Content),
			Model:     model.Model,
			CreatedAt: model.CreatedAt,
		}
	}

	return messages, nil
}

// GetMessagesAfter retrieves messages created after the specified time.
// Useful for fetching new messages after a summary point.
func (s *GormSessionStore) GetMessagesAfter(sessionID string, after time.Time) ([]session.StoredMessage, error) {
	var models []MessageModel
	if err := s.db.Where("session_id = ? AND created_at > ?", sessionID, after).
		Order("created_at ASC").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get messages after: %w", err)
	}

	messages := make([]session.StoredMessage, len(models))
	for i, model := range models {
		messages[i] = session.StoredMessage{
			ID:        model.ID,
			SessionID: model.SessionID,
			Role:      model.Role,
			Content:   json.RawMessage(model.Content),
			Model:     model.Model,
			CreatedAt: model.CreatedAt,
		}
	}

	return messages, nil
}

// DeleteMessages removes all messages for a session.
func (s *GormSessionStore) DeleteMessages(sessionID string) error {
	if err := s.db.Where("session_id = ?", sessionID).Delete(&MessageModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	return nil
}

// TrackReadFile records that a file was read in a session.
// If the record already exists, ReadAt is updated (upsert behavior).
func (s *GormSessionStore) TrackReadFile(sessionID, path string) error {
	model := &ReadFileModel{
		SessionID: sessionID,
		Path:      path,
		ReadAt:    time.Now(),
	}

	// Use upsert: on conflict, update ReadAt
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}, {Name: "path"}},
		DoUpdates: clause.AssignmentColumns([]string{"read_at"}),
	}).Create(model).Error; err != nil {
		return fmt.Errorf("failed to track read file: %w", err)
	}

	return nil
}

// HasReadFile checks if a file was previously read in a session.
func (s *GormSessionStore) HasReadFile(sessionID, path string) (bool, error) {
	var count int64
	if err := s.db.Model(&ReadFileModel{}).
		Where("session_id = ? AND path = ?", sessionID, path).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check read file: %w", err)
	}

	return count > 0, nil
}
