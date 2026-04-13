package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/internal/config"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
	"github.com/Edcko/techne-code/tui/components"
	"github.com/Edcko/techne-code/tui/diff"
	"github.com/Edcko/techne-code/tui/markdown"

	"charm.land/lipgloss/v2"
)

type State int

const (
	StateInit State = iota
	StateChatting
	StateStreaming
	StateExiting
)

type ChatMessage struct {
	Role     string
	Content  string
	Thinking string
	Diff     string
}

type Model struct {
	state         State
	cfg           *config.Config
	agent         *agent.Agent
	client        *llm.Client
	store         session.SessionStore
	bus           event.EventBus
	unsub         func()
	skillRegistry skill.SkillRegistry
	toolsEnabled  bool
	permDialog    *components.PermissionDialog

	program   *tea.Program
	programMu sync.RWMutex

	messages         []ChatMessage
	currentStreaming *int
	thinkingBuffer   string
	inputBuf         *InputBuffer
	inputHistory     []string
	historyIndex     int
	historyDraft     string
	statusText       string
	sessionID        string
	tokenUsage       *event.TokenUsageData
	err              error

	width  int
	height int

	ctx    context.Context
	cancel context.CancelFunc
}

func NewModel(
	cfg *config.Config,
	agentClient *llm.Client,
	store session.SessionStore,
	registry tool.ToolRegistry,
	perm *permission.Service,
	bus event.EventBus,
	skillRegistry skill.SkillRegistry,
	toolsEnabled bool,
	sessionID string,
) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	ag := agent.New(agentClient, store, registry, perm, bus)
	ag.WithSkills(skillRegistry)
	return &Model{
		state:         StateInit,
		cfg:           cfg,
		client:        agentClient,
		store:         store,
		bus:           bus,
		skillRegistry: skillRegistry,
		agent:         ag,
		toolsEnabled:  toolsEnabled,
		ctx:           ctx,
		cancel:        cancel,
		messages:      []ChatMessage{},
		inputBuf:      newInputBuffer(),
		inputHistory:  []string{},
		historyIndex:  -1,
		sessionID:     sessionID,
		permDialog:    components.NewPermissionDialog(),
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.programMu.Lock()
	m.program = p
	m.programMu.Unlock()
}

func (m *Model) getProgram() *tea.Program {
	m.programMu.RLock()
	defer m.programMu.RUnlock()
	return m.program
}

func (m *Model) Init() tea.Cmd {
	m.unsub = m.bus.Subscribe(func(e event.Event) {
		p := m.getProgram()
		if p == nil {
			return
		}

		switch e.Type {
		case event.EventMessageDelta:
			if data, ok := e.Data.(event.ThinkingDeltaData); ok {
				p.Send(thinkingDeltaMsg{text: data.Text})
				return
			}
			if data, ok := e.Data.(event.MessageDeltaData); ok {
				p.Send(messageDeltaMsg{text: data.Text})
			}
		case event.EventToolStart:
			if data, ok := e.Data.(event.ToolStartData); ok {
				p.Send(toolStartMsg{name: data.ToolName})
			}
		case event.EventToolResult:
			if data, ok := e.Data.(event.ToolResultData); ok {
				p.Send(toolResultMsg{name: data.ToolName, content: data.Content, isError: data.IsError, diffData: data.Diff})
			}
		case event.EventError:
			if data, ok := e.Data.(event.ErrorData); ok {
				p.Send(errorMsg{message: data.Message, fatal: data.Fatal})
			}
		case event.EventDone:
			p.Send(doneMsg{})

		case event.EventTokenUsage:
			if data, ok := e.Data.(event.TokenUsageData); ok {
				p.Send(tokenUsageMsg{data: data})
			}

		case event.EventPermissionReq:
			if data, ok := e.Data.(event.PermissionRequestData); ok {
				p.Send(permissionRequestMsg{data: data})
			}
		}
	})

	if m.sessionID != "" {
		sess, err := m.store.GetSession(m.sessionID)
		if err != nil {
			m.err = fmt.Errorf("failed to load session %s: %w", m.sessionID, err)
			m.state = StateExiting
			return nil
		}
		if sess == nil {
			m.err = fmt.Errorf("session %s not found", m.sessionID)
			m.state = StateExiting
			return nil
		}
		stored, err := m.store.GetMessages(m.sessionID)
		if err != nil {
			m.err = fmt.Errorf("failed to load messages for session %s: %w", m.sessionID, err)
			m.state = StateExiting
			return nil
		}
		for _, msg := range stored {
			var blocks []provider.ContentBlock
			if err := json.Unmarshal(msg.Content, &blocks); err == nil {
				for _, block := range blocks {
					if block.Type == provider.BlockText && block.Text != "" {
						m.messages = append(m.messages, ChatMessage{
							Role:    msg.Role,
							Content: block.Text,
						})
					}
				}
			}
		}
		m.cfg.DefaultModel = sess.Model
		m.cfg.DefaultProvider = sess.Provider
	} else {
		sess := &session.Session{
			Title:    "New Session",
			Model:    m.cfg.DefaultModel,
			Provider: m.cfg.DefaultProvider,
		}
		if err := m.store.CreateSession(sess); err != nil {
			m.err = err
			m.state = StateExiting
			return nil
		}
		m.sessionID = sess.ID
	}

	m.state = StateChatting
	m.statusText = fmt.Sprintf("%s/%s", m.cfg.DefaultProvider, m.cfg.DefaultModel)
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.PasteMsg:
		if m.state == StateChatting {
			m.inputBuf.InsertPaste(msg.Content)
		}
		return m, nil

	case thinkingDeltaMsg:
		m.thinkingBuffer += msg.text
		return m, nil

	case messageDeltaMsg:
		if m.currentStreaming != nil && *m.currentStreaming < len(m.messages) {
			m.messages[*m.currentStreaming].Content += msg.text
		} else {
			idx := len(m.messages)
			m.messages = append(m.messages, ChatMessage{
				Role:     "assistant",
				Content:  msg.text,
				Thinking: m.thinkingBuffer,
			})
			m.currentStreaming = &idx
			m.thinkingBuffer = ""
		}
		return m, nil

	case toolStartMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    "tool",
			Content: fmt.Sprintf("▶ %s", msg.name),
		})
		m.currentStreaming = nil
		return m, nil

	case toolResultMsg:
		prefix := "✓"
		if msg.isError {
			prefix = "✗"
		}
		var diffContent string
		if msg.diffData != nil && !msg.isError {
			diffContent = diff.Render(msg.diffData.OldContent, msg.diffData.NewContent, msg.diffData.FilePath, msg.diffData.IsNewFile)
		}
		m.messages = append(m.messages, ChatMessage{
			Role:    "tool",
			Content: fmt.Sprintf("  %s %s: %s", prefix, msg.name, truncate(msg.content, 200)),
			Diff:    diffContent,
		})
		return m, nil

	case errorMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    "error",
			Content: msg.message,
		})
		m.currentStreaming = nil
		if msg.fatal {
			m.state = StateChatting
		}
		return m, nil

	case doneMsg:
		m.state = StateChatting
		m.currentStreaming = nil
		return m, nil

	case tokenUsageMsg:
		m.tokenUsage = &msg.data
		return m, nil

	case permissionRequestMsg:
		m.permDialog.Show(msg.data)
		return m, nil
	}

	return m, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.permDialog.Visible() {
		handled := m.permDialog.HandleKey(msg)
		if handled {
			return m, nil
		}
	}

	switch msg.String() {
	case "ctrl+c":
		if m.state == StateStreaming {
			m.cancel()
			m.state = StateChatting
			return m, nil
		}
		if m.state == StateChatting && !m.inputBuf.IsEmpty() {
			m.inputBuf.Clear()
			m.historyIndex = -1
			m.historyDraft = ""
			return m, nil
		}
		m.state = StateExiting
		if m.unsub != nil {
			m.unsub()
		}
		m.bus.Close()
		return m, tea.Quit

	case "ctrl+enter", "alt+enter", "ctrl+s":
		if m.state == StateChatting && !m.inputBuf.IsEmpty() {
			return m.handleSubmit()
		}

	case "enter":
		if m.state == StateChatting {
			m.inputBuf.InsertNewline()
			m.historyIndex = -1
		}

	case "backspace":
		if m.state == StateChatting {
			m.inputBuf.Backspace()
			m.historyIndex = -1
		}

	case "delete":
		if m.state == StateChatting {
			m.inputBuf.Delete()
			m.historyIndex = -1
		}

	case "up":
		if m.state == StateChatting {
			if m.inputBuf.cursorLine == 0 && len(m.inputHistory) > 0 {
				m.navigateHistoryUp()
			} else {
				m.inputBuf.MoveUp()
			}
		}

	case "down":
		if m.state == StateChatting {
			if m.inputBuf.cursorLine == m.inputBuf.LineCount()-1 && m.historyIndex != -1 {
				m.navigateHistoryDown()
			} else {
				m.inputBuf.MoveDown()
			}
		}

	case "left":
		if m.state == StateChatting {
			m.inputBuf.MoveLeft()
		}

	case "right":
		if m.state == StateChatting {
			m.inputBuf.MoveRight()
		}

	case "home":
		if m.state == StateChatting {
			m.inputBuf.MoveHome()
		}

	case "end":
		if m.state == StateChatting {
			m.inputBuf.MoveEnd()
		}

	case "space":
		if m.state == StateChatting {
			m.inputBuf.InsertChar(" ")
			m.historyIndex = -1
		}

	case "ctrl+a":
		if m.state == StateChatting {
			m.inputBuf.cursorLine = 0
			m.inputBuf.cursorCol = 0
		}

	default:
		if m.state == StateChatting && msg.String() != "" && len(msg.String()) == 1 {
			m.inputBuf.InsertChar(msg.String())
			m.historyIndex = -1
		}
	}

	return m, nil
}

func (m *Model) navigateHistoryUp() {
	if len(m.inputHistory) == 0 {
		return
	}
	if m.historyIndex == -1 {
		m.historyDraft = m.inputBuf.Text()
		m.historyIndex = len(m.inputHistory) - 1
	} else if m.historyIndex > 0 {
		m.historyIndex--
	}
	m.inputBuf.SetText(m.inputHistory[m.historyIndex])
}

func (m *Model) navigateHistoryDown() {
	if m.historyIndex == -1 {
		return
	}
	if m.historyIndex < len(m.inputHistory)-1 {
		m.historyIndex++
		m.inputBuf.SetText(m.inputHistory[m.historyIndex])
	} else {
		m.historyIndex = -1
		m.inputBuf.SetText(m.historyDraft)
	}
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := m.inputBuf.Text()
	m.addToHistory(prompt)
	m.inputBuf.Clear()
	m.historyIndex = -1
	m.historyDraft = ""
	m.messages = append(m.messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})
	m.state = StateStreaming
	m.currentStreaming = nil

	go func() {
		systemPrompt := buildSystemPrompt()
		if m.skillRegistry != nil {
			skillCtx := skill.SkillContext{
				UserMessage: prompt,
			}
			skillPrompt := m.skillRegistry.BuildSystemPrompt(m.ctx, skillCtx)
			if skillPrompt != "" {
				systemPrompt = systemPrompt + skillPrompt
			}
		}
		_ = m.agent.Run(m.ctx, m.sessionID, prompt, agent.Config{
			Model:        m.cfg.DefaultModel,
			MaxTokens:    4096,
			SystemPrompt: systemPrompt,
			ToolsEnabled: m.toolsEnabled,
		})
	}()

	return m, nil
}

func (m *Model) addToHistory(text string) {
	if text == "" {
		return
	}
	if len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == text {
		return
	}
	m.inputHistory = append(m.inputHistory, text)
	if len(m.inputHistory) > 100 {
		m.inputHistory = m.inputHistory[len(m.inputHistory)-100:]
	}
}

func (m *Model) inputAreaReserved() int {
	reserved := 4
	if m.state == StateStreaming {
		reserved++
	}
	if m.permDialog.Visible() {
		reserved += 8
	}
	return reserved
}

func (m *Model) View() tea.View {
	var b strings.Builder

	b.WriteString(TitleStyle.Render("⚡ Techne Code"))
	b.WriteString("\n\n")

	visibleHeight := m.height - 6
	if visibleHeight < 5 {
		visibleHeight = 5
	}

	start := 0
	if len(m.messages) > visibleHeight {
		start = len(m.messages) - visibleHeight
	}

	for _, msg := range m.messages[start:] {
		switch msg.Role {
		case "user":
			b.WriteString(UserMessageStyle.Render("You: " + msg.Content))
		case "assistant":
			if msg.Thinking != "" {
				b.WriteString(markdown.RenderThinking(msg.Thinking))
				b.WriteString("\n")
			}
			rendered := markdown.Render(msg.Content)
			b.WriteString(AssistantMessageStyle.Render("Assistant:\n" + rendered))
		case "tool":
			b.WriteString(ToolMessageStyle.Render(msg.Content))
			if msg.Diff != "" {
				b.WriteString("\n")
				b.WriteString(msg.Diff)
			}
		case "error":
			b.WriteString(ErrorMessageStyle.Render("Error: " + msg.Content))
		}
		b.WriteString("\n")
	}

	if m.state == StateStreaming {
		if m.thinkingBuffer != "" {
			b.WriteString(markdown.RenderThinking(m.thinkingBuffer))
			b.WriteString("\n")
		} else {
			b.WriteString(DimStyle.Render("● Thinking..."))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.state == StateChatting {
		inputHeight := m.height - m.inputAreaReserved()
		if inputHeight < 3 {
			inputHeight = 3
		}
		visibleLines := m.inputBuf.VisibleLines(inputHeight)
		scrollOffset := m.inputBuf.ScrollOffset()

		for i, line := range visibleLines {
			lineNum := scrollOffset + i
			cursorLine, cursorCol := m.inputBuf.CursorPos()
			if lineNum == cursorLine {
				displayLine := line[:min(cursorCol, len(line))] + "█" + line[min(cursorCol, len(line)):]
				b.WriteString(InputPromptStyle.Render("  " + displayLine))
			} else {
				b.WriteString(InputPromptStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	} else {
		b.WriteString(DimStyle.Render("> ..."))
		b.WriteString("\n")
	}

	if m.permDialog.Visible() {
		b.WriteString("\n")
		b.WriteString(m.permDialog.View(m.width))
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return tea.NewView(b.String())
}

func (m *Model) renderStatusBar() string {
	modelName := fmt.Sprintf("%s/%s", m.cfg.DefaultProvider, m.cfg.DefaultModel)

	sessionShort := "------"
	if len(m.sessionID) >= 8 {
		sessionShort = m.sessionID[:8]
	} else if m.sessionID != "" {
		sessionShort = m.sessionID
	}

	tokenInfo := "tokens: --"
	if m.tokenUsage != nil {
		tokenInfo = fmt.Sprintf("tokens: %d", m.tokenUsage.TotalTokens)
	}

	contextInfo := "ctx: --"
	if m.tokenUsage != nil && m.tokenUsage.ContextWindow > 0 {
		pct := float64(m.tokenUsage.EstimatedContextUsage) / float64(m.tokenUsage.ContextWindow) * 100
		contextInfo = fmt.Sprintf("ctx: %.0f%%", pct)
	}

	separator := lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C")).Render(" │ ")

	stateIndicator := ""
	if m.state == StateStreaming {
		stateIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("● ")
	}

	helpText := "enter:newline ctrl+enter:send ctrl+c:quit"
	if m.state == StateStreaming {
		helpText = "ctrl+c:cancel"
	}
	if m.permDialog.Visible() {
		helpText = "y:allow a:always n:deny tab:cycle"
	}

	statusPart := stateIndicator + modelName + separator + sessionShort + separator + tokenInfo + separator + contextInfo
	helpPart := lipgloss.NewStyle().Foreground(lipgloss.Color("#5B6071")).Render(helpText)

	separatorMid := lipgloss.NewStyle().Foreground(lipgloss.Color("#3C3C3C")).Render("  ")

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render(statusPart + separatorMid + helpPart)
}

type messageDeltaMsg struct{ text string }
type thinkingDeltaMsg struct{ text string }
type toolStartMsg struct{ name string }
type toolResultMsg struct {
	name     string
	content  string
	isError  bool
	diffData *event.DiffData
}
type errorMsg struct {
	message string
	fatal   bool
}
type doneMsg struct{}

type tokenUsageMsg struct {
	data event.TokenUsageData
}

type permissionRequestMsg struct {
	data event.PermissionRequestData
}

func buildSystemPrompt() string {
	return `You are Techne Code, an expert AI coding assistant. You help developers write, review, debug, and understand code.

You have access to tools for reading, writing, and editing files, executing shell commands, and searching codebases.

Guidelines:
- Be concise and direct
- Explain your reasoning before making changes
- Use tools to verify your work (read files after writing, run tests)
- Always read a file before editing it
- Prefer small, focused changes over large rewrites`
}

func (m *Model) Client() *llm.Client {
	return m.client
}
