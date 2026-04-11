package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/tool"
	"github.com/google/uuid"
)

type SubAgentConfig struct {
	Name          string
	Description   string
	SystemPrompt  string
	AllowedTools  []string
	MaxIterations int
	Model         string
	MaxTokens     int
}

type SubAgent struct {
	provider provider.Provider
	store    session.SessionStore
	config   SubAgentConfig
	toolMap  map[string]tool.Tool
}

func NewSubAgent(
	prov provider.Provider,
	store session.SessionStore,
	config SubAgentConfig,
	allTools []tool.Tool,
) *SubAgent {
	toolMap := make(map[string]tool.Tool)
	allowedSet := make(map[string]bool)
	for _, name := range config.AllowedTools {
		allowedSet[name] = true
	}
	for _, t := range allTools {
		if allowedSet[t.Name()] {
			toolMap[t.Name()] = t
		}
	}

	if config.MaxIterations <= 0 {
		config.MaxIterations = 20
	}
	if config.MaxTokens <= 0 {
		config.MaxTokens = 4096
	}

	return &SubAgent{
		provider: prov,
		store:    store,
		config:   config,
		toolMap:  toolMap,
	}
}

func (sa *SubAgent) Run(ctx context.Context, task string) (string, error) {
	if sa.provider == nil {
		return "", fmt.Errorf("sub-agent %q has no provider configured", sa.config.Name)
	}

	childSession := &session.Session{
		ID:       uuid.New().String(),
		Title:    fmt.Sprintf("[sub-agent:%s] %s", sa.config.Name, truncateString(task, 50)),
		Model:    sa.config.Model,
		Provider: sa.provider.Name(),
	}
	if err := sa.store.CreateSession(childSession); err != nil {
		return "", fmt.Errorf("create child session: %w", err)
	}

	scoped := newScopedRegistry(sa.toolMap)
	silentBus := &SilentEventBus{}
	client := llm.NewClient(sa.provider, silentBus)
	subPerm := permission.NewService(permission.ModeAutoAllow, sa.config.AllowedTools)

	ag := New(client, sa.store, scoped, subPerm, silentBus)

	agentConfig := Config{
		Model:         sa.config.Model,
		MaxTokens:     sa.config.MaxTokens,
		SystemPrompt:  sa.config.SystemPrompt,
		MaxIterations: sa.config.MaxIterations,
		ToolsEnabled:  true,
	}

	if err := ag.Run(ctx, childSession.ID, task, agentConfig); err != nil {
		return "", fmt.Errorf("sub-agent %q failed: %w", sa.config.Name, err)
	}

	messages, err := sa.store.GetMessages(childSession.ID)
	if err != nil {
		return "", fmt.Errorf("load child messages: %w", err)
	}

	var textParts []string
	for _, msg := range messages {
		if msg.Role != string(provider.RoleAssistant) {
			continue
		}
		var blocks []provider.ContentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == provider.BlockText && b.Text != "" {
				textParts = append(textParts, b.Text)
			}
		}
	}

	if len(textParts) == 0 {
		return "(sub-agent completed with no text output)", nil
	}

	return strings.Join(textParts, "\n"), nil
}

func (sa *SubAgent) ToolCount() int {
	return len(sa.toolMap)
}

func (sa *SubAgent) HasTool(name string) bool {
	_, ok := sa.toolMap[name]
	return ok
}

type scopedRegistry struct {
	tools map[string]tool.Tool
}

func newScopedRegistry(toolMap map[string]tool.Tool) *scopedRegistry {
	return &scopedRegistry{tools: toolMap}
}

func (r *scopedRegistry) Register(t tool.Tool) error {
	r.tools[t.Name()] = t
	return nil
}

func (r *scopedRegistry) Get(name string) (tool.Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *scopedRegistry) List() []tool.Tool {
	result := make([]tool.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

func (r *scopedRegistry) Schemas() []provider.ToolDef {
	result := make([]provider.ToolDef, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, provider.ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return result
}

type SilentEventBus struct{}

func (s *SilentEventBus) Publish(event.Event)                 {}
func (s *SilentEventBus) Subscribe(event.EventHandler) func() { return func() {} }
func (s *SilentEventBus) Close()                              {}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
