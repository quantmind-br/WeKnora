package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// NvidiaChatBaseURL default BaseURL for NVIDIA Chat
	NvidiaChatBaseURL = "https://integrate.api.nvidia.com/v1"
	// NvidiaRerankBaseURL default BaseURL for NVIDIA Rerank
	NvidiaRerankBaseURL = "https://ai.api.nvidia.com/v1/retrieval/nvidia/reranking"
)

// NvidiaProvider implements the Provider interface for NVIDIA AI
type NvidiaProvider struct{}

func init() {
	Register(&NvidiaProvider{})
}

// Info returns metadata for the NVIDIA provider
func (p *NvidiaProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderNvidia,
		DisplayName: "NVIDIA",
		Description: "deepseek-ai-deepseek-v3_1, nv-embed-v1, rerank-qa-mistral-4b, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: NvidiaChatBaseURL,
			types.ModelTypeEmbedding:   NvidiaChatBaseURL,
			types.ModelTypeRerank:      NvidiaRerankBaseURL,
			types.ModelTypeVLLM:        NvidiaChatBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeEmbedding,
			types.ModelTypeRerank,
			types.ModelTypeVLLM,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates NVIDIA provider configuration
func (p *NvidiaProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for NVIDIA")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
