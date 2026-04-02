package permission

import (
	"context"
	"testing"
)

func TestNewService(t *testing.T) {
	s := NewService(ModeInteractive, []string{"git_status"})
	if s == nil {
		t.Fatal("expected non-nil service")
	}
	if s.Mode() != ModeInteractive {
		t.Errorf("expected mode interactive, got %s", s.Mode())
	}
}

func TestAutoAllowMode(t *testing.T) {
	s := NewService(ModeAutoAllow, nil)
	resp, err := s.Request(context.Background(), Request{
		ToolName: "bash",
		Action:   "execute",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Allowed {
		t.Error("expected allowed in auto_allow mode")
	}
}

func TestAutoDenyMode(t *testing.T) {
	s := NewService(ModeAutoDeny, nil)
	resp, err := s.Request(context.Background(), Request{
		ToolName: "read_file",
		Action:   "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Allowed {
		t.Error("expected denied in auto_deny mode")
	}
}

func TestAllowedToolsList(t *testing.T) {
	s := NewService(ModeInteractive, []string{"git_status", "ls"})
	resp, err := s.Request(context.Background(), Request{
		ToolName: "git_status",
		Action:   "run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Allowed {
		t.Error("expected git_status to be auto-approved")
	}
}

func TestSessionGrant(t *testing.T) {
	s := NewService(ModeInteractive, nil)

	// Before grant: bash requires permission, should be denied
	resp, _ := s.Request(context.Background(), Request{
		SessionID: "sess1",
		ToolName:  "bash",
		Action:    "execute",
	})
	if resp.Allowed {
		t.Error("expected bash to require permission")
	}

	// Grant permission
	s.Grant("sess1", "bash", "execute")

	// After grant: should be allowed
	resp, _ = s.Request(context.Background(), Request{
		SessionID: "sess1",
		ToolName:  "bash",
		Action:    "execute",
	})
	if !resp.Allowed {
		t.Error("expected bash to be allowed after grant")
	}
}

func TestReadOnlyToolsAutoApprove(t *testing.T) {
	s := NewService(ModeInteractive, nil)

	readOnlyTools := []string{"read_file", "glob", "grep"}
	for _, tool := range readOnlyTools {
		resp, err := s.Request(context.Background(), Request{
			ToolName: tool,
			Action:   "read",
		})
		if err != nil {
			t.Fatal(err)
		}
		if !resp.Allowed {
			t.Errorf("expected %s to be auto-approved (read-only)", tool)
		}
	}
}

func TestWriteToolsRequirePermission(t *testing.T) {
	s := NewService(ModeInteractive, nil)

	writeTools := []string{"write_file", "edit_file", "bash"}
	for _, tool := range writeTools {
		resp, _ := s.Request(context.Background(), Request{
			ToolName: tool,
			Action:   "write",
		})
		if resp.Allowed {
			t.Errorf("expected %s to require permission", tool)
		}
	}
}

func TestIsAllowed(t *testing.T) {
	s := NewService(ModeInteractive, []string{"ls"})

	if !s.IsAllowed("sess1", "ls", "run") {
		t.Error("expected ls to be allowed via allowedTools")
	}
	if s.IsAllowed("sess1", "bash", "execute") {
		t.Error("expected bash to not be allowed")
	}

	s.Grant("sess1", "bash", "execute")
	if !s.IsAllowed("sess1", "bash", "execute") {
		t.Error("expected bash to be allowed after grant")
	}
}

func TestSetMode(t *testing.T) {
	s := NewService(ModeInteractive, nil)
	s.SetMode(ModeAutoAllow)
	if s.Mode() != ModeAutoAllow {
		t.Errorf("expected auto_allow, got %s", s.Mode())
	}
}

func TestGrantIsSessionScoped(t *testing.T) {
	s := NewService(ModeInteractive, nil)
	s.Grant("sess1", "bash", "execute")

	// sess1 should be allowed
	if !s.IsAllowed("sess1", "bash", "execute") {
		t.Error("expected sess1 to be allowed")
	}

	// sess2 should NOT be allowed
	if s.IsAllowed("sess2", "bash", "execute") {
		t.Error("expected sess2 to NOT be allowed (different session)")
	}
}
