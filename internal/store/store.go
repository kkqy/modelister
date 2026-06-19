package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`pragma foreign_keys = on`); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	statements := []string{
		`create table if not exists providers (
			id integer primary key autoincrement,
			name text not null,
			base_url text not null,
			note text not null default '',
			enabled integer not null default 1,
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create table if not exists provider_keys (
			id integer primary key autoincrement,
			provider_id integer not null references providers(id) on delete cascade,
			name text not null,
			api_key text not null,
			note text not null default '',
			enabled integer not null default 1,
			last_sync_at text not null default '',
			last_sync_error text not null default '',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create table if not exists model_cache (
			id integer primary key autoincrement,
			provider_id integer not null references providers(id) on delete cascade,
			provider_key_id integer not null references provider_keys(id) on delete cascade,
			model_id text not null,
			owned_by text not null default '',
			raw_json text not null default '{}',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			unique(provider_key_id, model_id)
		)`,
		`create index if not exists idx_provider_keys_provider_id on provider_keys(provider_id)`,
		`create index if not exists idx_model_cache_provider_key_id on model_cache(provider_key_id)`,
		`create index if not exists idx_model_cache_model_id on model_cache(model_id)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
