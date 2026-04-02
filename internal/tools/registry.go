package tools

import (
	"fmt"
	"sync"

	"github.com/Edcko/techne-code/pkg/provider"
	"github.com/Edcko/techne-code/pkg/tool"
)

// Registry implements tool.ToolRegistry with thread-safe tool management.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]tool.Tool
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tool.Tool),
	}
}

// Register adds a tool to the registry.
// Returns an error if a tool with the same name already exists.
func (r *Registry) Register(t tool.Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q already registered", name)
	}
	r.tools[name] = t
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (tool.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *Registry) List() []tool.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Schemas returns tool definitions for LLM function calling.
func (r *Registry) Schemas() []provider.ToolDef {
	r.mu.RLock()
	defer r.mu.RUnlock()

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
