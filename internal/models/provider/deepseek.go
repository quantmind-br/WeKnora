package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// DeepSeekBaseURL the official DeepSeek API BaseURL
	DeepSeekBaseURL = "https://api.deepseek.com/v1"
)

// DeepSeekProvider implements the Provider interface for DeepSeek
type DeepSeekProvider struct{}

func init() {
	Register(&DeepSeekProvider{})
}

// Info returns metadata for the DeepSeek provider
func (p *DeepSeekProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderDeepSeek,
		DisplayName: "DeepSeek",
		Description: "deepseek-chat, deepseek-reasoner, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: DeepSeekBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates the DeepSeek provider configuration
func (p *DeepSeekProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for DeepSeek provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
