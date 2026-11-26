package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetReturnsResponseAndSetsUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("User-Agent"); got != "test-mpr/1.2" {
			t.Errorf("User-Agent = %q, want test-mpr/1.2", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := New(server.Client(), nil, 1, "test-mpr/1.2")
	body, status, err := c.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if status != http.StatusOK || string(body) != `{"ok":true}` {
		t.Fatalf("Get() = (%q, %d), want response body and 200", body, status)
	}
}

func TestGetReturnsNonRetryableHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	body, status, err := New(server.Client(), nil, 3, "").Get(context.Background(), server.URL)
	if err == nil || err.Error() != "http 400" {
		t.Fatalf("Get() error = %v, want http 400", err)
	}
	if status != http.StatusBadRequest || !strings.Contains(string(body), "bad request") {
		t.Fatalf("Get() = (%q, %d), want error response body and 400", body, status)
	}
}

func TestRetryableStatus(t *testing.T) {
	for _, tc := range []struct {
		code int
		want bool
	}{{http.StatusTooManyRequests, true}, {http.StatusInternalServerError, true}, {http.StatusBadGateway, true}, {http.StatusOK, false}, {http.StatusBadRequest, false}} {
		if got := isRetryableStatus(tc.code); got != tc.want {
			t.Errorf("isRetryableStatus(%d) = %t, want %t", tc.code, got, tc.want)
		}
	}
}
