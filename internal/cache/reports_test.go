package cache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mpr/internal/client"
	"mpr/internal/models"
)

func TestLoadReportsFetchesAndCaches(t *testing.T) {
	want := []models.ReportItem{{SlugID: 7, ReportTitle: "Cattle"}}
	data, _ := json.Marshal(want)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/services/v1.1/reports/" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()

	dir := t.TempDir()
	c := client.New(server.Client(), nil, 1, "test")
	got, fromCache, err := LoadReports(context.Background(), c, dir, time.Hour, false, server.URL)
	if err != nil || fromCache || len(got) != 1 || got[0].SlugID != 7 {
		t.Fatalf("first LoadReports() = (%+v, %t, %v)", got, fromCache, err)
	}
	got, fromCache, err = LoadReports(context.Background(), c, dir, time.Hour, false, server.URL)
	if err != nil || !fromCache || requests != 1 || got[0].ReportTitle != "Cattle" {
		t.Fatalf("cached LoadReports() = (%+v, %t, %v), requests=%d", got, fromCache, err, requests)
	}
}

func TestLoadReportsUsesStaleCacheOnFetchError(t *testing.T) {
	dir := t.TempDir()
	want := ReportsCache{FetchedAt: time.Now().Add(-time.Hour), Reports: []models.ReportItem{{SlugID: 9}}}
	data, _ := json.Marshal(want)
	if err := os.WriteFile(filepath.Join(dir, "reports.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadRequest) }))
	defer server.Close()

	got, fromCache, err := LoadReports(context.Background(), client.New(server.Client(), nil, 1, ""), dir, time.Minute, false, server.URL)
	if err != nil || !fromCache || len(got) != 1 || got[0].SlugID != 9 {
		t.Fatalf("LoadReports() = (%+v, %t, %v), want stale cache fallback", got, fromCache, err)
	}
}

func TestLoadReportsFromCacheMiss(t *testing.T) {
	got, ok, err := LoadReportsFromCache(t.TempDir())
	if err != nil || ok || got != nil {
		t.Fatalf("LoadReportsFromCache() = (%v, %t, %v), want nil cache miss", got, ok, err)
	}
}
