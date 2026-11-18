// Package config handles subpanel's JSON configuration file and environment
// overrides, following the sbx convention (SUBPANEL_CONF / SUBPANEL_* env).
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const Version = "1.1.0"

// Config is the on-disk panel configuration.
type Config struct {
	Listen     string `json:"listen"`
	Data       string `json:"data"`
	AdminToken string `json:"admin_token"`
}

// DefaultPath returns the config file path from SUBPANEL_CONF or default.
func DefaultPath() string {
	if p := os.Getenv("SUBPANEL_CONF"); p != "" {
		return p
	}
	return "/etc/subpanel/config.json"
}

// Default returns built-in defaults (listen 20012, data /var/lib/subpanel/nodes.json).
func Default() Config {
	return Config{
		Listen: ":20012",
		Data:   "/var/lib/subpanel/nodes.json",
	}
}

// Load reads the config file, applying defaults for missing fields.
// Environment variables SUBPANEL_LISTEN / SUBPANEL_DATA / SUBPANEL_ADMIN
// override file values (highest priority).
func Load(path string) (Config, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &cfg); err != nil {
			return cfg, err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return cfg, err
	}
	if cfg.Listen == "" {
		cfg.Listen = Default().Listen
	}
	if cfg.Data == "" {
		cfg.Data = Default().Data
	}
	if v := os.Getenv("SUBPANEL_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("SUBPANEL_DATA"); v != "" {
		cfg.Data = v
	}
	if v := os.Getenv("SUBPANEL_ADMIN"); v != "" {
		cfg.AdminToken = v
	}
	return cfg, nil
}

// Save writes the config atomically with mode 0600.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Get returns a config field by key (listen/data/admin_token).
func (c Config) Get(key string) (string, bool) {
	switch key {
	case "listen":
		return c.Listen, true
	case "data":
		return c.Data, true
	case "admin_token", "admin":
		return c.AdminToken, true
	}
	return "", false
}

// Set updates a config field by key.
func (c *Config) Set(key, value string) bool {
	switch key {
	case "listen":
		c.Listen = value
	case "data":
		c.Data = value
	case "admin_token", "admin":
		c.AdminToken = value
	default:
		return false
	}
	return true
}
