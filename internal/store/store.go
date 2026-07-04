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
		`create table if not exists model_change_runs (
			id integer primary key autoincrement,
			scope text not null default '',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create table if not exists model_change_events (
			id integer primary key autoincrement,
			run_id integer not null references model_change_runs(id) on delete cascade,
			provider_id integer not null,
			provider_key_id integer not null,
			provider_name text not null,
			key_name text not null,
			base_url text not null,
			added_count integer not null default 0,
			removed_count integer not null default 0,
			added_models text not null default '[]',
			removed_models text not null default '[]',
			created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)`,
		`create index if not exists idx_provider_keys_provider_id on provider_keys(provider_id)`,
		`create index if not exists idx_model_cache_provider_key_id on model_cache(provider_key_id)`,
		`create index if not exists idx_model_cache_model_id on model_cache(model_id)`,
		`create index if not exists idx_model_change_runs_id on model_change_runs(id)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := ensureModelChangeEventsSchema(db); err != nil {
		return err
	}
	eventIndexes := []string{
		`create index if not exists idx_model_change_events_id on model_change_events(id)`,
		`create index if not exists idx_model_change_events_run_id on model_change_events(run_id)`,
		`create index if not exists idx_model_change_events_provider_key_id on model_change_events(provider_key_id)`,
	}
	for _, stmt := range eventIndexes {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

func ensureModelChangeEventsSchema(db *sql.DB) error {
	hasRunID, err := columnExists(db, "model_change_events", "run_id")
	if err != nil {
		return err
	}
	if hasRunID {
		return nil
	}
	if _, err := db.Exec(`drop table if exists model_change_events`); err != nil {
		return fmt.Errorf("drop legacy model_change_events: %w", err)
	}
	_, err = db.Exec(`create table model_change_events (
		id integer primary key autoincrement,
		run_id integer not null references model_change_runs(id) on delete cascade,
		provider_id integer not null,
		provider_key_id integer not null,
		provider_name text not null,
		key_name text not null,
		base_url text not null,
		added_count integer not null default 0,
		removed_count integer not null default 0,
		added_models text not null default '[]',
		removed_models text not null default '[]',
		created_at text not null default (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`)
	return err
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`pragma table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
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
