package provider

import (
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// LKEAPBaseURL Tencent Cloud LKEAP (Large Knowledge Enhanced Application Platform) BaseURL for the OpenAI-compatible protocol
	LKEAPBaseURL = "https://api.lkeap.cloud.tencent.com/v1"
	// LKEAPRerankBaseURL Tencent Cloud LKEAP Rerank API domain (TC3 signature)
	LKEAPRerankBaseURL = "https://lkeap.tencentcloudapi.com"
)

// LKEAPProvider implements the Provider interface for Tencent Cloud LKEAP
// Supports DeepSeek-R1, DeepSeek-V3 series models, with chain-of-thought capability
type LKEAPProvider struct{}

func init() {
	Register(&LKEAPProvider{})
}

// Info returns metadata for the LKEAP provider
func (p *LKEAPProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderLKEAP,
		DisplayName: "Tencent Cloud LKEAP",
		Description: "DeepSeek-R1, DeepSeek-V3, lke-reranker-base, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: LKEAPBaseURL,
			types.ModelTypeRerank:      LKEAPRerankBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
			types.ModelTypeRerank,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates the LKEAP provider configuration
func (p *LKEAPProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for LKEAP provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}

// IsLKEAPDeepSeekV3Model checks whether it's a DeepSeek V3.x series model
// The V3.x series supports toggling chain-of-thought via the Thinking parameter
func IsLKEAPDeepSeekV3Model(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "deepseek-v3")
}

// IsLKEAPDeepSeekR1Model checks whether it's a DeepSeek R1 series model
// The R1 series has chain-of-thought enabled by default
func IsLKEAPDeepSeekR1Model(modelName string) bool {
	return strings.Contains(strings.ToLower(modelName), "deepseek-r1")
}

// IsLKEAPThinkingModel checks whether it's an LKEAP model that supports chain-of-thought
func IsLKEAPThinkingModel(modelName string) bool {
	return IsLKEAPDeepSeekR1Model(modelName) || IsLKEAPDeepSeekV3Model(modelName)
}
