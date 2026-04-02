// Package db provides SQLite persistence for Techne Code using GORM.
// It implements the SessionStore interface for storing sessions, messages,
// and tracking file reads and permission grants.
package db

import (
	"time"
)

// SessionModel is the GORM model for sessions table.
type SessionModel struct {
	ID               string         `gorm:"primaryKey"`
	Title            string         `gorm:"not null;default:'New Session'"`
	Model            string         `gorm:"not null"`
	Provider         string         `gorm:"not null"`
	SummaryMessageID *string        `gorm:"column:summary_message_id"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	Messages         []MessageModel `gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for SessionModel.
func (SessionModel) TableName() string {
	return "sessions"
}

// MessageModel is the GORM model for messages table.
type MessageModel struct {
	ID        string       `gorm:"primaryKey"`
	SessionID string       `gorm:"not null;index"`
	Role      string       `gorm:"not null;size:20"`
	Content   string       `gorm:"not null;default:'[]'"` // JSON string
	Model     string       `gorm:"column:model"`
	CreatedAt time.Time    `gorm:"not null;index"`
	Session   SessionModel `gorm:"foreignKey:SessionID"`
}

// TableName returns the table name for MessageModel.
func (MessageModel) TableName() string {
	return "messages"
}

// ReadFileModel is the GORM model for read_files table.
// Tracks which files have been read in each session for context management.
type ReadFileModel struct {
	SessionID string    `gorm:"primaryKey"`
	Path      string    `gorm:"primaryKey"`
	ReadAt    time.Time `gorm:"not null"`
}

// TableName returns the table name for ReadFileModel.
func (ReadFileModel) TableName() string {
	return "read_files"
}

// PermissionGrantModel is the GORM model for permission_grants table.
// Tracks which tool/action combinations have been granted permission in a session.
type PermissionGrantModel struct {
	SessionID string    `gorm:"primaryKey"`
	ToolName  string    `gorm:"primaryKey;column:tool_name"`
	Action    string    `gorm:"primaryKey"`
	GrantedAt time.Time `gorm:"not null"`
}

// TableName returns the table name for PermissionGrantModel.
func (PermissionGrantModel) TableName() string {
	return "permission_grants"
}
