package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/provider"
	modelsutils "github.com/Tencent/WeKnora/internal/models/utils"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
)

type weKnoraCloudService struct {
	tenantRepo interfaces.TenantRepository
}

// NewWeKnoraCloudService constructs a WeKnoraCloudService
func NewWeKnoraCloudService(
	repo interfaces.ModelRepository,
	tenantRepo interfaces.TenantRepository,
) interfaces.WeKnoraCloudService {
	return &weKnoraCloudService{
		tenantRepo: tenantRepo,
	}
}

func IsWeKnoraCloudDocReaderAddr(addr string) bool {
	return strings.TrimSuffix(strings.TrimSpace(addr), "/") == strings.TrimRight(provider.WeKnoraCloudBaseURL, "/")+"/api/v1/doc/reader"
}

// SaveCredentials only saves the APPID/APPSECRET credentials, without automatically creating a model
func (s *weKnoraCloudService) SaveCredentials(ctx context.Context, appID, appSecret string) error {
	if appID == "" {
		return fmt.Errorf("app_id is required")
	}
	if appSecret == "" {
		return fmt.Errorf("app_secret is required")
	}

	if err := s.verifyCredentials(ctx, appID, appSecret); err != nil {
		return fmt.Errorf("credential verification failed: %w", err)
	}

	tenantID := types.MustTenantIDFromContext(ctx)
	return s.updateTenantCredentials(ctx, tenantID, appID, appSecret)
}

// verifyCredentials sends a signed GET request to WeKnoraCloud /api/v1/health.
//
// Note: health is generally a liveness-check endpoint; the remote side often doesn't validate APPID/SECRET or the signature — an HTTP 200 usually only means
// the "gateway/service is reachable," not strict proof that the credentials are valid. For strict validation, call a business endpoint that requires auth instead.
func (s *weKnoraCloudService) verifyCredentials(ctx context.Context, appID, appSecret string) error {
	baseURL := strings.TrimRight(provider.WeKnoraCloudBaseURL, "/")
	healthURL := baseURL + "/api/v1/health"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
	if err != nil {
		return fmt.Errorf("create verification request failed: %w", err)
	}

	requestID := fmt.Sprintf("verify-%d", time.Now().UnixNano())
	signHeaders := modelsutils.Sign(appID, appSecret, requestID, "{}")
	for k, v := range signHeaders {
		req.Header.Set(k, v)
	}

	logger.Infof(ctx, "credential verification request: method=GET url=%s app_id=%s request_id=%s ",
		healthURL, appID, requestID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warnf(ctx, "credential verification HTTP failed: url=%s err=%v", healthURL, err)
		return fmt.Errorf("service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid APPID or APPSECRET (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid response status code: %d", resp.StatusCode)
	}
	return nil
}

// CheckStatus checks whether WeKnoraCloud credentials can be decrypted correctly
func (s *weKnoraCloudService) CheckStatus(ctx context.Context) (*types.WeKnoraCloudStatusResult, error) {
	tenantID := types.MustTenantIDFromContext(ctx)

	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil || tenant == nil {
		return &types.WeKnoraCloudStatusResult{HasModels: false, NeedsReinit: false}, nil
	}

	creds := tenant.Credentials.GetWeKnoraCloud()
	if creds == nil {
		return &types.WeKnoraCloudStatusResult{HasModels: false, NeedsReinit: false}, nil
	}

	// CredentialsConfig.Scan already attempts decryption.
	// If the AES key has rotated, Scan silently keeps the enc:v1:... blob.
	if strings.HasPrefix(creds.AppSecret, utils.EncPrefix) {
		return &types.WeKnoraCloudStatusResult{
			HasModels:   true,
			NeedsReinit: true,
			Reason:      "WeKnoraCloud credentials could not be decrypted (encryption key changed after restart); please re-enter APPID and APPSECRET",
		}, nil
	}

	return &types.WeKnoraCloudStatusResult{HasModels: true, NeedsReinit: false}, nil
}

// updateTenantCredentials updates the tenant's WeKnoraCloud credentials
func (s *weKnoraCloudService) updateTenantCredentials(ctx context.Context, tenantID uint64, appID, appSecret string) error {
	if s.tenantRepo == nil {
		return fmt.Errorf("tenant repository is required")
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.Credentials == nil {
		tenant.Credentials = &types.CredentialsConfig{}
	}
	tenant.Credentials.WeKnoraCloud = &types.WeKnoraCloudCredentials{
		AppID:     appID,
		AppSecret: appSecret,
	}
	return s.tenantRepo.UpdateTenant(ctx, tenant)
}
