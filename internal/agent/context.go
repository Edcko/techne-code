package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Edcko/techne-code/internal/llm"
	"github.com/Edcko/techne-code/pkg/event"
	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/session"
)

const (
	defaultThreshold       = 0.9
	keepRecentMessages     = 6
	minMessagesToSummarize = 8
)

type TokenUsageData struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	CachedTokens int `json:"cached_tokens,omitempty"`
}

type ContextManager struct {
	store      session.SessionStore
	bus        event.EventBus
	summarizer *Summarizer
	client     *llm.Client

	mu    sync.Mutex
	usage map[string]*TokenUsageData
}

func NewContextManager(store session.SessionStore, bus event.EventBus, client *llm.Client) *ContextManager {
	return &ContextManager{
		store:      store,
		bus:        bus,
		summarizer: NewSummarizer(client, bus),
		client:     client,
		usage:      make(map[string]*TokenUsageData),
	}
}

func (cm *ContextManager) GetTokenUsage(sessionID string) TokenUsageData {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if u, ok := cm.usage[sessionID]; ok {
		return *u
	}
	return TokenUsageData{}
}

func (cm *ContextManager) TrackUsage(sessionID string, usage provider.Usage) {
	cm.mu.Lock()

	if cm.usage[sessionID] == nil {
		cm.usage[sessionID] = &TokenUsageData{}
	}

	u := cm.usage[sessionID]
	u.InputTokens += usage.InputTokens
	u.OutputTokens += usage.OutputTokens
	u.TotalTokens = u.InputTokens + u.OutputTokens
	u.CachedTokens += usage.CacheReadTokens

	data := *u
	cm.mu.Unlock()

	cm.bus.Publish(event.NewEvent(event.EventSessionUpdate, sessionID, data))
}

func (cm *ContextManager) CheckAndCompress(ctx context.Context, sessionID string, model string, messages []provider.Message, systemPrompt string) ([]provider.Message, error) {
	contextWindow := GetContextWindow(cm.client.Provider().Models(), model)

	systemTokens := EstimateSystemPromptTokens(systemPrompt)
	messageTokens := EstimateMessagesTokens(messages)
	totalTokens := systemTokens + messageTokens

	if !IsApproachingLimit(totalTokens, contextWindow, defaultThreshold) {
		return messages, nil
	}

	if len(messages) <= minMessagesToSummarize {
		return messages, nil
	}

	splitIdx := len(messages) - keepRecentMessages
	if splitIdx <= 0 {
		return messages, nil
	}

	oldMessages := messages[:splitIdx]
	recentMessages := messages[splitIdx:]

	summary, err := cm.summarizer.Summarize(ctx, sessionID, model, oldMessages)
	if err != nil {
		log.Printf("context compression failed: %v", err)
		return messages, nil
	}

	summaryMsg := provider.Message{
		Role: provider.RoleUser,
		Content: []provider.ContentBlock{
			{
				Type: provider.BlockText,
				Text: fmt.Sprintf("[Conversation Summary]\n%s\n[End of Summary — recent messages follow]", summary),
			},
		},
	}

	compressed := make([]provider.Message, 0, 1+len(recentMessages))
	compressed = append(compressed, summaryMsg)
	compressed = append(compressed, recentMessages...)

	cm.persistSummary(sessionID, summary)

	return compressed, nil
}

func (cm *ContextManager) persistSummary(sessionID string, summary string) {
	summaryMsg := &session.StoredMessage{
		SessionID: sessionID,
		Role:      string(provider.RoleSystem),
		Content:   toJSON([]provider.ContentBlock{{Type: provider.BlockText, Text: summary}}),
	}

	if err := cm.store.SaveMessage(summaryMsg); err != nil {
		log.Printf("failed to save summary message: %v", err)
		return
	}

	var content []provider.ContentBlock
	if err := json.Unmarshal(summaryMsg.Content, &content); err == nil {
		_ = content
	}

	if err := cm.store.UpdateSessionSummary(sessionID, summaryMsg.ID); err != nil {
		log.Printf("failed to update session summary: %v", err)
	}
}

func (cm *ContextManager) EstimateCurrentUsage(messages []provider.Message, systemPrompt string) int {
	systemTokens := EstimateSystemPromptTokens(systemPrompt)
	messageTokens := EstimateMessagesTokens(messages)
	return systemTokens + messageTokens
}
