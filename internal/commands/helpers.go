package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"mpr/internal/cache"
	"mpr/internal/client"
	"mpr/internal/config"
	"mpr/internal/models"
)

func readConfig(cmdFlagGetter flagGetter) (config.Config, error) {
	baseURL, err := cmdFlagGetter.GetString("base-url")
	if err != nil {
		return config.Config{}, err
	}
	timeout, err := cmdFlagGetter.GetDuration("timeout")
	if err != nil {
		return config.Config{}, err
	}
	rps, err := cmdFlagGetter.GetFloat64("rps")
	if err != nil {
		return config.Config{}, err
	}
	cacheTTL, err := cmdFlagGetter.GetDuration("cache-ttl")
	if err != nil {
		return config.Config{}, err
	}
	cacheDir, err := cmdFlagGetter.GetString("cache-dir")
	if err != nil {
		return config.Config{}, err
	}

	return config.Config{
		BaseURL:  baseURL,
		Timeout:  timeout,
		RPS:      rps,
		CacheTTL: cacheTTL,
		CacheDir: cacheDir,
	}, nil
}

func newClient(cfg config.Config) *client.Client {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	limiter := client.NewRateLimiter(cfg.RPS)
	return client.New(httpClient, limiter, 6, "mpr-cli/0.1")
}

func loadReports(ctx context.Context, cfg config.Config, refresh bool) ([]models.ReportItem, bool, error) {
	c := newClient(cfg)
	return cache.LoadReports(ctx, c, cfg.CacheDir, cfg.CacheTTL, refresh, cfg.BaseURL)
}

func writeJSON(data []byte) {
	_, _ = os.Stdout.Write(data)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		_, _ = fmt.Fprintln(os.Stdout)
	}
}

type flagGetter interface {
	GetString(name string) (string, error)
	GetDuration(name string) (time.Duration, error)
	GetFloat64(name string) (float64, error)
}
