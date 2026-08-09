package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// HunyuanBaseURL Tencent Hunyuan API BaseURL (OpenAI-compatible mode)
	HunyuanBaseURL = "https://api.hunyuan.cloud.tencent.com/v1"
)

// HunyuanProvider implements the Provider interface for Tencent Hunyuan
type HunyuanProvider struct{}

func init() {
	Register(&HunyuanProvider{})
}

// Info returns metadata for the Tencent Hunyuan provider
func (p *HunyuanProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderHunyuan,
		DisplayName: "Tencent Hunyuan",
		Description: "hunyuan-pro, hunyuan-standard, hunyuan-embedding, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: HunyuanBaseURL,
			types.ModelTypeEmbedding:   HunyuanBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates the Tencent Hunyuan provider configuration
func (p *HunyuanProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for Hunyuan provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
