package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigServiceLoadStripsUTF8BOM(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	payload := []byte{0xEF, 0xBB, 0xBF, '{', '"', 'T', 'h', 'e', 'm', 'e', '"', ':', '"', 'd', 'e', 'f', 'a', 'u', 'l', 't', '"', ',', '"', 'S', 'c', 'a', 'n', 'P', 'a', 't', 'h', 's', '"', ':', '[', '"', 'C', ':', '\\', '\\', 'W', 'o', 'r', 'k', '"', ']', ',', '"', 'P', 'u', 'r', 'g', 'e', 'D', 'e', 'p', 't', 'h', '"', ':', '4', ',', '"', 'R', 'e', 'f', 'r', 'e', 's', 'h', 'I', 'n', 't', 'e', 'r', 'v', 'a', 'l', 'S', 'e', 'c', 'o', 'n', 'd', 's', '"', ':', '5', ',', '"', 'W', 'i', 'n', 'g', 'e', 't', 'E', 'n', 'a', 'b', 'l', 'e', 'd', '"', ':', 't', 'r', 'u', 'e', ',', '"', 'C', 'h', 'e', 'c', 'k', 'U', 'p', 'd', 'a', 't', 'e', 's', '"', ':', 't', 'r', 'u', 'e', '}'}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	service := &ConfigService{path: path, config: defaultConfig()}
	if err := service.load(); err != nil {
		t.Fatalf("load config with BOM: %v", err)
	}

	cfg := service.State()
	if cfg.PurgeDepth != 4 {
		t.Fatalf("purge depth = %d, want 4", cfg.PurgeDepth)
	}
	if cfg.RefreshIntervalSeconds != 5 {
		t.Fatalf("refresh interval = %d, want 5", cfg.RefreshIntervalSeconds)
	}
	if len(cfg.ScanPaths) != 1 {
		t.Fatalf("scan path count = %d, want 1", len(cfg.ScanPaths))
	}
}
