package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Mode != "server" {
		t.Errorf("expected mode 'server', got %q", cfg.Mode)
	}
	if cfg.ServerPort != 24800 {
		t.Errorf("expected port 24800, got %d", cfg.ServerPort)
	}
	if len(cfg.Screens) != 4 {
		t.Errorf("expected 4 screens, got %d", len(cfg.Screens))
	}
}

func TestValidate_valid(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestValidate_badMode(t *testing.T) {
	cfg := Default()
	cfg.Mode = "foobar"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestValidate_badPort(t *testing.T) {
	cfg := Default()
	cfg.ServerPort = 80
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for privileged port")
	}
}

func TestValidate_emptyScreens(t *testing.T) {
	cfg := Default()
	cfg.Screens = nil
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for empty screens")
	}
}

func TestValidate_tooManyScreens(t *testing.T) {
	cfg := Default()
	cfg.Screens = append(cfg.Screens, ScreenConfig{ID: 4, Name: "Extra"})
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for more than 4 screens")
	}
}

func TestLoad_missingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.json")
	if err != nil {
		t.Errorf("missing file should return default, got err: %v", err)
	}
	if cfg.ServerPort != 24800 {
		t.Error("expected default config when file missing")
	}
}

func TestSaveAndLoad_roundtrip(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.json")
	orig := Default()
	orig.LogLevel = "debug"
	orig.Screens[1].IP = "10.0.0.5"

	if err := orig.Save(tmp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.LogLevel != "debug" {
		t.Errorf("LogLevel not persisted: got %q", loaded.LogLevel)
	}
	if loaded.Screens[1].IP != "10.0.0.5" {
		t.Errorf("Screens[1].IP not persisted: got %q", loaded.Screens[1].IP)
	}
}

func TestLoad_invalidJSON(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(tmp, []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(tmp)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
