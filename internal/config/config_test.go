package config

import "testing"

func TestLoadRequiresAdminUsernamePasswordAndSessionSecret(t *testing.T) {
	t.Setenv("APP_ADMIN_USERNAME", "")
	t.Setenv("APP_ADMIN_PASSWORD", "")
	t.Setenv("APP_SESSION_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing required env vars to fail")
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	t.Setenv("APP_ADMIN_USERNAME", "admin")
	t.Setenv("APP_ADMIN_PASSWORD", "secret")
	t.Setenv("APP_SESSION_SECRET", "session-secret")
	t.Setenv("APP_DATABASE_PATH", "")
	t.Setenv("APP_HTTP_ADDR", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config to load: %v", err)
	}
	if cfg.DatabasePath != "/data/modelister.db" {
		t.Fatalf("unexpected database path: %q", cfg.DatabasePath)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected http addr: %q", cfg.HTTPAddr)
	}
}
