package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// WeKnoraCloudService handles WeKnoraCloud credential management
type WeKnoraCloudService interface {
	// SaveCredentials only saves APPID/APPSECRET credentials to the space configuration, without automatically creating a model
	SaveCredentials(ctx context.Context, appID, appSecret string) error
	// CheckStatus checks whether the current space's WeKnoraCloud credentials can be decrypted successfully
	// needsReinit=true indicates the encryption state is corrupted (salt changed, etc.) and the user needs to re-enter credentials
	CheckStatus(ctx context.Context) (*types.WeKnoraCloudStatusResult, error)
}
