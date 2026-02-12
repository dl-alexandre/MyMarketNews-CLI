package client

import (
	"context"
	"time"
)

type RateLimiter struct {
	ch <-chan time.Time
}

func NewRateLimiter(rps float64) *RateLimiter {
	if rps <= 0 {
		return nil
	}
	interval := time.Duration(float64(time.Second) / rps)
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	return &RateLimiter{ch: ticker.C}
}

func (r *RateLimiter) Wait(ctx context.Context) {
	if r == nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	case <-r.ch:
	}
}
