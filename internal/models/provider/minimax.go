package provider

import (
	"fmt"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	// MiniMaxBaseURL MiniMax international API BaseURL
	MiniMaxBaseURL = "https://api.minimax.io/v1"
	// MiniMaxCNBaseURL MiniMax China API BaseURL
	MiniMaxCNBaseURL = "https://api.minimaxi.com/v1"
)

// MiniMaxProvider implements the Provider interface for MiniMax
type MiniMaxProvider struct{}

func init() {
	Register(&MiniMaxProvider{})
}

// Info returns metadata for the MiniMax provider
func (p *MiniMaxProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderMiniMax,
		DisplayName: "MiniMax",
		Description: "MiniMax-M3, MiniMax-M2.7, MiniMax-M2.7-highspeed, etc.",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: MiniMaxCNBaseURL,
		},
		ModelTypes: []types.ModelType{
			types.ModelTypeKnowledgeQA,
		},
		RequiresAuth: true,
	}
}

// ValidateConfig validates the MiniMax provider configuration
func (p *MiniMaxProvider) ValidateConfig(config *Config) error {
	if config.APIKey == "" {
		return fmt.Errorf("API key is required for MiniMax provider")
	}
	if config.ModelName == "" {
		return fmt.Errorf("model name is required")
	}
	return nil
}
