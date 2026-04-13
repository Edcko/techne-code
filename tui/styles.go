package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Color palette
var (
	colorPrimary   = lipgloss.Color("#7D56F4") // Purple
	colorSecondary = lipgloss.Color("#5B6071") // Gray
	colorAccent    = lipgloss.Color("#04B575") // Green
	colorError     = lipgloss.Color("#FF6B6B") // Red
	colorDim       = lipgloss.Color("#3C3C3C") // Dim gray
	colorText      = lipgloss.Color("#FAFAFA") // White
	colorBg        = lipgloss.Color("#1A1A2E") // Dark blue-black
)

// Styles
var (
	// Title style for the header
	TitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			MarginBottom(1)

	// StatusBar style at the bottom
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Background(colorPrimary).
			Padding(0, 1)

	// UserMessage style for user chat bubbles
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	// AssistantMessage style for assistant chat
	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(colorText)

	// ToolMessage style for tool execution
	ToolMessageStyle = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Italic(true)

	// ErrorMessage style
	ErrorMessageStyle = lipgloss.NewStyle().
				Foreground(colorError).
				Bold(true)

	// InputPrompt style
	InputPromptStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// DimStyle for less important text
	DimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	// HelpStyle for keybinding hints
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	// SelectedStyle for list selections
	SelectedStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// BorderStyle for panels
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary)
)

func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	if width < 10 {
		width = 10
	}
	contentWidth := width - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	var sb strings.Builder
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if isCodeBlockBorder(line) || isCodeBlockContent(line) || isDiffLine(line) {
			sb.WriteString(line)
			continue
		}
		sb.WriteString(lipgloss.Wrap(line, contentWidth, " "))
	}
	return sb.String()
}

func isCodeBlockBorder(line string) bool {
	plain := stripANSIForWrap(line)
	trimmed := strings.TrimLeft(plain, " \t")
	return strings.HasPrefix(trimmed, "┌") || strings.HasPrefix(trimmed, "└")
}

func isCodeBlockContent(line string) bool {
	plain := stripANSIForWrap(line)
	trimmed := strings.TrimLeft(plain, " \t")
	return strings.HasPrefix(trimmed, "│")
}

func isDiffLine(line string) bool {
	plain := stripANSIForWrap(line)
	if len(plain) == 0 {
		return false
	}
	return plain[0] == '+' || plain[0] == '-' || plain[0] == ' '
}

func stripANSIForWrap(s string) string {
	var sb strings.Builder
	esc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') {
				esc = false
			}
			continue
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}
