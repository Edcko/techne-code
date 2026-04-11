package components

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Edcko/techne-code/pkg/event"
)

type PermissionDialog struct {
	visible  bool
	request  *event.PermissionRequestData
	response chan<- event.PermissionResponseData
	focused  int
}

func NewPermissionDialog() *PermissionDialog {
	return &PermissionDialog{}
}

func (d *PermissionDialog) Show(req event.PermissionRequestData) {
	d.visible = true
	d.request = &req
	d.response = req.Response
	d.focused = 0
}

func (d *PermissionDialog) Dismiss() {
	d.visible = false
	d.request = nil
	d.response = nil
	d.focused = 0
}

func (d *PermissionDialog) Visible() bool {
	return d.visible
}

func (d *PermissionDialog) HandleKey(msg tea.KeyPressMsg) bool {
	if !d.visible {
		return false
	}

	switch msg.String() {
	case "y", "Y":
		d.respond(true, false)
		return true
	case "a", "A":
		d.respond(true, true)
		return true
	case "n", "N", "escape":
		d.respond(false, false)
		return true
	case "tab":
		d.focused = (d.focused + 1) % 3
		return true
	case "enter":
		switch d.focused {
		case 0:
			d.respond(true, false)
		case 1:
			d.respond(true, true)
		case 2:
			d.respond(false, false)
		}
		return true
	}

	return true
}

func (d *PermissionDialog) respond(allowed, remember bool) {
	if d.response != nil {
		d.response <- event.PermissionResponseData{
			Allowed:  allowed,
			Remember: remember,
		}
	}
	d.Dismiss()
}

func (d *PermissionDialog) View(width int) string {
	if !d.visible || d.request == nil {
		return ""
	}

	dialogWidth := 60
	if width > 0 && dialogWidth > width-4 {
		dialogWidth = width - 4
	}

	var b strings.Builder

	b.WriteString("┌")
	b.WriteString(strings.Repeat("─", dialogWidth-2))
	b.WriteString("┐")
	b.WriteString("\n")

	b.WriteString("│ ⚠ Permission Required")
	padding := dialogWidth - 2 - 23
	if padding > 0 {
		b.WriteString(strings.Repeat(" ", padding))
	}
	b.WriteString("│")
	b.WriteString("\n")

	b.WriteString("├")
	b.WriteString(strings.Repeat("─", dialogWidth-2))
	b.WriteString("┤")
	b.WriteString("\n")

	toolLine := fmt.Sprintf("│ Tool: %s", d.request.ToolName)
	if len(toolLine) < dialogWidth-1 {
		toolLine += strings.Repeat(" ", dialogWidth-1-len(toolLine))
	}
	toolLine += "│"
	b.WriteString(toolLine)
	b.WriteString("\n")

	actionLine := fmt.Sprintf("│ Action: %s", d.request.Action)
	if len(actionLine) < dialogWidth-1 {
		actionLine += strings.Repeat(" ", dialogWidth-1-len(actionLine))
	}
	actionLine += "│"
	b.WriteString(actionLine)
	b.WriteString("\n")

	if d.request.Description != "" {
		descLines := wrapText(d.request.Description, dialogWidth-4)
		for _, line := range descLines {
			trimmed := fmt.Sprintf("│ %s", line)
			if len(trimmed) < dialogWidth-1 {
				trimmed += strings.Repeat(" ", dialogWidth-1-len(trimmed))
			}
			trimmed += "│"
			b.WriteString(trimmed)
			b.WriteString("\n")
		}
	}

	if d.request.Params != nil {
		var pretty json.RawMessage = d.request.Params
		paramStr := string(pretty)
		if len(paramStr) > 200 {
			paramStr = paramStr[:200] + "..."
		}
		paramLines := wrapText(paramStr, dialogWidth-4)
		for _, line := range paramLines {
			pl := fmt.Sprintf("│ %s", line)
			if len(pl) < dialogWidth-1 {
				pl += strings.Repeat(" ", dialogWidth-1-len(pl))
			}
			pl += "│"
			b.WriteString(pl)
			b.WriteString("\n")
		}
	}

	b.WriteString("├")
	b.WriteString(strings.Repeat("─", dialogWidth-2))
	b.WriteString("┤")
	b.WriteString("\n")

	options := []string{
		"[Y] Allow once",
		"[A] Always allow (this session)",
		"[N] Deny",
	}
	for i, opt := range options {
		prefix := "│ "
		if i == d.focused {
			prefix = "│ ► "
		}
		optLine := prefix + opt
		if len(optLine) < dialogWidth-1 {
			optLine += strings.Repeat(" ", dialogWidth-1-len(optLine))
		}
		optLine += "│"
		b.WriteString(optLine)
		b.WriteString("\n")
	}

	b.WriteString("└")
	b.WriteString(strings.Repeat("─", dialogWidth-2))
	b.WriteString("┘")

	return b.String()
}

func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if len(line) <= maxWidth {
			lines = append(lines, line)
			continue
		}

		for len(line) > maxWidth {
			breakIdx := maxWidth
			for i := maxWidth; i > 0; i-- {
				if line[i] == ' ' {
					breakIdx = i
					break
				}
			}
			lines = append(lines, line[:breakIdx])
			line = line[breakIdx:]
			if len(line) > 0 && line[0] == ' ' {
				line = line[1:]
			}
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}

	if len(lines) == 0 {
		lines = append(lines, "")
	}

	return lines
}
