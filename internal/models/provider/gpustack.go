package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// GPUStackBaseURL GPUStack API BaseURL (OpenAI-compatible mode)
	GPUStackBaseURL = "http://your_gpustack_server_url/v1-openai"
	// GPUStackRerankBaseURL GPUStack Rerank API — although compatible with OpenAI, the path differs (/v1/rerank instead of /v1-openai/rerank)
	GPUStackRerankBaseURL = "http://your_gpustack_server_url/v1"
)

// GPUStackProvider implements the Provider interface for GPUStack
type GPUStackProvider struct{}

func init() {
	Register(&GPUStackProvider{})
}

// Info returns metadata for the GPUStack provider
func (p *GPUStackProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderGPUStack,
		DisplayName: "GPUStack",
		Description: "Choose your deployed model on GPUStack",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: GPUStackBaseURL,
			types.ModelTypeEmbedding:   GPUStackBaseURL,
			types.ModelTypeRerank:      GPUStackRerankBaseURL,
			types.ModelTypeVLLM:        GPUStackBaseURL,
			types.ModelTypeASR:         GPUStackBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
			types.ModelTypeASR,
		},
		RequiresAuth: true, // GPUStack requires an API Key
	}
}

// ValidateConfig validates the GPUStack provider configuration
func (p *GPUStackProvider) ValidateConfig(config *Config) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base URL is required for GPUStack provider")
	}
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for GPUStack provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
