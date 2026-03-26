package tools

import (
	"context"
	"customclaw/internal/llm"
	"fmt"
)

// Registry holds all registered tools and dispatches execution.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools.
func (r *Registry) All() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Filter returns only the tools whose names appear in the allowed list.
// If allowed is empty, all tools are returned.
func (r *Registry) Filter(allowed []string) []Tool {
	if len(allowed) == 0 {
		return r.All()
	}
	out := make([]Tool, 0, len(allowed))
	for _, name := range allowed {
		if t, ok := r.tools[name]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Definitions converts a slice of tools to LLM tool definitions.
func Definitions(tools []Tool) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, llm.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}

// Execute runs a tool by name with the given input.
func (r *Registry) Execute(ctx context.Context, name string, input map[string]any) (string, error) {
	t, ok := r.tools[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, input)
}
