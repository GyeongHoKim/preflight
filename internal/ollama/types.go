package ollama

import (
	"encoding/json"
)

// ChatRequest is the JSON body for POST /api/chat.
type ChatRequest struct {
	Model    string          `json:"model"`
	Messages []ChatMessage   `json:"messages"`
	Stream   bool            `json:"stream"`
	Tools    []ToolDef       `json:"tools,omitempty"`
	Format   json.RawMessage `json:"format,omitempty"`
}

// ChatMessage is one entry in the chat history.
type ChatMessage struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Name      string     `json:"name,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// ToolDef is a function tool exposed to the model.
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction describes the callable function metadata and JSON Schema parameters.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall is a tool invocation requested by the assistant.
type ToolCall struct {
	ID       string          `json:"id,omitempty"`
	Function ToolCallFunc    `json:"function"`
	RawIndex json.RawMessage `json:"index,omitempty"`
}

// ToolCallFunc holds the function name and arguments from the model.
type ToolCallFunc struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ChatResponse is a non-streaming /api/chat response.
type ChatResponse struct {
	Model   string      `json:"model"`
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}
