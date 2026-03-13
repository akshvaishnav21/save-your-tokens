package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/saveyourtokens/syt/internal/utils"
)

// TrackingConfig holds SQLite tracking settings.
type TrackingConfig struct {
	DatabasePath string `toml:"database_path"`
	HistoryDays  int    `toml:"history_days"`
}

// HooksConfig holds hook-related settings.
type HooksConfig struct {
	ExcludeCommands []string `toml:"exclude_commands"`
}

// TeeConfig holds tee (output recovery) settings.
type TeeConfig struct {
	Enabled     bool   `toml:"enabled"`
	Mode        string `toml:"mode"`
	MinSize     int    `toml:"min_size"`
	MaxFiles    int    `toml:"max_files"`
	MaxFileSize int64  `toml:"max_file_size"`
	Directory   string `toml:"directory"`
}

// DisplayConfig holds display/UI settings.
type DisplayConfig struct {
	Colors       bool `toml:"colors"`
	UltraCompact bool `toml:"ultra_compact"`
}

// Config is the top-level configuration struct.
type Config struct {
	Tracking TrackingConfig `toml:"tracking"`
	Hooks    HooksConfig    `toml:"hooks"`
	Tee      TeeConfig      `toml:"tee"`
	Display  DisplayConfig  `toml:"display"`
}

func defaults() Config {
	return Config{
		Tracking: TrackingConfig{
			HistoryDays: 90,
		},
		Tee: TeeConfig{
			Enabled:     true,
			Mode:        "failures",
			MinSize:     500,
			MaxFiles:    20,
			MaxFileSize: 1048576,
			Directory:   filepath.Join(utils.DataDir(), "tee"),
		},
		Display: DisplayConfig{
			Colors:       true,
			UltraCompact: false,
		},
	}
}

// Load reads the config file then applies env var overrides.
// Never errors — returns defaults on any failure.
func Load() Config {
	cfg := defaults()

	cfgPath := filepath.Join(utils.ConfigDir(), "config.toml")
	if data, err := os.ReadFile(cfgPath); err == nil {
		// Ignore TOML decode errors; keep defaults for missing fields
		_ = toml.Unmarshal(data, &cfg)
		// Re-apply defaults for zero values
		if cfg.Tracking.HistoryDays == 0 {
			cfg.Tracking.HistoryDays = 90
		}
		if cfg.Tee.Mode == "" {
			cfg.Tee.Mode = "failures"
		}
		if cfg.Tee.MinSize == 0 {
			cfg.Tee.MinSize = 500
		}
		if cfg.Tee.MaxFiles == 0 {
			cfg.Tee.MaxFiles = 20
		}
		if cfg.Tee.MaxFileSize == 0 {
			cfg.Tee.MaxFileSize = 1048576
		}
		if cfg.Tee.Directory == "" {
			cfg.Tee.Directory = filepath.Join(utils.DataDir(), "tee")
		}
	}

	// Apply env var overrides
	if v := os.Getenv("SYT_DB_PATH"); v != "" {
		cfg.Tracking.DatabasePath = v
	}
	if v := os.Getenv("SYT_TEE"); v == "0" {
		cfg.Tee.Enabled = false
	}
	if v := os.Getenv("SYT_TEE_DIR"); v != "" {
		cfg.Tee.Directory = v
	}
	if v := os.Getenv("SYT_TEE_MODE"); v != "" {
		cfg.Tee.Mode = v
	}
	if v := os.Getenv("SYT_NO_COLOR"); v == "1" {
		cfg.Display.Colors = false
	}

	// Validate/sanitize paths
	if cfg.Tracking.DatabasePath != "" && !filepath.IsAbs(cfg.Tracking.DatabasePath) {
		cfg.Tracking.DatabasePath = ""
	}
	if cfg.Tee.Directory != "" && !filepath.IsAbs(cfg.Tee.Directory) {
		cfg.Tee.Directory = filepath.Join(utils.DataDir(), "tee")
	}

	return cfg
}
