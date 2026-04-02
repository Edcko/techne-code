package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/Edcko/techne-code/internal/agent"
	"github.com/Edcko/techne-code/internal/config"
	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/tool"
)

// State represents the current TUI state
type State int

const (
	StateInit State = iota
	StateChatting
	StateStreaming
	StateExiting
)

// ChatMessage represents a message in the chat view
type ChatMessage struct {
	Role    string // "user", "assistant", "tool", "error"
	Content string
}

// Model is the root Bubbletea model for Techne Code
type Model struct {
	state  State
	cfg    *config.Config
	agent  *agent.Agent
	client *llm.Client
	store  session.SessionStore
	bus    event.EventBus
	unsub  func() // event bus unsubscribe

	// UI state
	messages   []ChatMessage
	input      string
	statusText string
	sessionID  string
	err        error

	// Dimensions
	width  int
	height int

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewModel creates a new TUI model
func NewModel(
	cfg *config.Config,
	agentClient *llm.Client,
	store session.SessionStore,
	registry tool.ToolRegistry,
	perm *permission.Service,
	bus event.EventBus,
) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{
		state:    StateInit,
		cfg:      cfg,
		client:   agentClient,
		store:    store,
		bus:      bus,
		agent:    agent.New(agentClient, store, registry, perm, bus),
		ctx:      ctx,
		cancel:   cancel,
		messages: []ChatMessage{},
		input:    "",
	}
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	// Subscribe to events
	m.unsub = m.bus.Subscribe(func(e event.Event) {
		// Events are handled via poll in Update
	})

	// Create new session
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

	m.state = StateChatting
	m.statusText = fmt.Sprintf("%s/%s", m.cfg.DefaultProvider, m.cfg.DefaultModel)
	return nil
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	// Custom messages from event bus
	case streamUpdateMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    "assistant",
			Content: msg.text,
		})
		m.state = StateChatting
		return m, nil

	case streamErrMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    "error",
			Content: msg.text,
		})
		m.state = StateChatting
		return m, nil

	case toolUpdateMsg:
		m.messages = append(m.messages, ChatMessage{
			Role:    "tool",
			Content: fmt.Sprintf("▶ %s", msg.text),
		})
		return m, nil
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if m.state == StateStreaming {
			m.cancel()
			m.state = StateChatting
			return m, nil
		}
		m.state = StateExiting
		if m.unsub != nil {
			m.unsub()
		}
		m.bus.Close()
		return m, tea.Quit

	case "enter":
		if m.state == StateChatting && strings.TrimSpace(m.input) != "" {
			return m.handleSubmit()
		}

	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}

	default:
		if m.state == StateChatting && msg.String() != "" && len(msg.String()) == 1 {
			m.input += msg.String()
		}
	}

	return m, nil
}

func (m *Model) handleSubmit() (tea.Model, tea.Cmd) {
	prompt := m.input
	m.input = ""
	m.messages = append(m.messages, ChatMessage{
		Role:    "user",
		Content: prompt,
	})
	m.state = StateStreaming

	// Run agent in background
	go func() {
		err := m.agent.Run(m.ctx, m.sessionID, prompt, agent.Config{
			Model:        m.cfg.DefaultModel,
			MaxTokens:    4096,
			SystemPrompt: buildSystemPrompt(),
		})
		_ = err // TODO: handle error via event bus
	}()

	return m, nil
}

// View renders the TUI
func (m *Model) View() tea.View {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render("⚡ Techne Code"))
	b.WriteString("\n\n")

	// Chat messages
	visibleHeight := m.height - 6 // header + input + status
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
			b.WriteString(AssistantMessageStyle.Render("Assistant: " + msg.Content))
		case "tool":
			b.WriteString(ToolMessageStyle.Render(msg.Content))
		case "error":
			b.WriteString(ErrorMessageStyle.Render("Error: " + msg.Content))
		}
		b.WriteString("\n")
	}

	// Streaming indicator
	if m.state == StateStreaming {
		b.WriteString(DimStyle.Render("● Thinking..."))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Input
	prompt := "> "
	if m.state == StateChatting {
		b.WriteString(InputPromptStyle.Render(prompt + m.input + "█"))
	} else {
		b.WriteString(DimStyle.Render(prompt + "..."))
	}
	b.WriteString("\n")

	// Status bar
	help := "ctrl+c: quit | enter: send"
	if m.state == StateStreaming {
		help = "ctrl+c: cancel"
	}
	b.WriteString(StatusBarStyle.Render(fmt.Sprintf(" %s | %s ", m.statusText, help)))

	return tea.NewView(b.String())
}

// custom message types for event bus → Bubbletea bridge
type streamUpdateMsg struct{ text string }
type streamErrMsg struct{ text string }
type toolUpdateMsg struct{ text string }

// buildSystemPrompt creates the system prompt for the agent
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

// Client returns the LLM client (for external access)
func (m *Model) Client() *llm.Client {
	return m.client
}
