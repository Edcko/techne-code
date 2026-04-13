package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/internal/permission"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
	"github.com/Edcko/techne-code/pkg/skill"
	"github.com/Edcko/techne-code/pkg/tool"
)

// Agent is the core AI coding agent that runs the conversation loop.
type Agent struct {
	client         *llm.Client
	store          session.SessionStore
	registry       tool.ToolRegistry
	permission     *permission.Service
	bus            event.EventBus
	skillRegistry  skill.SkillRegistry
	contextManager *ContextManager
}

// Config holds agent configuration.
type Config struct {
	Model         string
	MaxTokens     int
	SystemPrompt  string
	MaxIterations int
	ToolsEnabled  bool
}

// New creates a new Agent.
func New(
	client *llm.Client,
	store session.SessionStore,
	registry tool.ToolRegistry,
	perm *permission.Service,
	bus event.EventBus,
) *Agent {
	ag := &Agent{
		client:         client,
		store:          store,
		registry:       registry,
		permission:     perm,
		bus:            bus,
		contextManager: NewContextManager(store, bus, client),
	}
	return ag
}

func (a *Agent) WithSkills(registry skill.SkillRegistry) {
	a.skillRegistry = registry
}

// Run executes the agent loop for a user prompt.
// It creates/reuses a session, runs the LLM, executes tools, and repeats
// until the LLM stops requesting tool calls.
func (a *Agent) Run(ctx context.Context, sessionID string, userPrompt string, config Config) error {
	if config.MaxIterations <= 0 {
		config.MaxIterations = 50
	}

	// Save user message
	userMsg := &session.StoredMessage{
		SessionID: sessionID,
		Role:      string(provider.RoleUser),
		Content:   toJSON([]provider.ContentBlock{{Type: provider.BlockText, Text: userPrompt}}),
	}
	if err := a.store.SaveMessage(userMsg); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// Track tool call hashes for loop detection
	var recentHashes []string

	// Agent loop: send to LLM, execute tools, repeat
	for i := 0; i < config.MaxIterations; i++ {
		// Load conversation history
		messages, err := a.loadMessages(sessionID)
		if err != nil {
			return fmt.Errorf("load messages: %w", err)
		}

		systemPrompt := config.SystemPrompt
		if a.skillRegistry != nil {
			skillCtx := skill.SkillContext{
				UserMessage: userPrompt,
			}
			skillPrompt := a.skillRegistry.BuildSystemPrompt(ctx, skillCtx)
			if skillPrompt != "" {
				systemPrompt = systemPrompt + skillPrompt
			}
		}

		messages, err = a.contextManager.CheckAndCompress(ctx, sessionID, config.Model, messages, systemPrompt)
		if err != nil {
			return fmt.Errorf("context compression: %w", err)
		}

		req := provider.ChatRequest{
			Messages: messages,
			System:   systemPrompt,
			Config: provider.ProviderConfig{
				Model:     config.Model,
				MaxTokens: config.MaxTokens,
			},
		}

		if config.ToolsEnabled {
			req.Tools = a.registry.Schemas()
		}

		// Call LLM with streaming
		resp, err := a.client.Stream(ctx, sessionID, req)
		if err != nil {
			a.bus.Publish(event.NewEvent(event.EventError, sessionID, event.ErrorData{
				Message: err.Error(),
				Fatal:   true,
			}))
			return fmt.Errorf("LLM stream: %w", err)
		}

		a.contextManager.TrackUsage(sessionID, resp.Usage)

		a.publishTokenUsage(sessionID, config.Model, messages, systemPrompt)

		// Save assistant message
		assistantMsg := &session.StoredMessage{
			SessionID: sessionID,
			Role:      string(provider.RoleAssistant),
			Content:   toJSON(resp.Content),
		}
		if err := a.store.SaveMessage(assistantMsg); err != nil {
			return fmt.Errorf("save assistant message: %w", err)
		}

		// Extract tool calls from response
		var toolCalls []provider.ToolCallBlock
		for _, block := range resp.Content {
			if block.Type == provider.BlockToolCall && block.ToolCall != nil {
				toolCalls = append(toolCalls, *block.ToolCall)
			}
		}

		// No tool calls — agent is done
		if len(toolCalls) == 0 {
			a.bus.Publish(event.NewEvent(event.EventDone, sessionID, nil))
			return nil
		}

		// Execute each tool call
		var toolResults []provider.ContentBlock
		for _, tc := range toolCalls {
			// Loop detection
			hash := toolCallHash(tc.Name, tc.Input)
			recentHashes = append(recentHashes, hash)
			if detectLoop(recentHashes, 5, 10) {
				a.bus.Publish(event.NewEvent(event.EventError, sessionID, event.ErrorData{
					Message: fmt.Sprintf("Loop detected: tool %q called repeatedly with same input", tc.Name),
					Fatal:   true,
				}))
				return fmt.Errorf("loop detected on tool %q", tc.Name)
			}

			// Permission check
			if a.permission != nil && !a.permission.IsAllowed(sessionID, tc.Name, "execute") {
				t, exists := a.registry.Get(tc.Name)
				if exists && t.RequiresPermission() {
					respCh := make(chan event.PermissionResponseData, 1)
					permReq := event.PermissionRequestData{
						ToolName:    tc.Name,
						Action:      "execute",
						Description: t.Description(),
						Params:      tc.Input,
						Response:    respCh,
					}

					a.bus.Publish(event.NewEvent(event.EventPermissionReq, sessionID, permReq))

					select {
					case resp := <-respCh:
						if resp.Allowed {
							if resp.Remember {
								a.permission.Grant(sessionID, tc.Name, "execute")
							}
						} else {
							a.bus.Publish(event.NewEvent(event.EventToolResult, sessionID, event.ToolResultData{
								ToolName: tc.Name,
								Content:  "Permission denied by user",
								IsError:  true,
							}))

							toolResults = append(toolResults, provider.ContentBlock{
								Type: provider.BlockToolResult,
								ToolResult: &provider.ToolResultBlock{
									ToolCallID: tc.ID,
									Name:       tc.Name,
									Content:    "Permission denied by user",
									IsError:    true,
								},
							})
							continue
						}
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}

			// Publish tool start event
			a.bus.Publish(event.NewEvent(event.EventToolStart, sessionID, event.ToolStartData{
				ToolName: tc.Name,
				Input:    tc.Input,
			}))

			// Execute tool
			result := a.executeTool(ctx, tc)

			var diffData *event.DiffData
			if result.Diff != nil {
				diffData = &event.DiffData{
					FilePath:   result.Diff.FilePath,
					OldContent: result.Diff.OldContent,
					NewContent: result.Diff.NewContent,
					IsNewFile:  result.Diff.IsNewFile,
				}
			}

			a.bus.Publish(event.NewEvent(event.EventToolResult, sessionID, event.ToolResultData{
				ToolName: tc.Name,
				Content:  result.Content,
				IsError:  result.IsError,
				Diff:     diffData,
			}))

			// Build tool result content block
			toolResults = append(toolResults, provider.ContentBlock{
				Type: provider.BlockToolResult,
				ToolResult: &provider.ToolResultBlock{
					ToolCallID: tc.ID,
					Name:       tc.Name,
					Content:    result.Content,
					IsError:    result.IsError,
				},
			})

			// Track read files for edit protection
			if tc.Name == "read_file" {
				var params struct {
					Path string `json:"path"`
				}
				if json.Unmarshal(tc.Input, &params) == nil && params.Path != "" {
					_ = a.store.TrackReadFile(sessionID, params.Path)
				}
			}
		}

		// Save tool results as a user message
		toolMsg := &session.StoredMessage{
			SessionID: sessionID,
			Role:      string(provider.RoleTool),
			Content:   toJSON(toolResults),
		}
		if err := a.store.SaveMessage(toolMsg); err != nil {
			return fmt.Errorf("save tool results: %w", err)
		}
	}

	// Max iterations reached
	a.bus.Publish(event.NewEvent(event.EventError, sessionID, event.ErrorData{
		Message: fmt.Sprintf("Agent reached maximum iterations (%d)", config.MaxIterations),
		Fatal:   true,
	}))
	return fmt.Errorf("max iterations reached")
}

// executeTool finds and executes a tool by name.
func (a *Agent) executeTool(ctx context.Context, tc provider.ToolCallBlock) tool.ToolResult {
	t, exists := a.registry.Get(tc.Name)
	if !exists {
		return tool.ToolResult{
			Content: fmt.Sprintf("Error: unknown tool %q", tc.Name),
			IsError: true,
		}
	}

	result, err := t.Execute(ctx, tc.Input)
	if err != nil {
		return tool.ToolResult{
			Content: fmt.Sprintf("Tool error: %v", err),
			IsError: true,
		}
	}
	return result
}

// loadMessages loads and converts stored messages to provider format.
func (a *Agent) loadMessages(sessionID string) ([]provider.Message, error) {
	stored, err := a.store.GetMessages(sessionID)
	if err != nil {
		return nil, err
	}

	messages := make([]provider.Message, len(stored))
	for i, sm := range stored {
		var blocks []provider.ContentBlock
		if err := json.Unmarshal(sm.Content, &blocks); err != nil {
			// Fallback: treat as plain text
			blocks = []provider.ContentBlock{
				{Type: provider.BlockText, Text: string(sm.Content)},
			}
		}
		messages[i] = provider.Message{
			Role:    provider.Role(sm.Role),
			Content: blocks,
		}
	}
	return messages, nil
}

// toolCallHash creates a SHA-256 hash of a tool call for loop detection.
func toolCallHash(name string, input json.RawMessage) string {
	h := sha256.Sum256([]byte(name + ":" + string(input)))
	return fmt.Sprintf("%x", h[:8])
}

// detectLoop checks if the same hash appears more than maxCount times
// in the last windowSize entries.
func detectLoop(hashes []string, maxCount int, windowSize int) bool {
	if len(hashes) < maxCount+1 {
		return false
	}

	// Only look at last windowSize entries
	start := len(hashes) - windowSize
	if start < 0 {
		start = 0
	}
	window := hashes[start:]

	// Count occurrences of the last hash
	last := window[len(window)-1]
	count := 0
	for _, h := range window {
		if h == last {
			count++
		}
	}
	return count > maxCount
}

// toJSON is a helper to marshal content blocks.
func toJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("[]")
	}
	return json.RawMessage(data)
}

func (a *Agent) publishTokenUsage(sessionID string, model string, messages []provider.Message, systemPrompt string) {
	usage := a.contextManager.GetTokenUsage(sessionID)
	contextWindow := GetContextWindow(a.client.Provider().Models(), model)
	estimatedContext := a.contextManager.EstimateCurrentUsage(messages, systemPrompt)

	a.bus.Publish(event.NewEvent(event.EventTokenUsage, sessionID, event.TokenUsageData{
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		TotalTokens:           usage.TotalTokens,
		CachedTokens:          usage.CachedTokens,
		EstimatedContextUsage: estimatedContext,
		ContextWindow:         contextWindow,
	}))
}
