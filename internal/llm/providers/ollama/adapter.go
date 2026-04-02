package ollama

import (
	"github.com/Edcko/techne-code/internal/llm/providers/openai"
	"github.com/Edcko/techne-code/pkg/provider"
)

const defaultBaseURL = "http://localhost:11434/v1"

var ollamaModels = []provider.ModelInfo{
	{ID: "llama3", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 8192},
	{ID: "llama3.1", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 128000},
	{ID: "llama3.2", MaxTokens: 4096, SupportsTools: true, SupportsVision: true, ContextWindow: 128000},
	{ID: "codellama", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 16384},
	{ID: "deepseek-coder", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 16384},
	{ID: "qwen2.5-coder", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 32768},
	{ID: "mistral", MaxTokens: 4096, SupportsTools: true, SupportsVision: false, ContextWindow: 32768},
}

type Adapter struct {
	*openai.Adapter
}

func New(baseURL string) *Adapter {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Adapter{
		Adapter: openai.NewAdapter("ollama", baseURL, ollamaModels),
	}
}

func (a *Adapter) Name() string {
	return "ollama"
}
