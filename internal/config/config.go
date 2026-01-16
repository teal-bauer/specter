package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds a single instance configuration
type Config struct {
	URL string `yaml:"url"`
	Key string `yaml:"key"`
}

// FileConfig holds the full config file structure
type FileConfig struct {
	Default   string            `yaml:"default"`
	Instances map[string]Config `yaml:"instances"`
	// Legacy single-instance fields for backwards compatibility
	URL string `yaml:"url,omitempty"`
	Key string `yaml:"key,omitempty"`
}

// Global flags set by CLI
var (
	FlagURL     string
	FlagKey     string
	FlagOutput  string
	FlagProfile string
)

// Load reads configuration from file, environment, and CLI flags
// Priority: CLI flags > env vars > config file
func Load() (*Config, error) {
	cfg := &Config{}

	// Try config file first
	fileCfg, _ := loadFileConfig()
	if fileCfg != nil {
		// Determine which profile to use
		profile := FlagProfile
		if profile == "" {
			profile = os.Getenv("GHOST_PROFILE")
		}
		if profile == "" {
			profile = fileCfg.Default
		}

		if profile != "" && fileCfg.Instances != nil {
			if inst, ok := fileCfg.Instances[profile]; ok {
				cfg.URL = inst.URL
				cfg.Key = inst.Key
			}
		}

		// Fallback to legacy single-instance config
		if cfg.URL == "" && fileCfg.URL != "" {
			cfg.URL = fileCfg.URL
			cfg.Key = fileCfg.Key
		}
	}

	// Environment variables override config file
	if url := os.Getenv("GHOST_URL"); url != "" {
		cfg.URL = url
	}
	if key := os.Getenv("GHOST_ADMIN_KEY"); key != "" {
		cfg.Key = key
	}

	// CLI flags override everything
	if FlagURL != "" {
		cfg.URL = FlagURL
	}
	if FlagKey != "" {
		cfg.Key = FlagKey
	}

	// Validate
	if cfg.URL == "" {
		return nil, fmt.Errorf("ghost URL not configured (use 'specter login', set GHOST_URL, or use --url)")
	}
	if cfg.Key == "" {
		return nil, fmt.Errorf("ghost admin key not configured (use 'specter login', set GHOST_ADMIN_KEY, or use --key)")
	}

	return cfg, nil
}

// ConfigPath returns the path to the config file
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "specter", "config.yaml")
}

func loadFileConfig() (*FileConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	paths := []string{
		filepath.Join(home, ".config", "specter", "config.yaml"),
		filepath.Join(home, ".specter.yaml"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg FileConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		return &cfg, nil
	}

	return nil, fmt.Errorf("no config file found")
}

// SaveInstance saves an instance configuration to the config file
func SaveInstance(name string, cfg Config, setDefault bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".config", "specter")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Load existing config or create new
	fileCfg, _ := loadFileConfig()
	if fileCfg == nil {
		fileCfg = &FileConfig{
			Instances: make(map[string]Config),
		}
	}
	if fileCfg.Instances == nil {
		fileCfg.Instances = make(map[string]Config)
	}

	// Migrate legacy config if present
	if fileCfg.URL != "" && len(fileCfg.Instances) == 0 {
		fileCfg.Instances["default"] = Config{
			URL: fileCfg.URL,
			Key: fileCfg.Key,
		}
		if fileCfg.Default == "" {
			fileCfg.Default = "default"
		}
		fileCfg.URL = ""
		fileCfg.Key = ""
	}

	// Add/update instance
	fileCfg.Instances[name] = cfg

	// Set as default if requested or if first instance
	if setDefault || fileCfg.Default == "" {
		fileCfg.Default = name
	}

	// Write config
	data, err := yaml.Marshal(fileCfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// ListInstances returns all configured instance names
func ListInstances() ([]string, string, error) {
	fileCfg, err := loadFileConfig()
	if err != nil {
		return nil, "", err
	}

	var names []string
	for name := range fileCfg.Instances {
		names = append(names, name)
	}

	return names, fileCfg.Default, nil
}

// OutputFormat returns the configured output format
func OutputFormat() string {
	if FlagOutput != "" {
		return FlagOutput
	}
	return "text"
}
