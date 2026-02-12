package config

import "time"

const DefaultBaseURL = "https://mpr.datamart.ams.usda.gov"

type Config struct {
	BaseURL  string
	Timeout  time.Duration
	RPS      float64
	CacheTTL time.Duration
	CacheDir string
}
