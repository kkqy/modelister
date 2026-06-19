package providers

type Provider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	Note      string `json:"note"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProviderKey struct {
	ID            int64  `json:"id"`
	ProviderID    int64  `json:"provider_id"`
	Name          string `json:"name"`
	APIKey        string `json:"-"`
	APIKeyMasked  string `json:"api_key_masked"`
	Note          string `json:"note"`
	Enabled       bool   `json:"enabled"`
	LastSyncAt    string `json:"last_sync_at"`
	LastSyncError string `json:"last_sync_error"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type ProviderWithKey struct {
	Provider Provider
	Key      ProviderKey
}

type CreateProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Note    string `json:"note"`
	Enabled *bool  `json:"enabled"`
}

type UpdateProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}

type CreateKeyRequest struct {
	Name    string `json:"name"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
	Enabled *bool  `json:"enabled"`
}

type UpdateKeyRequest struct {
	Name    string `json:"name"`
	APIKey  string `json:"api_key"`
	Note    string `json:"note"`
	Enabled bool   `json:"enabled"`
}
