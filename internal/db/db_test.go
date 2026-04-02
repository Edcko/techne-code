package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Edcko/techne-code/pkg/session"
	"gorm.io/gorm"
)

// TestOpen creates database and runs migrations
func TestOpen(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify tables were created
	var tables []string
	db.Raw("SELECT name FROM sqlite_master WHERE type='table'").Scan(&tables)

	expectedTables := map[string]bool{
		"sessions":          false,
		"messages":          false,
		"read_files":        false,
		"permission_grants": false,
	}

	for _, table := range tables {
		if _, exists := expectedTables[table]; exists {
			expectedTables[table] = true
		}
	}

	for table, found := range expectedTables {
		if !found {
			t.Errorf("table %s was not created", table)
		}
	}
}

// TestSessionStore_CreateSession tests session creation
func TestSessionStore_CreateSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test Session",
		Model:    "gpt-4",
		Provider: "openai",
	}

	if err := store.CreateSession(sess); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	// Verify ID was generated
	if sess.ID == "" {
		t.Error("session ID was not generated")
	}

	// Verify timestamps were set
	if sess.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if sess.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}

	// Verify session can be retrieved
	retrieved, err := store.GetSession(sess.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("retrieved session is nil")
	}
	if retrieved.Title != sess.Title {
		t.Errorf("Title = %q, want %q", retrieved.Title, sess.Title)
	}
}

// TestSessionStore_GetSession tests session retrieval
func TestSessionStore_GetSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	// Test getting non-existent session
	sess, err := store.GetSession("non-existent")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if sess != nil {
		t.Error("expected nil for non-existent session")
	}

	// Create and retrieve session
	created := &session.Session{
		Title:    "Test Session",
		Model:    "claude-3",
		Provider: "anthropic",
	}
	store.CreateSession(created)

	retrieved, err := store.GetSession(created.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("retrieved session is nil")
	}
	if retrieved.ID != created.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, created.ID)
	}
}

// TestSessionStore_ListSessions tests listing sessions
func TestSessionStore_ListSessions(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		sess := &session.Session{
			Title:    "Session",
			Model:    "model",
			Provider: "provider",
		}
		if err := store.CreateSession(sess); err != nil {
			t.Fatalf("CreateSession() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessions) != 3 {
		t.Errorf("len(sessions) = %d, want 3", len(sessions))
	}

	// Verify ordering (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		if sessions[i].UpdatedAt.Before(sessions[i+1].UpdatedAt) {
			t.Error("sessions not ordered by UpdatedAt DESC")
		}
	}
}

// TestSessionStore_UpdateSessionTitle tests updating session title
func TestSessionStore_UpdateSessionTitle(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Original Title",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	newTitle := "Updated Title"
	if err := store.UpdateSessionTitle(sess.ID, newTitle); err != nil {
		t.Fatalf("UpdateSessionTitle() error = %v", err)
	}

	retrieved, _ := store.GetSession(sess.ID)
	if retrieved.Title != newTitle {
		t.Errorf("Title = %q, want %q", retrieved.Title, newTitle)
	}
}

// TestSessionStore_UpdateSessionSummary tests updating session summary
func TestSessionStore_UpdateSessionSummary(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	summaryID := "msg-123"
	if err := store.UpdateSessionSummary(sess.ID, summaryID); err != nil {
		t.Fatalf("UpdateSessionSummary() error = %v", err)
	}

	retrieved, _ := store.GetSession(sess.ID)
	if retrieved.SummaryMessageID == nil {
		t.Fatal("SummaryMessageID is nil")
	}
	if *retrieved.SummaryMessageID != summaryID {
		t.Errorf("SummaryMessageID = %q, want %q", *retrieved.SummaryMessageID, summaryID)
	}
}

// TestSessionStore_DeleteSession tests session deletion
func TestSessionStore_DeleteSession(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	// Add a message to verify cascade delete
	store.SaveMessage(&session.StoredMessage{
		SessionID: sess.ID,
		Role:      "user",
		Content:   json.RawMessage(`"hello"`),
	})

	if err := store.DeleteSession(sess.ID); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	// Verify session is deleted
	retrieved, _ := store.GetSession(sess.ID)
	if retrieved != nil {
		t.Error("session was not deleted")
	}

	// Verify messages are also deleted (cascade)
	messages, _ := store.GetMessages(sess.ID)
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(messages))
	}
}

// TestSessionStore_SaveMessage tests saving messages
func TestSessionStore_SaveMessage(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	// Create session first
	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	msg := &session.StoredMessage{
		SessionID: sess.ID,
		Role:      "user",
		Content:   json.RawMessage(`[{"type":"text","text":"Hello"}]`),
		Model:     "gpt-4",
	}

	if err := store.SaveMessage(msg); err != nil {
		t.Fatalf("SaveMessage() error = %v", err)
	}

	// Verify ID was generated
	if msg.ID == "" {
		t.Error("message ID was not generated")
	}

	// Verify CreatedAt was set
	if msg.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
}

// TestSessionStore_GetMessages tests retrieving messages
func TestSessionStore_GetMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	// Create multiple messages
	for i := 0; i < 3; i++ {
		store.SaveMessage(&session.StoredMessage{
			SessionID: sess.ID,
			Role:      "user",
			Content:   json.RawMessage(`"message"`),
		})
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	messages, err := store.GetMessages(sess.ID)
	if err != nil {
		t.Fatalf("GetMessages() error = %v", err)
	}

	if len(messages) != 3 {
		t.Errorf("len(messages) = %d, want 3", len(messages))
	}

	// Verify ordering (chronological)
	for i := 0; i < len(messages)-1; i++ {
		if messages[i].CreatedAt.After(messages[i+1].CreatedAt) {
			t.Error("messages not ordered chronologically")
		}
	}
}

// TestSessionStore_GetMessagesAfter tests retrieving messages after a time
func TestSessionStore_GetMessagesAfter(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	// Create first message
	store.SaveMessage(&session.StoredMessage{
		SessionID: sess.ID,
		Role:      "user",
		Content:   json.RawMessage(`"first"`),
	})

	// Wait and record time
	time.Sleep(20 * time.Millisecond)
	afterTime := time.Now()
	time.Sleep(20 * time.Millisecond)

	// Create second message
	store.SaveMessage(&session.StoredMessage{
		SessionID: sess.ID,
		Role:      "user",
		Content:   json.RawMessage(`"second"`),
	})

	messages, err := store.GetMessagesAfter(sess.ID, afterTime)
	if err != nil {
		t.Fatalf("GetMessagesAfter() error = %v", err)
	}

	if len(messages) != 1 {
		t.Errorf("len(messages) = %d, want 1", len(messages))
	}
}

// TestSessionStore_DeleteMessages tests deleting messages
func TestSessionStore_DeleteMessages(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	store.SaveMessage(&session.StoredMessage{
		SessionID: sess.ID,
		Role:      "user",
		Content:   json.RawMessage(`"test"`),
	})

	if err := store.DeleteMessages(sess.ID); err != nil {
		t.Fatalf("DeleteMessages() error = %v", err)
	}

	messages, _ := store.GetMessages(sess.ID)
	if len(messages) != 0 {
		t.Errorf("expected 0 messages after delete, got %d", len(messages))
	}
}

// TestSessionStore_TrackReadFile tests tracking read files
func TestSessionStore_TrackReadFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sessionID := "test-session"
	path := "/path/to/file.go"

	if err := store.TrackReadFile(sessionID, path); err != nil {
		t.Fatalf("TrackReadFile() error = %v", err)
	}

	hasRead, err := store.HasReadFile(sessionID, path)
	if err != nil {
		t.Fatalf("HasReadFile() error = %v", err)
	}

	if !hasRead {
		t.Error("expected file to be tracked as read")
	}
}

// TestSessionStore_HasReadFile tests checking read files
func TestSessionStore_HasReadFile(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sessionID := "test-session"
	path := "/path/to/file.go"

	// Check non-tracked file
	hasRead, err := store.HasReadFile(sessionID, path)
	if err != nil {
		t.Fatalf("HasReadFile() error = %v", err)
	}
	if hasRead {
		t.Error("expected file to not be tracked")
	}

	// Track and check
	store.TrackReadFile(sessionID, path)
	hasRead, _ = store.HasReadFile(sessionID, path)
	if !hasRead {
		t.Error("expected file to be tracked after TrackReadFile")
	}
}

// TestSessionStore_TrackReadFile_Upsert tests upsert behavior
func TestSessionStore_TrackReadFile_Upsert(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	sessionID := "test-session"
	path := "/path/to/file.go"

	// Track first time
	store.TrackReadFile(sessionID, path)
	time.Sleep(20 * time.Millisecond)

	// Track again (should update ReadAt, not create duplicate)
	store.TrackReadFile(sessionID, path)

	// Verify only one record exists
	var count int64
	db.Model(&ReadFileModel{}).
		Where("session_id = ? AND path = ?", sessionID, path).
		Count(&count)

	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// TestCascadeDelete tests that deleting a session cascades to messages
func TestCascadeDelete(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := NewSessionStore(db)

	// Create session
	sess := &session.Session{
		Title:    "Test",
		Model:    "model",
		Provider: "provider",
	}
	store.CreateSession(sess)

	// Create messages
	for i := 0; i < 5; i++ {
		store.SaveMessage(&session.StoredMessage{
			SessionID: sess.ID,
			Role:      "user",
			Content:   json.RawMessage(`"test"`),
		})
	}

	// Verify messages exist
	messages, _ := store.GetMessages(sess.ID)
	if len(messages) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(messages))
	}

	// Delete session
	store.DeleteSession(sess.ID)

	// Verify messages are deleted
	var count int64
	db.Model(&MessageModel{}).Where("session_id = ?", sess.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 messages after cascade delete, got %d", count)
	}
}

// setupTestDB creates a test database in a temporary directory
func setupTestDB(t *testing.T) (db *gorm.DB, cleanup func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	var err error
	db, err = Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	cleanup = func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}

	return db, cleanup
}
