package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Mode represents the permission mode for tool execution.
type Mode string

const (
	// ModeInteractive asks the user for each permission request.
	ModeInteractive Mode = "interactive"
	// ModeAutoAllow automatically approves all permission requests.
	ModeAutoAllow Mode = "auto_allow"
	// ModeAutoDeny automatically denies all permission requests.
	ModeAutoDeny Mode = "auto_deny"
)

// Request represents a permission request from a tool.
type Request struct {
	SessionID   string          `json:"session_id"`
	ToolName    string          `json:"tool_name"`
	Action      string          `json:"action"`
	Description string          `json:"description"`
	Params      json.RawMessage `json:"params"`
}

// Response represents the user's response to a permission request.
type Response struct {
	Allowed  bool `json:"allowed"`
	Remember bool `json:"remember"` // true = grant for rest of session
}

// Service manages tool permissions for the agent.
type Service struct {
	mu           sync.RWMutex
	mode         Mode
	allowedTools map[string]bool // tools that auto-approve
	grants       map[string]bool // "sessionID:toolName:action" -> true
}

// NewService creates a new permission service.
func NewService(mode Mode, allowedTools []string) *Service {
	s := &Service{
		mode:         mode,
		allowedTools: make(map[string]bool),
		grants:       make(map[string]bool),
	}
	for _, t := range allowedTools {
		s.allowedTools[strings.ToLower(t)] = true
	}
	return s
}

// Request checks if a tool action is allowed.
// In auto_allow mode, always returns allowed.
// In auto_deny mode, always returns denied.
// In interactive mode, checks grants first, then returns pending.
// The caller (agent loop) handles the actual user interaction.
func (s *Service) Request(ctx context.Context, req Request) (Response, error) {
	// Check mode first
	switch s.mode {
	case ModeAutoAllow:
		return Response{Allowed: true}, nil
	case ModeAutoDeny:
		return Response{Allowed: false}, nil
	}

	// Check if tool is in auto-approve list
	if s.allowedTools[req.ToolName] {
		return Response{Allowed: true}, nil
	}

	// Check if tool is always safe (read-only tools)
	if !requiresPermission(req.ToolName) {
		return Response{Allowed: true}, nil
	}

	// Check session grants
	key := grantKey(req.SessionID, req.ToolName, req.Action)
	s.mu.RLock()
	allowed := s.grants[key]
	s.mu.RUnlock()
	if allowed {
		return Response{Allowed: true}, nil
	}

	// Interactive: return pending response
	// The agent loop will handle the actual user interaction
	// and call Grant() if approved
	return Response{Allowed: false, Remember: false}, nil
}

// IsAllowed checks if a tool/action is currently allowed without prompting.
func (s *Service) IsAllowed(sessionID, toolName, action string) bool {
	// Check mode
	if s.mode == ModeAutoAllow {
		return true
	}
	if s.mode == ModeAutoDeny {
		return false
	}

	// Check auto-approve list
	if s.allowedTools[toolName] {
		return true
	}

	// Check session grants
	key := grantKey(sessionID, toolName, action)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.grants[key]
}

// Grant records a permission grant for the current session.
func (s *Service) Grant(sessionID, toolName, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(sessionID, toolName, action)
	s.grants[key] = true
}

// Revoke removes a permission grant.
func (s *Service) Revoke(sessionID, toolName, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := grantKey(sessionID, toolName, action)
	delete(s.grants, key)
}

// Mode returns the current permission mode.
func (s *Service) Mode() Mode {
	return s.mode
}

// SetMode changes the permission mode.
func (s *Service) SetMode(mode Mode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// grantKey creates a unique key for session+tool+action grants.
func grantKey(sessionID, toolName, action string) string {
	return fmt.Sprintf("%s:%s:%s", sessionID, toolName, action)
}

// requiresPermission returns true if a tool requires user permission.
func requiresPermission(toolName string) bool {
	switch toolName {
	case "read_file", "glob", "grep":
		return false
	case "write_file", "edit_file", "bash", "task":
		return true
	default:
		return true
	}
}
