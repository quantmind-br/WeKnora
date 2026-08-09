package provider

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// AliyunChatBaseURL the default BaseURL for Alibaba Cloud DashScope Chat/Embedding
	AliyunChatBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	// AliyunRerankBaseURL the default BaseURL for Alibaba Cloud DashScope Rerank
	AliyunRerankBaseURL = "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
)

// AliyunProvider implements the Provider interface for Alibaba Cloud DashScope
type AliyunProvider struct{}

func init() {
	Register(&AliyunProvider{})
}

// Info returns metadata for the Aliyun provider
func (p *AliyunProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderAliyun,
		DisplayName: "Alibaba Cloud DashScope",
		Description: "qwen-plus, tongyi-embedding-vision-plus, qwen3-rerank, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: AliyunChatBaseURL,
			types.ModelTypeEmbedding:   AliyunChatBaseURL,
			types.ModelTypeRerank:      AliyunRerankBaseURL,
			types.ModelTypeVLLM:        AliyunChatBaseURL,
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

// ValidateConfig validates the Aliyun provider configuration
func (p *AliyunProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for Aliyun DashScope")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

// IsQwenThinkingModel checks whether the model name is a Qwen model that supports chain-of-thought reasoning
// Models that support chain-of-thought reasoning require special handling of the enable_thinking parameter
func IsQwenThinkingModel(modelName string) bool {
	lowerName := strings.ToLower(modelName)
	return strings.HasPrefix(lowerName, "qwen3") ||
		strings.HasPrefix(lowerName, "qwen-plus") ||
		strings.HasPrefix(lowerName, "qwen-max") ||
		strings.HasPrefix(lowerName, "qwen-turbo")
}

// IsQwen3Model checks whether the model belongs to the Qwen3 family only.
func IsQwen3Model(modelName string) bool {
	return strings.HasPrefix(strings.ToLower(modelName), "qwen3")
}

// IsDeepSeekModel checks whether the model name is a DeepSeek model
// DeepSeek models do not support the tool_choice parameter
func IsDeepSeekModel(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "deepseek")
}
