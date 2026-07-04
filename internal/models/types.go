package models

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by"`
	RawJSON string `json:"-"`
}

type KeyModels struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Note          string  `json:"note"`
	LastSyncAt    string  `json:"last_sync_at"`
	LastSyncError string  `json:"last_sync_error"`
	Models        []Model `json:"models"`
}

type ProviderModels struct {
	ID      int64       `json:"id"`
	Name    string      `json:"name"`
	BaseURL string      `json:"base_url"`
	Note    string      `json:"note"`
	Keys    []KeyModels `json:"keys,omitempty"`
	Models  []Model     `json:"models,omitempty"`
}

type ListResponse struct {
	Mode      string           `json:"mode"`
	Query     string           `json:"query,omitempty"`
	Providers []ProviderModels `json:"providers"`
}

type ModelChangeEvent struct {
	ID            int64    `json:"id"`
	ProviderID    int64    `json:"provider_id"`
	KeyID         int64    `json:"key_id"`
	ProviderName  string   `json:"provider_name"`
	KeyName       string   `json:"key_name"`
	BaseURL       string   `json:"base_url"`
	AddedCount    int      `json:"added_count"`
	RemovedCount  int      `json:"removed_count"`
	AddedModels   []string `json:"added_models"`
	RemovedModels []string `json:"removed_models"`
	CreatedAt     string   `json:"created_at"`
}

type ModelChangeGroup struct {
	ID           int64                      `json:"id"`
	CreatedAt    string                     `json:"created_at"`
	AddedCount   int                        `json:"added_count"`
	RemovedCount int                        `json:"removed_count"`
	Providers    []ModelChangeProviderGroup `json:"providers"`
}

type ModelChangeProviderGroup struct {
	ProviderID   int64                 `json:"provider_id"`
	ProviderName string                `json:"provider_name"`
	BaseURL      string                `json:"base_url"`
	AddedCount   int                   `json:"added_count"`
	RemovedCount int                   `json:"removed_count"`
	Keys         []ModelChangeKeyEvent `json:"keys"`
}

type ModelChangeKeyEvent struct {
	ID            int64    `json:"id"`
	KeyID         int64    `json:"key_id"`
	KeyName       string   `json:"key_name"`
	AddedCount    int      `json:"added_count"`
	RemovedCount  int      `json:"removed_count"`
	AddedModels   []string `json:"added_models"`
	RemovedModels []string `json:"removed_models"`
	CreatedAt     string   `json:"created_at"`
}

type ChangeEventsResponse struct {
	Groups       []ModelChangeGroup `json:"groups"`
	HasMore      bool               `json:"has_more"`
	NextBeforeID int64              `json:"next_before_id,omitempty"`
}

type SyncResult struct {
	ProviderID int64  `json:"provider_id"`
	KeyID      int64  `json:"key_id"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	Count      int    `json:"count"`
}

type SyncedKeyModels struct {
	ProviderID   int64
	ProviderName string
	BaseURL      string
	KeyID        int64
	KeyName      string
	Models       []Model
}
