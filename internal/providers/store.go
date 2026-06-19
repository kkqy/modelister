package providers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalid  = errors.New("invalid provider request")
	ErrNotFound = errors.New("provider resource not found")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateProvider(req CreateProviderRequest) (Provider, error) {
	name, baseURL, note, err := cleanProviderInput(req.Name, req.BaseURL, req.Note)
	if err != nil {
		return Provider{}, err
	}
	res, err := r.db.Exec(
		`insert into providers (name, base_url, note, enabled) values (?, ?, ?, ?)`,
		name, baseURL, note, boolInt(defaultTrue(req.Enabled)),
	)
	if err != nil {
		return Provider{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Provider{}, err
	}
	return r.GetProvider(id)
}

func (r *Repository) ListProviders() ([]Provider, error) {
	rows, err := r.db.Query(`select id, name, base_url, note, enabled, created_at, updated_at from providers order by id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}

func (r *Repository) GetProvider(id int64) (Provider, error) {
	row := r.db.QueryRow(`select id, name, base_url, note, enabled, created_at, updated_at from providers where id=?`, id)
	p, err := scanProvider(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Provider{}, ErrNotFound
	}
	return p, err
}

func (r *Repository) UpdateProvider(id int64, req UpdateProviderRequest) (Provider, error) {
	name, baseURL, note, err := cleanProviderInput(req.Name, req.BaseURL, req.Note)
	if err != nil {
		return Provider{}, err
	}
	res, err := r.db.Exec(
		`update providers set name=?, base_url=?, note=?, enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') where id=?`,
		name, baseURL, note, boolInt(req.Enabled), id,
	)
	if err != nil {
		return Provider{}, err
	}
	if err := ensureAffected(res); err != nil {
		return Provider{}, err
	}
	return r.GetProvider(id)
}

func (r *Repository) DeleteProvider(id int64) error {
	res, err := r.db.Exec(`delete from providers where id=?`, id)
	if err != nil {
		return err
	}
	return ensureAffected(res)
}

func (r *Repository) CreateKey(providerID int64, req CreateKeyRequest) (ProviderKey, error) {
	name := strings.TrimSpace(req.Name)
	apiKey := strings.TrimSpace(req.APIKey)
	note := strings.TrimSpace(req.Note)
	if name == "" || apiKey == "" {
		return ProviderKey{}, fmt.Errorf("%w: key name and api_key are required", ErrInvalid)
	}
	if _, err := r.GetProvider(providerID); err != nil {
		return ProviderKey{}, err
	}
	res, err := r.db.Exec(
		`insert into provider_keys (provider_id, name, api_key, note, enabled) values (?, ?, ?, ?, ?)`,
		providerID, name, apiKey, note, boolInt(defaultTrue(req.Enabled)),
	)
	if err != nil {
		return ProviderKey{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ProviderKey{}, err
	}
	return r.GetKey(providerID, id)
}

func (r *Repository) ListKeys(providerID int64) ([]ProviderKey, error) {
	rows, err := r.db.Query(`select id, provider_id, name, api_key, note, enabled, last_sync_at, last_sync_error, created_at, updated_at from provider_keys where provider_id=? order by id`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []ProviderKey
	for rows.Next() {
		k, err := scanKey(rows, false)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *Repository) GetKey(providerID, keyID int64) (ProviderKey, error) {
	return r.getKey(providerID, keyID, false)
}

func (r *Repository) GetKeyWithSecret(providerID, keyID int64) (ProviderKey, error) {
	return r.getKey(providerID, keyID, true)
}

func (r *Repository) getKey(providerID, keyID int64, includeSecret bool) (ProviderKey, error) {
	row := r.db.QueryRow(`select id, provider_id, name, api_key, note, enabled, last_sync_at, last_sync_error, created_at, updated_at from provider_keys where provider_id=? and id=?`, providerID, keyID)
	k, err := scanKey(row, includeSecret)
	if errors.Is(err, sql.ErrNoRows) {
		return ProviderKey{}, ErrNotFound
	}
	return k, err
}

func (r *Repository) UpdateKey(providerID, keyID int64, req UpdateKeyRequest) (ProviderKey, error) {
	name := strings.TrimSpace(req.Name)
	apiKey := strings.TrimSpace(req.APIKey)
	note := strings.TrimSpace(req.Note)
	if name == "" {
		return ProviderKey{}, fmt.Errorf("%w: key name is required", ErrInvalid)
	}
	if apiKey == "" {
		current, err := r.GetKeyWithSecret(providerID, keyID)
		if err != nil {
			return ProviderKey{}, err
		}
		apiKey = current.APIKey
	}
	res, err := r.db.Exec(
		`update provider_keys set name=?, api_key=?, note=?, enabled=?, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') where provider_id=? and id=?`,
		name, apiKey, note, boolInt(req.Enabled), providerID, keyID,
	)
	if err != nil {
		return ProviderKey{}, err
	}
	if err := ensureAffected(res); err != nil {
		return ProviderKey{}, err
	}
	return r.GetKey(providerID, keyID)
}

func (r *Repository) DeleteKey(providerID, keyID int64) error {
	res, err := r.db.Exec(`delete from provider_keys where provider_id=? and id=?`, providerID, keyID)
	if err != nil {
		return err
	}
	return ensureAffected(res)
}

func (r *Repository) ListEnabledProviderKeys() ([]ProviderWithKey, error) {
	rows, err := r.db.Query(`
		select
			p.id, p.name, p.base_url, p.note, p.enabled, p.created_at, p.updated_at,
			k.id, k.provider_id, k.name, k.api_key, k.note, k.enabled, k.last_sync_at, k.last_sync_error, k.created_at, k.updated_at
		from providers p
		join provider_keys k on k.provider_id = p.id
		where p.enabled=1 and k.enabled=1
		order by p.id, k.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ProviderWithKey
	for rows.Next() {
		var p Provider
		var k ProviderKey
		var pEnabled, kEnabled int
		if err := rows.Scan(
			&p.ID, &p.Name, &p.BaseURL, &p.Note, &pEnabled, &p.CreatedAt, &p.UpdatedAt,
			&k.ID, &k.ProviderID, &k.Name, &k.APIKey, &k.Note, &kEnabled, &k.LastSyncAt, &k.LastSyncError, &k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.Enabled = pEnabled == 1
		k.Enabled = kEnabled == 1
		k.APIKeyMasked = maskKey(k.APIKey)
		items = append(items, ProviderWithKey{Provider: p, Key: k})
	}
	return items, rows.Err()
}

func cleanProviderInput(name, rawBaseURL, note string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	rawBaseURL = strings.TrimRight(strings.TrimSpace(rawBaseURL), "/")
	note = strings.TrimSpace(note)
	if name == "" || rawBaseURL == "" {
		return "", "", "", fmt.Errorf("%w: provider name and base_url are required", ErrInvalid)
	}
	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", "", "", fmt.Errorf("%w: base_url must be http or https URL", ErrInvalid)
	}
	return name, rawBaseURL, note, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProvider(s scanner) (Provider, error) {
	var p Provider
	var enabled int
	err := s.Scan(&p.ID, &p.Name, &p.BaseURL, &p.Note, &enabled, &p.CreatedAt, &p.UpdatedAt)
	p.Enabled = enabled == 1
	return p, err
}

func scanKey(s scanner, includeSecret bool) (ProviderKey, error) {
	var k ProviderKey
	var enabled int
	var secret string
	err := s.Scan(&k.ID, &k.ProviderID, &k.Name, &secret, &k.Note, &enabled, &k.LastSyncAt, &k.LastSyncError, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return ProviderKey{}, err
	}
	k.Enabled = enabled == 1
	k.APIKeyMasked = maskKey(secret)
	if includeSecret {
		k.APIKey = secret
	}
	return k, nil
}

func maskKey(value string) string {
	if len(value) <= 8 {
		return "..." + value
	}
	return value[:3] + "..." + value[len(value)-4:]
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func defaultTrue(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
}

func ensureAffected(res sql.Result) error {
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
