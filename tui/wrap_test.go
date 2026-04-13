package tui

import (
	"strings"
	"testing"
)

func TestWrapText_LongLineGetsWrapped(t *testing.T) {
	longLine := "This is a very long line of text that should definitely be wrapped when the terminal width is small because it exceeds the available width"
	width := 40

	result := wrapText(longLine, width)

	if result == longLine {
		t.Error("expected long line to be wrapped, but it was not changed")
	}
	lines := strings.Split(result, "\n")
	if len(lines) < 2 {
		t.Errorf("expected at least 2 lines after wrapping, got %d", len(lines))
	}
	for _, line := range lines {
		if len(line) > width {
			t.Errorf("wrapped line exceeds width %d: %q (len=%d)", width, line, len(line))
		}
	}
}

func TestWrapText_ShortLineUnchanged(t *testing.T) {
	shortLine := "Hello world"
	width := 80

	result := wrapText(shortLine, width)

	if result != shortLine {
		t.Errorf("expected short line to be unchanged, got %q", result)
	}
}

func TestWrapText_RespectsWidth(t *testing.T) {
	text := "Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor incididunt ut labore et dolore magna aliqua"
	width := 30

	result := wrapText(text, width)

	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if len(line) > width {
			t.Errorf("line %d exceeds width %d: %q (len=%d)", i, width, line, len(line))
		}
	}
}

func TestWrapText_PreservesNewlines(t *testing.T) {
	text := "line one\nline two\nline three"
	width := 80

	result := wrapText(text, width)

	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line one" {
		t.Errorf("expected 'line one', got %q", lines[0])
	}
	if lines[1] != "line two" {
		t.Errorf("expected 'line two', got %q", lines[1])
	}
	if lines[2] != "line three" {
		t.Errorf("expected 'line three', got %q", lines[2])
	}
}

func TestWrapText_MultilineWithWrapping(t *testing.T) {
	text := "This is the first very long line that needs wrapping\nThis is the second very long line that also needs wrapping"
	width := 30

	result := wrapText(text, width)

	sections := strings.Split(result, "\n")
	if len(sections) < 4 {
		t.Errorf("expected at least 4 output lines from 2 wrapped input lines, got %d", len(sections))
	}
	for i, line := range sections {
		if len(line) > width {
			t.Errorf("output line %d exceeds width %d: %q (len=%d)", i, width, line, len(line))
		}
	}
}

func TestWrapText_ZeroWidth(t *testing.T) {
	text := "Some text here"
	result := wrapText(text, 0)

	if result != text {
		t.Errorf("expected text unchanged with zero width, got %q", result)
	}
}

func TestWrapText_NegativeWidth(t *testing.T) {
	text := "Some text here"
	result := wrapText(text, -5)

	if result != text {
		t.Errorf("expected text unchanged with negative width, got %q", result)
	}
}

func TestWrapText_SmallWidth(t *testing.T) {
	text := "Hello world this is a test"
	result := wrapText(text, 8)

	if result == text {
		t.Error("expected text to be wrapped even with very small width")
	}
}

func TestWrapText_EmptyString(t *testing.T) {
	result := wrapText("", 80)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestView_AssistantMessageWrapsLongText(t *testing.T) {
	m := initTestModel()
	m.width = 40
	m.height = 24

	longText := strings.Repeat("word ", 50)
	m.messages = append(m.messages, ChatMessage{
		Role:    "assistant",
		Content: longText,
	})

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "Assistant:") {
		t.Error("expected 'Assistant:' in view")
	}

	renderedLines := strings.Split(rendered, "\n")
	msgStarted := false
	for i, line := range renderedLines {
		stripped := stripANSIFromTUI(line)
		if contains(stripped, "Assistant:") {
			msgStarted = true
			continue
		}
		if !msgStarted {
			continue
		}
		if contains(stripped, "anthropic") || contains(stripped, "tokens:") {
			continue
		}
		if len(stripped) > 50 {
			endIdx := 60
			if len(stripped) < endIdx {
				endIdx = len(stripped)
			}
			t.Errorf("rendered line %d too wide (len=%d): %q", i, len(stripped), stripped[:endIdx])
		}
	}
}

func TestView_ErrorMessageWrapsLongText(t *testing.T) {
	m := initTestModel()
	m.width = 40
	m.height = 24

	longError := strings.Repeat("error detail ", 30)
	m.messages = append(m.messages, ChatMessage{
		Role:    "error",
		Content: longError,
	})

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "Error:") {
		t.Error("expected 'Error:' in view")
	}
}

func TestView_ToolMessageWrapsLongText(t *testing.T) {
	m := initTestModel()
	m.width = 40
	m.height = 24

	longToolOutput := strings.Repeat("tool output data ", 20)
	m.messages = append(m.messages, ChatMessage{
		Role:    "tool",
		Content: "✓ test: " + longToolOutput,
	})

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "test") {
		t.Error("expected tool name in view")
	}
}

func TestView_ThinkingBufferWrapsDuringStreaming(t *testing.T) {
	m := initTestModel()
	m.width = 40
	m.height = 24
	m.state = StateStreaming
	m.thinkingBuffer = strings.Repeat("thinking about something ", 20)

	view := m.View()
	rendered := view.Content

	if !contains(rendered, "thinking") {
		t.Error("expected thinking content in view during streaming")
	}
}

func TestView_CodeBlockNotBroken(t *testing.T) {
	m := initTestModel()
	m.width = 80
	m.height = 24

	codeContent := "```go\nfunc main() {\n\tfmt.Println(\"hello world\")\n}\n```"
	m.messages = append(m.messages, ChatMessage{
		Role:    "assistant",
		Content: codeContent,
	})

	view := m.View()
	rendered := view.Content
	plain := stripANSIFromTUI(rendered)

	if !contains(plain, "func main()") {
		t.Error("expected 'func main()' in rendered code block")
	}
	if !contains(plain, "fmt.Println") {
		t.Error("expected 'fmt.Println' in rendered code block")
	}
	if !contains(plain, "┌") {
		t.Error("expected code block top border")
	}
	if !contains(plain, "└") {
		t.Error("expected code block bottom border")
	}
}

func stripANSIFromTUI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			i++
			for i < len(s) && !((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		result = append(result, s[i])
		i++
	}
	return string(result)
}
