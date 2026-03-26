package tools

import (
	"context"
	"customclaw/internal/llm"
)

// Tool is the interface every tool must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() llm.ParameterSchema
	Execute(ctx context.Context, input map[string]any) (string, error)
}
