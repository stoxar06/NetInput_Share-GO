// Package config loads and saves the NetInput Share configuration file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ScreenConfig defines a single laptop in the layout.
type ScreenConfig struct {
	ID     uint8  `json:"id"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Config is the top-level configuration structure.
type Config struct {
	Mode          string         `json:"mode"`
	ServerPort    int            `json:"server_port"`
	ServerIP      string         `json:"server_ip"`
	Screens       []ScreenConfig `json:"screens"`
	Layout        string         `json:"layout"`
	HotkeySwitch  string         `json:"hotkey_switch"`
	HotkeyBack    string         `json:"hotkey_back"`
	EdgeSwitch    bool           `json:"edge_switch"`
	ClipboardSync bool           `json:"clipboard_sync"`
	LogLevel      string         `json:"log_level"`
}

// Default returns a sensible default configuration.
func Default() Config {
	return Config{
		Mode:          "server",
		ServerPort:    24800,
		ServerIP:      "auto",
		Layout:        "horizontal",
		HotkeySwitch:  "ctrl+alt+right",
		HotkeyBack:    "ctrl+alt+left",
		EdgeSwitch:    true,
		ClipboardSync: true,
		LogLevel:      "info",
		Screens: []ScreenConfig{
			{ID: 0, Name: "Main", IP: "self", Width: 1920, Height: 1080},
			{ID: 1, Name: "Laptop2", IP: "192.168.1.101", Width: 1920, Height: 1080},
			{ID: 2, Name: "Laptop3", IP: "192.168.1.102", Width: 1920, Height: 1080},
			{ID: 3, Name: "Laptop4", IP: "192.168.1.103", Width: 1920, Height: 1080},
		},
	}
}

// Load reads config from path. Returns Default() merged with file values.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config load: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config parse: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config invalid: %w", err)
	}
	return cfg, nil
}

// Save writes config to path as pretty-printed JSON.
func (c Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("config save: %w", err)
	}
	return nil
}

// Validate checks config for obvious errors.
func (c Config) Validate() error {
	if c.Mode != "server" && c.Mode != "client" {
		return fmt.Errorf("mode must be 'server' or 'client', got %q", c.Mode)
	}
	if c.ServerPort < 1024 || c.ServerPort > 65535 {
		return fmt.Errorf("server_port %d out of range", c.ServerPort)
	}
	if len(c.Screens) == 0 {
		return fmt.Errorf("screens list is empty")
	}
	if len(c.Screens) > 4 {
		return fmt.Errorf("max 4 screens supported, got %d", len(c.Screens))
	}
	return nil
}
