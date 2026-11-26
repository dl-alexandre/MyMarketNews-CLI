package config

import (
	"testing"
	"time"
)

func TestConfigPreservesValues(t *testing.T) {
	cfg := Config{BaseURL: "https://example.test", Timeout: 12 * time.Second, RPS: 2.5, CacheTTL: time.Hour, CacheDir: t.TempDir()}
	if cfg.BaseURL != "https://example.test" || cfg.Timeout != 12*time.Second || cfg.RPS != 2.5 || cfg.CacheTTL != time.Hour || cfg.CacheDir == "" {
		t.Fatalf("config values were not preserved: %+v", cfg)
	}
}

func TestDefaultBaseURL(t *testing.T) {
	if DefaultBaseURL != "https://mpr.datamart.ams.usda.gov" {
		t.Fatalf("DefaultBaseURL = %q", DefaultBaseURL)
	}
}
