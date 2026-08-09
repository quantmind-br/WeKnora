package provider

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	OpenAIBaseURL = "https://api.openai.com/v1"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct{}

func init() {
	Register(&OpenAIProvider{})
}

// Info returns metadata for the OpenAI provider
func (p *OpenAIProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderOpenAI,
		DisplayName: "OpenAI",
		Description: "gpt-5.2, gpt-5-mini, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: OpenAIBaseURL,
			types.ModelTypeEmbedding:   OpenAIBaseURL,
			types.ModelTypeRerank:      OpenAIBaseURL,
			types.ModelTypeVLLM:        OpenAIBaseURL,
			types.ModelTypeASR:         OpenAIBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
			types.ModelTypeASR,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates OpenAI provider configuration
func (p *OpenAIProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for OpenAI provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

// IsOpenAIReasoningOrGPT5Model checks whether the model is an OpenAI / Azure OpenAI
// reasoning (o-series) or GPT-5 series model.
//
// For these models, in the OpenAI Chat Completions API:
// - `max_tokens` is no longer supported; `max_completion_tokens` must be used instead;
// - only the default `temperature=1`, `top_p=1` are supported, and sampling parameters such as `frequency_penalty` /
// `presence_penalty` are not supported (passing non-default values will be rejected).
//
// Reference:
//   - https://platform.openai.com/docs/api-reference/chat
//   - https://learn.microsoft.com/azure/ai-services/openai/how-to/reasoning
//
// Matches heuristically on model name only; for Azure OpenAI, since the model name is actually the deployment name,
// if the user uses a custom deployment name we won't be able to recognize it, and it will still be treated as a regular model (keeping the original behavior).
func IsOpenAIReasoningOrGPT5Model(modelName string) bool {
	name := strings.ToLower(strings.TrimSpace(modelName))
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "gpt-5") {
		return true
	}
	// o1 / o1-mini / o1-preview / o3 / o3-mini / o4-mini ...
	// Must match exactly to avoid false positives like "openai-...".
	for _, prefix := range []string{"o1", "o3", "o4"} {
		if name == prefix || strings.HasPrefix(name, prefix+"-") {
			return true
		}
	}
	return false
}
