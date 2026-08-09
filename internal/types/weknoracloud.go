package types

// WeKnoraCloudStatusResult status check result
type WeKnoraCloudStatusResult struct {
	HasModels   bool   `json:"has_models"`       // Whether WeKnoraCloud credentials are configured
	NeedsReinit bool   `json:"needs_reinit"`     // Whether re-initialization is required (credentials corrupted)
	Reason      string `json:"reason,omitempty"` // Reason re-initialization is required
}
