package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mic-360/wimo/internal/state"
)

type ConfigService struct {
	path   string
	config state.ConfigState
}

func NewConfigService() (*ConfigService, error) {
	configDir, err := defaultConfigDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return nil, err
	}
	service := &ConfigService{path: filepath.Join(configDir, "config.json"), config: defaultConfig()}
	if err := service.load(); err != nil {
		return nil, err
	}
	return service, nil
}

func defaultConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "WiMo"), nil
}

func defaultConfig() state.ConfigState {
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, "Projects"), filepath.Join(home, "Documents", "dev"))
	}
	return state.ConfigState{
		Theme:                  "default",
		ScanPaths:              paths,
		PurgeDepth:             6,
		RefreshIntervalSeconds: 3,
		WingetEnabled:          true,
		CheckUpdates:           true,
	}
}

func (c *ConfigService) load() error {
	payload, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return c.Save(c.config)
		}
		return err
	}
	payload = stripUTF8BOM(payload)
	var cfg state.ConfigState
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return err
	}
	cfg.ScanPaths = cleanPaths(cfg.ScanPaths)
	if cfg.PurgeDepth <= 0 {
		cfg.PurgeDepth = defaultConfig().PurgeDepth
	}
	if cfg.RefreshIntervalSeconds <= 0 {
		cfg.RefreshIntervalSeconds = defaultConfig().RefreshIntervalSeconds
	}
	if cfg.Theme == "" {
		cfg.Theme = defaultConfig().Theme
	}
	c.config = cfg
	return nil
}

func (c *ConfigService) State() state.ConfigState {
	copyState := c.config
	copyState.ScanPaths = append([]string{}, c.config.ScanPaths...)
	return copyState
}

func (c *ConfigService) Save(cfg state.ConfigState) error {
	cfg.ScanPaths = cleanPaths(cfg.ScanPaths)
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(c.path, payload, 0o644); err != nil {
		return err
	}
	c.config = cfg
	return nil
}

func (c *ConfigService) Path() string {
	return c.path
}

func (c *ConfigService) LogsDir() string {
	return filepath.Join(filepath.Dir(c.path), "logs")
}

func (c *ConfigService) RefreshInterval() time.Duration {
	seconds := c.config.RefreshIntervalSeconds
	if seconds <= 0 {
		seconds = 3
	}
	return time.Duration(seconds) * time.Second
}

func cleanPaths(paths []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		abs, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			continue
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		cleaned = append(cleaned, abs)
	}
	sort.Strings(cleaned)
	return cleaned
}
