package llm

import "context"

// Message represents a single turn in the conversation.
type Message struct {
	Role       string     // "user", "assistant", "tool"
	Content    string
	ToolCalls  []ToolCall // set when role is "assistant" and LLM wants to call tools
	ToolCallID string     // set when role is "tool", matches the originating ToolCall.ID
}

// ToolCall is a request from the LLM to invoke a tool.
type ToolCall struct {
	ID    string
	Name  string
	Input map[string]any
}

// ToolDefinition describes a tool the LLM can call.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  ParameterSchema
}

// ParameterSchema is a JSON Schema object describing tool inputs.
type ParameterSchema struct {
	Type       string                     `json:"type"`
	Properties map[string]PropertySchema  `json:"properties"`
	Required   []string                   `json:"required,omitempty"`
}

// PropertySchema describes a single tool input field.
type PropertySchema struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Response is what the LLM returns for a single turn.
type Response struct {
	Content   string     // text response from the LLM
	ToolCalls []ToolCall // non-empty when the LLM wants to call tools
}

// Provider is the interface that all LLM backends must implement.
type Provider interface {
	// Chat sends a conversation to the LLM and returns its response.
	// tools is the list of tools the LLM is allowed to call.
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (*Response, error)
	Name() string
}
