package store

import (
	"database/sql"
	"testing"
)

func TestOpenInitializesSchema(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	tables := []string{"providers", "provider_keys", "model_cache"}
	for _, table := range tables {
		var name string
		err := db.QueryRow(`select name from sqlite_master where type='table' and name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("expected table %s to exist: %v", table, err)
		}
	}
}

func TestForeignKeysCascadeProviderDelete(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`insert into providers (name, base_url, note, enabled) values ('p', 'https://example.com', '', 1)`)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	providerID, err := lastID(db)
	if err != nil {
		t.Fatalf("provider id: %v", err)
	}
	_, err = db.Exec(`insert into provider_keys (provider_id, name, api_key, note, enabled) values (?, 'k', 'secret', '', 1)`, providerID)
	if err != nil {
		t.Fatalf("insert key: %v", err)
	}
	keyID, err := lastID(db)
	if err != nil {
		t.Fatalf("key id: %v", err)
	}
	_, err = db.Exec(`insert into model_cache (provider_id, provider_key_id, model_id, owned_by, raw_json) values (?, ?, 'gpt-4o', 'openai', '{}')`, providerID, keyID)
	if err != nil {
		t.Fatalf("insert model: %v", err)
	}

	if _, err := db.Exec(`delete from providers where id=?`, providerID); err != nil {
		t.Fatalf("delete provider: %v", err)
	}

	assertCount(t, db, "provider_keys", 0)
	assertCount(t, db, "model_cache", 0)
}

func lastID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow(`select last_insert_rowid()`).Scan(&id)
	return id, err
}

func assertCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`select count(*) from ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("count %s = %d, want %d", table, got, want)
	}
}
