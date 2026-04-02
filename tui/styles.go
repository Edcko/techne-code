package tui

import (
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
