package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Edcko/techne-code/pkg/event"
)

func TestNewPermissionDialog_NotVisible(t *testing.T) {
	d := NewPermissionDialog()
	if d.Visible() {
		t.Error("new dialog should not be visible")
	}
}

func TestPermissionDialog_ShowSetsVisible(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName:    "bash",
		Action:      "execute",
		Description: "Run a shell command",
		Response:    ch,
	}

	d.Show(req)

	if !d.Visible() {
		t.Error("dialog should be visible after Show")
	}
	if d.request == nil {
		t.Error("request should be set after Show")
	}
}

func TestPermissionDialog_DismissClearsState(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)
	d.Dismiss()

	if d.Visible() {
		t.Error("dialog should not be visible after Dismiss")
	}
	if d.request != nil {
		t.Error("request should be nil after Dismiss")
	}
}

func TestPermissionDialog_AllowOnce(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)

	handled := d.HandleKey(testKeyPress("y"))
	if !handled {
		t.Error("should handle key")
	}

	if d.Visible() {
		t.Error("dialog should dismiss after response")
	}

	select {
	case resp := <-ch:
		if !resp.Allowed {
			t.Error("expected allowed=true")
		}
		if resp.Remember {
			t.Error("expected remember=false for allow once")
		}
	default:
		t.Error("expected response on channel")
	}
}

func TestPermissionDialog_AllowAlways(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)

	d.HandleKey(testKeyPress("a"))

	select {
	case resp := <-ch:
		if !resp.Allowed {
			t.Error("expected allowed=true")
		}
		if !resp.Remember {
			t.Error("expected remember=true for always allow")
		}
	default:
		t.Error("expected response on channel")
	}
}

func TestPermissionDialog_Deny(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)

	d.HandleKey(testKeyPress("n"))

	select {
	case resp := <-ch:
		if resp.Allowed {
			t.Error("expected allowed=false")
		}
		if resp.Remember {
			t.Error("expected remember=false for deny")
		}
	default:
		t.Error("expected response on channel")
	}
}

func TestPermissionDialog_DenyWithEscape(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)

	d.HandleKey(testKeyPress("escape"))

	select {
	case resp := <-ch:
		if resp.Allowed {
			t.Error("expected allowed=false on escape")
		}
	default:
		t.Error("expected response on channel")
	}
}

func TestPermissionDialog_HandleKeyWhenHidden(t *testing.T) {
	d := NewPermissionDialog()
	handled := d.HandleKey(testKeyPress("y"))
	if handled {
		t.Error("should not handle key when not visible")
	}
}

func TestPermissionDialog_TabCyclesFocus(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)

	if d.focused != 0 {
		t.Errorf("expected initial focus 0, got %d", d.focused)
	}

	d.HandleKey(testKeyPress("tab"))
	if d.focused != 1 {
		t.Errorf("expected focus 1 after tab, got %d", d.focused)
	}

	d.HandleKey(testKeyPress("tab"))
	if d.focused != 2 {
		t.Errorf("expected focus 2 after tab, got %d", d.focused)
	}

	d.HandleKey(testKeyPress("tab"))
	if d.focused != 0 {
		t.Errorf("expected focus 0 after tab wrap, got %d", d.focused)
	}
}

func TestPermissionDialog_EnterSelectsFocusedOption(t *testing.T) {
	tests := []struct {
		name      string
		focusIdx  int
		wantAllow bool
		wantRemem bool
	}{
		{"enter on Allow once", 0, true, false},
		{"enter on Always allow", 1, true, true},
		{"enter on Deny", 2, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewPermissionDialog()
			ch := make(chan event.PermissionResponseData, 1)
			req := event.PermissionRequestData{
				ToolName: "bash",
				Action:   "execute",
				Response: ch,
			}

			d.Show(req)
			d.focused = tt.focusIdx

			d.HandleKey(testKeyPress("enter"))

			select {
			case resp := <-ch:
				if resp.Allowed != tt.wantAllow {
					t.Errorf("expected allowed=%v, got %v", tt.wantAllow, resp.Allowed)
				}
				if resp.Remember != tt.wantRemem {
					t.Errorf("expected remember=%v, got %v", tt.wantRemem, resp.Remember)
				}
			default:
				t.Error("expected response on channel")
			}
		})
	}
}

func TestPermissionDialog_ViewEmptyWhenHidden(t *testing.T) {
	d := NewPermissionDialog()
	view := d.View(80)
	if view != "" {
		t.Errorf("expected empty view when hidden, got %q", view)
	}
}

func TestPermissionDialog_ViewShowsContentWhenVisible(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName:    "write_file",
		Action:      "write",
		Description: "Write content to a file",
		Response:    ch,
	}

	d.Show(req)
	view := d.View(80)

	if view == "" {
		t.Error("expected non-empty view when visible")
	}

	checks := []string{"write_file", "write", "Write content to a file"}
	for _, check := range checks {
		if !containsString(view, check) {
			t.Errorf("expected view to contain %q", check)
		}
	}
}

func TestPermissionDialog_ViewShowsOptions(t *testing.T) {
	d := NewPermissionDialog()
	ch := make(chan event.PermissionResponseData, 1)
	req := event.PermissionRequestData{
		ToolName: "bash",
		Action:   "execute",
		Response: ch,
	}

	d.Show(req)
	view := d.View(80)

	checks := []string{"Allow once", "Always allow", "Deny"}
	for _, check := range checks {
		if !containsString(view, check) {
			t.Errorf("expected view to contain option %q", check)
		}
	}
}

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		expected int
	}{
		{"short text", "hello", 80, 1},
		{"exact width", "hello", 5, 1},
		{"needs wrap", "hello world foo bar", 8, 3},
		{"empty", "", 80, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := wrapText(tt.input, tt.maxWidth)
			if len(lines) != tt.expected {
				t.Errorf("expected %d lines, got %d: %v", tt.expected, len(lines), lines)
			}
		})
	}
}

func containsString(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func testKeyPress(s string) tea.KeyPressMsg {
	var code rune
	switch s {
	case "enter":
		code = tea.KeyEnter
	case "tab":
		code = tea.KeyTab
	case "escape":
		code = tea.KeyEscape
	default:
		if len(s) > 0 {
			code = rune(s[0])
		}
	}
	return tea.KeyPressMsg{Text: s, Code: code}
}
