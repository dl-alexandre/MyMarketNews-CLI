package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"mpr/internal/client"
	"mpr/internal/models"
)

type ReportsCache struct {
	FetchedAt time.Time           `json:"fetched_at"`
	Reports   []models.ReportItem `json:"reports"`
}

func LoadReports(ctx context.Context, c *client.Client, cacheDir string, ttl time.Duration, refresh bool, baseURL string) ([]models.ReportItem, bool, error) {
	resolvedDir, err := ResolveCacheDir(cacheDir)
	if err != nil {
		return nil, false, err
	}

	if err := os.MkdirAll(resolvedDir, 0o750); err != nil {
		return nil, false, err
	}

	cachePath := filepath.Join(resolvedDir, "reports.json")
	if !refresh {
		if cached, ok := readCache(cachePath, ttl); ok {
			return cached.Reports, true, nil
		}
	}

	url := baseURL + "/services/v1.1/reports/"
	body, _, err := c.Get(ctx, url)
	if err != nil {
		if cached, ok := readCache(cachePath, 0); ok {
			return cached.Reports, true, nil
		}
		return nil, false, err
	}

	var reports []models.ReportItem
	if err := json.Unmarshal(body, &reports); err != nil {
		return nil, false, err
	}

	cache := ReportsCache{FetchedAt: time.Now(), Reports: reports}
	if err := writeCache(cachePath, cache); err != nil {
		return reports, false, nil
	}

	return reports, false, nil
}

func LoadReportsFromCache(cacheDir string) ([]models.ReportItem, bool, error) {
	resolvedDir, err := ResolveCacheDir(cacheDir)
	if err != nil {
		return nil, false, err
	}

	cachePath := filepath.Join(resolvedDir, "reports.json")
	cached, ok := readCache(cachePath, 0)
	if !ok {
		return nil, false, nil
	}
	return cached.Reports, true, nil
}

func ResolveCacheDir(cacheDir string) (string, error) {
	if cacheDir != "" {
		return cacheDir, nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mpr"), nil
}

func readCache(path string, ttl time.Duration) (*ReportsCache, bool) {
	// #nosec G304 - path is constructed from resolved internal cache directory only
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cache ReportsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	if ttl > 0 {
		if time.Since(cache.FetchedAt) > ttl {
			return nil, false
		}
	}

	return &cache, true
}

func writeCache(path string, cache ReportsCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	file, err := os.CreateTemp(filepath.Dir(path), "reports-*.json")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	if err := os.Rename(file.Name(), path); err != nil {
		return err
	}

	return nil
}

var ErrCacheMiss = errors.New("cache miss")
