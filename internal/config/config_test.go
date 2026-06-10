package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUses6060AsDefaultAppPort(t *testing.T) {
	t.Setenv("APP_PORT", "不是端口")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	// 仅写入最小配置，确保应用端口走默认值分支，不受 YAML 显式端口影响。
	if err := os.WriteFile(configPath, []byte("app:\n  env: test\n"), 0o600); err != nil {
		t.Fatalf("write config failed: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}
	if cfg.App.Port != 6060 {
		t.Fatalf("expected default app port 6060, got %d", cfg.App.Port)
	}
}
