package client

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	httpClient *http.Client
	limiter    *RateLimiter
	maxRetries int
	userAgent  string
}

func New(httpClient *http.Client, limiter *RateLimiter, maxRetries int, userAgent string) *Client {
	if maxRetries <= 0 {
		maxRetries = 5
	}
	if userAgent == "" {
		userAgent = "mpr-cli/0.1"
	}
	return &Client{
		httpClient: httpClient,
		limiter:    limiter,
		maxRetries: maxRetries,
		userAgent:  userAgent,
	}
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, int, error) {
	var lastErr error
	var lastStatus int

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		if c.limiter != nil {
			c.limiter.Wait(ctx)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", c.userAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if isRetryableError(err) {
				lastErr = err
				c.sleepBackoff(attempt, resp)
				continue
			}
			return nil, 0, err
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, resp.StatusCode, readErr
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, resp.StatusCode, nil
		}

		if isRetryableStatus(resp.StatusCode) {
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			lastStatus = resp.StatusCode
			c.sleepBackoff(attempt, resp)
			continue
		}

		return body, resp.StatusCode, fmt.Errorf("http %d", resp.StatusCode)
	}

	if lastErr != nil {
		return nil, lastStatus, fmt.Errorf("request failed after retries: %w", lastErr)
	}

	return nil, lastStatus, errors.New("request failed after retries")
}

func (c *Client) sleepBackoff(attempt int, resp *http.Response) {
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
				time.Sleep(time.Duration(seconds) * time.Second)
				return
			}
		}
	}

	// Use safe exponential backoff calculation to avoid integer overflow
	var base time.Duration
	if attempt <= 1 {
		base = 1 * time.Second
	} else if attempt >= 6 {
		base = 30 * time.Second // Cap at 30s (2^5 = 32)
	} else {
		base = time.Duration(1<<(attempt-1)) * time.Second
	}

	jitter, _ := rand.Int(rand.Reader, big.NewInt(250))
	time.Sleep(base + time.Duration(jitter.Int64())*time.Millisecond)
}

func isRetryableStatus(code int) bool {
	if code == http.StatusTooManyRequests {
		return true
	}
	return code >= 500 && code <= 599
}

func isRetryableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}
