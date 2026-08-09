package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

// GenericProvider implements a generic OpenAI-compatible Provider interface
type GenericProvider struct{}

func init() {
	Register(&GenericProvider{})
}

// Info returns metadata for the generic provider
func (p *GenericProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderGeneric,
		DisplayName: "Custom (OpenAI-compatible)",
		Description: "Generic API endpoint (OpenAI-compatible)",
		DefaultURLs: map[types.ModelType]string{}, // Must be configured and filled in by the user
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
			types.ModelTypeASR,
		},
		RequiresAuth: false, // May or may not be required
	}
}

// ValidateConfig validates the generic provider configuration
func (p *GenericProvider) ValidateConfig(config *Config) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required for generic provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
