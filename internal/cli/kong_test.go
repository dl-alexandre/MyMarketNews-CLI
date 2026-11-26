package cli

import (
	"io"
	"testing"
	"time"

	"github.com/alecthomas/kong"
)

func TestKongReportsListCommandAndFlags(t *testing.T) {
	var app CLI
	parser, err := kong.New(&app, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parser.Parse([]string{"reports", "list", "--json", "--market", "Dairy", "--office", "West", "--market-type", "Retail", "--refresh"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !app.Reports.List.JSON || !app.Reports.List.Refresh || app.Reports.List.Market != "Dairy" || app.Reports.List.Office != "West" || app.Reports.List.MarketType != "Retail" {
		t.Fatalf("parsed report list options incorrectly: %+v", app.Reports.List)
	}
}

func TestKongGetCommandAndGlobalFlags(t *testing.T) {
	var app CLI
	parser, err := kong.New(&app, kong.Writers(io.Discard, io.Discard))
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"--base-url", "https://api.example.test", "--timeout", "5s", "--cache-ttl", "2h", "get", "123", "--all-sections", "--where", "market=Texas", "--format", "pretty"}
	if _, err := parser.Parse(args); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if app.BaseURL != "https://api.example.test" || app.Timeout != 5*time.Second || app.CacheTTL != 2*time.Hour {
		t.Fatalf("parsed global flags incorrectly: %+v", app.Globals)
	}
	if app.Get.SlugID != "123" || !app.Get.AllSections || app.Get.Format != "pretty" || len(app.Get.Where) != 1 || app.Get.Where[0] != "market=Texas" {
		t.Fatalf("parsed get options incorrectly: %+v", app.Get)
	}
}

func TestCommonFlagsRejectConflictingSectionOptions(t *testing.T) {
	flags := CommonFlags{Section: "Prices", AllSections: true}
	if err := flags.ValidateSection(); err == nil {
		t.Fatal("ValidateSection() accepted --section with --all-sections")
	}
	flags.Section = "Summary"
	if err := flags.ValidateSection(); err != nil {
		t.Fatalf("ValidateSection() error = %v", err)
	}
	if flags.Section != "" {
		t.Fatalf("Section = %q, want empty for all sections", flags.Section)
	}
}
