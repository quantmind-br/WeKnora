package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/models/provider"
	"github.com/Tencent/WeKnora/internal/models/utils/ollama"
	"github.com/Tencent/WeKnora/internal/types"
)

// Tool represents a function/tool definition
type Tool struct {
	Type     string      `json:"type"` // "function"
	Function FunctionDef `json:"function"`
}

// FunctionDef represents a function definition
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ChatOptions chat options
type ChatOptions struct {
	Temperature         float64         `json:"temperature"`                   // Temperature parameter
	TopP                float64         `json:"top_p"`                         // Top P parameter
	Seed                int             `json:"seed"`                          // Random seed
	MaxTokens           int             `json:"max_tokens"`                    // Maximum token count
	MaxCompletionTokens int             `json:"max_completion_tokens"`         // Maximum completion token count
	FrequencyPenalty    float64         `json:"frequency_penalty"`             // Frequency penalty
	PresencePenalty     float64         `json:"presence_penalty"`              // Presence penalty
	Thinking            *bool           `json:"thinking"`                      // Whether thinking is enabled
	Tools               []Tool          `json:"tools,omitempty"`               // List of available tools
	ToolChoice          string          `json:"tool_choice,omitempty"`         // "auto", "required", "none", or specific tool
	ParallelToolCalls   *bool           `json:"parallel_tool_calls,omitempty"` // Whether to allow parallel tool calls (default nil means the model decides)
	Format              json.RawMessage `json:"format,omitempty"`              // Response format definition
}

// MessageContentPart represents a part of multi-content message
type MessageContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // For type="text"
	ImageURL *ImageURL `json:"image_url,omitempty"` // For type="image_url"
}

// ImageURL represents the image URL structure
type ImageURL struct {
	URL    string `json:"url"`              // URL or base64 data URI
	Detail string `json:"detail,omitempty"` // "auto", "low", "high"
}

// Message represents a chat message
type Message struct {
	Role         string               `json:"role"`                    // Role: system, user, assistant, tool
	Content      string               `json:"content"`                 // Message content
	MultiContent []MessageContentPart `json:"multi_content,omitempty"` // Multi-content message (text + image)
	Name         string               `json:"name,omitempty"`          // Function/tool name (for tool role)
	ToolCallID   string               `json:"tool_call_id,omitempty"`  // Tool call ID (for tool role)
	ToolCalls    []ToolCall           `json:"tool_calls,omitempty"`    // Tool calls (for assistant role)
	Images       []string             `json:"images,omitempty"`        // Image URLs for multimodal (only for current user message)
	// ReasoningContent is used by assistant reasoning models (DeepSeek thinking, Xiaomi MiMo, vLLM reasoning, etc.)
	// Thinking content output in the previous round. Some providers (MiMo, DeepSeek V3.2/V4 thinking mode) require that in multi-turn conversations
	// the assistant's reasoning_content be passed back verbatim, otherwise the request is rejected with 400; other providers that don't require this
	// will ignore unknown fields, with no side effects.
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"` // "function"
	Function         FunctionCall           `json:"function"`
	ProviderMetadata types.ToolCallMetadata `json:"provider_metadata,omitempty"`
}

// FunctionCall represents a function call
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// Chat defines the chat interface
type Chat interface {
	// Chat performs non-streaming chat
	Chat(ctx context.Context, messages []Message, opts *ChatOptions) (*types.ChatResponse, error)

	// ChatStream performs streaming chat
	ChatStream(ctx context.Context, messages []Message, opts *ChatOptions) (<-chan types.StreamResponse, error)

	// GetModelName gets the model name
	GetModelName() string

	// GetModelID gets the model ID
	GetModelID() string
}

type ChatConfig struct {
	Source    types.ModelSource
	BaseURL   string
	ModelName string
	APIKey    string
	ModelID   string
	Provider  string
	// MaxConcurrency caps concurrent background calls to this model; 0 falls
	// back to the process-wide default (see limiter.GateN).
	MaxConcurrency int
	ExtraConfig    map[string]string
	// CustomHeaders allows attaching custom HTTP headers when calling a remote OpenAI-compatible API (like OpenAI Python SDK's extra_headers).
	CustomHeaders map[string]string
	AppID         string
	AppSecret     string // Encrypted value, passed in by the factory function caller, already decrypted before use in NewWeKnoraCloudChat
}

// ConfigFromModel builds a ChatConfig from types.Model.
// Ensures the production path (the service layer spins up an instance based on the model config in the DB) and the test path
// (the handler layer temporarily spins up an instance based on the frontend form) go through exactly the same field mapping, avoiding duplicated boilerplate.
// appID / appSecret are already decrypted/parsed WeKnoraCloud credentials; the caller is responsible for passing them in.
func ConfigFromModel(m *types.Model, appID, appSecret string) *ChatConfig {
	if m == nil {
		return nil
	}
	return &ChatConfig{
		ModelID:        m.ID,
		APIKey:         m.Parameters.APIKey,
		BaseURL:        m.Parameters.BaseURL,
		ModelName:      m.Name,
		Source:         m.Source,
		Provider:       m.Parameters.Provider,
		MaxConcurrency: m.Parameters.MaxConcurrency,
		ExtraConfig:    m.Parameters.ExtraConfig,
		CustomHeaders:  m.Parameters.CustomHeaders,
		AppID:          appID,
		AppSecret:      appSecret,
	}
}

// NewChat creates a chat instance
func NewChat(config *ChatConfig, ollamaService *ollama.OllamaService) (Chat, error) {
	var c Chat
	var err error
	switch strings.ToLower(string(config.Source)) {
	case string(types.ModelSourceLocal):
		c, err = NewOllamaChat(config, ollamaService)
	case string(types.ModelSourceRemote):
		c, err = NewRemoteChat(config)
	default:
		return nil, fmt.Errorf("unsupported chat model source: %s", config.Source)
	}
	c, err = wrapChatDebug(c, err)
	c, err = wrapChatLangfuse(c, err)
	// Outermost: hold the per-model concurrency slot only around the real
	// provider round-trip, so the wait is excluded from debug/langfuse timing.
	return wrapChatConcurrency(c, config.MaxConcurrency, err)
}

// NewRemoteChat creates a remote chat instance based on the provider.
// Anthropic uses a separate Messages protocol implementation; other OpenAI-compatible providers are handled uniformly by
// RemoteAPIChat, with provider-specific behavior resolved at construction time via providerAdapter.
func NewRemoteChat(config *ChatConfig) (Chat, error) {
	providerName := provider.ProviderName(config.Provider)
	if providerName == "" {
		providerName = provider.DetectProvider(config.BaseURL)
	}
	if providerName == provider.ProviderAnthropic {
		return NewAnthropicChat(config)
	}
	return NewRemoteAPIChat(config)
}
