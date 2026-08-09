package provider

import "github.com/Tencent/WeKnora/internal/types"

const (
	ProviderWeKnoraCloud ProviderName = "weknoracloud"

	// WeKnoraCloudBaseURL hardcoded Base URL for the WeKnoraCloud service (unified entry point; each implementation appends its own path)
	WeKnoraCloudBaseURL = "https://weknora.weixin.qq.com"
)

type WeKnoraCloudProvider struct{}

func init() {
	Register(&WeKnoraCloudProvider{})
}

func (p *WeKnoraCloudProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        ProviderWeKnoraCloud,
		DisplayName: "WeKnoraCloud",
		Description: "WeKnora cloud service, models: chat, embedding, rerank, vlm",
		DefaultURLs: map[types.ModelType]string{
			types.ModelTypeKnowledgeQA: WeKnoraCloudBaseURL,
			types.ModelTypeEmbedding:   WeKnoraCloudBaseURL,
			types.ModelTypeRerank:      WeKnoraCloudBaseURL,
			types.ModelTypeVLLM:        WeKnoraCloudBaseURL,
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

func (p *WeKnoraCloudProvider) ValidateConfig(config *Config) error {
	// AppID/AppSecret are written via a dedicated initialization endpoint; only structural validation is done here.
	// The AppSecret field currently actually holds the upstream API Key.
	return nil
}
