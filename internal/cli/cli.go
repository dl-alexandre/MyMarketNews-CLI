package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"mpr/internal/cache"
	"mpr/internal/client"
	"mpr/internal/config"
	"mpr/internal/models"
	"mpr/internal/query"
	"mpr/internal/util"
)

// CLI is the main command-line interface structure using Kong
type CLI struct {
	Globals

	Reports     ReportsCmd     `cmd:"" help:"Discover available reports"`
	Get         GetCmd         `cmd:"" help:"Fetch report data"`
	Corrections CorrectionsCmd `cmd:"" help:"Fetch corrections for a report section"`
	Recent      RecentCmd      `cmd:"" help:"Fetch recent report data"`
	Email       EmailCmd       `cmd:"" help:"Request an emailed ZIP of CSV data"`
	Timeseries  TimeseriesCmd  `cmd:"" help:"Fetch time series data"`
	URL         URLCmd         `cmd:"" help:"Print the composed API URL"`
	Curl        CurlCmd        `cmd:"" help:"Print an equivalent curl command"`
	Completion  CompletionCmd  `cmd:"" help:"Generate shell completion script"`
}

// Globals contains global flags available to all commands
type Globals struct {
	BaseURL  string        `help:"API base URL" default:"https://mpr.datamart.ams.usda.gov" env:"MPR_BASE_URL"`
	Timeout  time.Duration `help:"HTTP request timeout" default:"30s" env:"MPR_TIMEOUT"`
	RPS      float64       `help:"Requests per second limit" default:"1.0" env:"MPR_RPS"`
	CacheTTL time.Duration `help:"Reports cache TTL" default:"12h" env:"MPR_CACHE_TTL"`
	CacheDir string        `help:"Cache directory (defaults to OS user cache dir)" env:"MPR_CACHE_DIR"`
	Debug    bool          `help:"Enable debug output" env:"MPR_DEBUG"`
}

func (g *Globals) AfterApply() error {
	return nil
}

func (g *Globals) ToConfig() config.Config {
	return config.Config{
		BaseURL:  g.BaseURL,
		Timeout:  g.Timeout,
		RPS:      g.RPS,
		CacheTTL: g.CacheTTL,
		CacheDir: g.CacheDir,
	}
}

// ReportsCmd handles report discovery commands
type ReportsCmd struct {
	List   ReportsListCmd   `cmd:"" help:"List published reports"`
	Search ReportsSearchCmd `cmd:"" help:"Search report titles"`
	Show   ReportsShowCmd   `cmd:"" help:"Show metadata for a report"`
}

type ReportsListCmd struct {
	JSON       bool   `help:"Output JSON"`
	Table      bool   `help:"Output table"`
	Market     string `help:"Filter by market"`
	Office     string `help:"Filter by office"`
	MarketType string `help:"Filter by market type" name:"market-type"`
	Refresh    bool   `help:"Refresh cached reports"`
}

func (c *ReportsListCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()

	reports, _, err := cache.LoadReports(context.Background(), newClient(cfg), cfg.CacheDir, cfg.CacheTTL, c.Refresh, cfg.BaseURL)
	if err != nil {
		return err
	}

	filtered := filterReports(reports, c.Market, c.Office, c.MarketType)
	if c.JSON && c.Table {
		return fmt.Errorf("use either --json or --table")
	}

	if c.JSON {
		data, err := json.MarshalIndent(filtered, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if !c.Table && !c.JSON {
		c.Table = true
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "slug_id\tslug_name\treport_title\tpublished_date")
	for _, report := range filtered {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", report.SlugID, report.SlugName, report.ReportTitle, report.PublishedDate)
	}
	return w.Flush()
}

type ReportsSearchCmd struct {
	Term    string `arg:"" help:"Search text"`
	Regex   bool   `help:"Treat search text as regex"`
	Refresh bool   `help:"Refresh cached reports"`
}

func (c *ReportsSearchCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()

	reports, _, err := cache.LoadReports(context.Background(), newClient(cfg), cfg.CacheDir, cfg.CacheTTL, c.Refresh, cfg.BaseURL)
	if err != nil {
		return err
	}

	matches, err := searchReports(reports, c.Term, c.Regex)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(matches, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

type ReportsShowCmd struct {
	SlugID  string `arg:"" help:"Report slug ID"`
	Refresh bool   `help:"Refresh cached reports"`
}

func (c *ReportsShowCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()

	reports, _, err := cache.LoadReports(context.Background(), newClient(cfg), cfg.CacheDir, cfg.CacheTTL, c.Refresh, cfg.BaseURL)
	if err != nil {
		return err
	}

	slugID := strings.TrimSpace(c.SlugID)
	report, ok := findReport(reports, slugID)
	if !ok {
		return fmt.Errorf("slug_id not found: %s", slugID)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// Common flags for data commands
type CommonFlags struct {
	Section     string   `help:"Report section name" default:"Summary"`
	AllSections bool     `help:"Request all sections"`
	Sort        string   `help:"Sort field"`
	Desc        bool     `help:"Sort descending"`
	Where       []string `help:"Filter condition field=value"`
	Between     []string `help:"Range filter field=start:end"`
	In          []string `help:"Multi-value filter field=v1,v2"`
	ReportDate  string   `help:"Filter report_date (value or start:end)"`
	PubDate     string   `help:"Filter published_date (value or start:end)"`
	ReportYear  string   `help:"Filter report_year (value or start:end)"`
	ReportMonth string   `help:"Filter report_month"`
	LimitDays   int      `help:"Max allowed date range in days" default:"180"`
	ChunkDays   int      `help:"Split requests into N-day chunks"`
	Merge       bool     `help:"Merge chunked responses into single JSON"`
	Format      string   `help:"Output format: json, pretty, ndjson" default:"json"`
}

func (f *CommonFlags) ToFilterInputs() filterInputs {
	return filterInputs{
		where:         f.Where,
		between:       f.Between,
		inValues:      f.In,
		reportDate:    f.ReportDate,
		publishedDate: f.PubDate,
		reportYear:    f.ReportYear,
		reportMonth:   f.ReportMonth,
	}
}

func (f *CommonFlags) ValidateSection() error {
	if !f.AllSections {
		return nil
	}
	if f.Section != "" && f.Section != "Summary" {
		return fmt.Errorf("--section cannot be used with --all-sections")
	}
	f.Section = ""
	return nil
}

// Data commands
type GetCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	CommonFlags
}

func (c *GetCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if err := c.ValidateSection(); err != nil {
		return err
	}
	return fetchWithOptions(context.Background(), cfg, c.SlugID, "v1.1", "reports", &c.CommonFlags, map[string]string{})
}

type CorrectionsCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	Since  string `help:"Only include corrections since date"`
	CommonFlags
}

func (c *CorrectionsCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if c.Section == "" {
		return fmt.Errorf("--section is required")
	}

	params := map[string]string{"correctionsOnly": "true"}
	if strings.TrimSpace(c.Since) != "" {
		params["anyChangesSince"] = strings.TrimSpace(c.Since)
	}

	if err := enforceRangeLimitFromFlags(c.ToFilterInputs(), c.LimitDays); err != nil {
		return err
	}

	return fetchWithOptions(context.Background(), cfg, c.SlugID, "v1.1", "reports", &c.CommonFlags, params)
}

type RecentCmd struct {
	SlugID      string `arg:"" help:"Report slug ID"`
	LastDays    int    `help:"Include last N days"`
	LastReports int    `help:"Include last N reports"`
	CommonFlags
}

func (c *RecentCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if c.Section == "" {
		return fmt.Errorf("--section is required")
	}
	if (c.LastDays == 0 && c.LastReports == 0) || (c.LastDays > 0 && c.LastReports > 0) {
		return fmt.Errorf("use either --last-days or --last-reports")
	}

	params := map[string]string{}
	if c.LastDays > 0 {
		params["lastDays"] = fmt.Sprintf("%d", c.LastDays)
	}
	if c.LastReports > 0 {
		params["lastReports"] = fmt.Sprintf("%d", c.LastReports)
	}

	return fetchWithOptions(context.Background(), cfg, c.SlugID, "v1.1", "reports", &c.CommonFlags, params)
}

type EmailCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	To     string `help:"Email address for ZIP" required:""`
	CommonFlags
}

func (c *EmailCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if err := c.ValidateSection(); err != nil {
		return err
	}
	if c.ChunkDays > 0 {
		return fmt.Errorf("--chunk-days is not supported with email delivery")
	}

	params := map[string]string{
		"sendEmail": "true",
		"email":     c.To,
	}

	if err := enforceRangeLimitFromFlags(c.ToFilterInputs(), 30); err != nil {
		return err
	}

	return fetchWithOptions(context.Background(), cfg, c.SlugID, "v1.1", "reports", &c.CommonFlags, params)
}

type TimeseriesCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	CommonFlags
}

func (c *TimeseriesCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if c.AllSections {
		return fmt.Errorf("--all-sections is not supported for timeseries")
	}

	return fetchWithOptions(context.Background(), cfg, c.SlugID, "v1.2", "timeseries", &c.CommonFlags, map[string]string{})
}

type URLCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	CommonFlags
}

func (c *URLCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if err := c.ValidateSection(); err != nil {
		return err
	}

	inputs := c.ToFilterInputs()
	builder, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}

	if err := enforceRangeLimit(ranges, c.LimitDays); err != nil {
		return err
	}

	req := query.Request{
		BaseURL:     cfg.BaseURL,
		APIVersion:  "v1.1",
		Endpoint:    "reports",
		SlugID:      c.SlugID,
		Section:     strings.TrimSpace(c.Section),
		AllSections: c.AllSections,
		Q:           builder.String(),
		Sort:        sortValue(c.Sort, c.Desc),
	}

	url, err := req.URL()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(os.Stdout, url)
	return nil
}

type CurlCmd struct {
	SlugID string `arg:"" help:"Report slug ID"`
	CommonFlags
}

func (c *CurlCmd) Run(globals *Globals) error {
	cfg := globals.ToConfig()
	if err := c.ValidateSection(); err != nil {
		return err
	}

	inputs := c.ToFilterInputs()
	builder, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}

	if err := enforceRangeLimit(ranges, c.LimitDays); err != nil {
		return err
	}

	req := query.Request{
		BaseURL:     cfg.BaseURL,
		APIVersion:  "v1.1",
		Endpoint:    "reports",
		SlugID:      c.SlugID,
		Section:     strings.TrimSpace(c.Section),
		AllSections: c.AllSections,
		Q:           builder.String(),
		Sort:        sortValue(c.Sort, c.Desc),
	}
	url, err := req.URL()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(os.Stdout, "curl -s '%s'\n", url)
	return nil
}

type CompletionCmd struct {
	Shell string `arg:"" help:"Shell: bash, zsh, fish, powershell"`
}

func (c *CompletionCmd) Run() error {
	fmt.Printf("# %s completion for mpr\n", c.Shell)
	return nil
}

// Helper functions
func sortValue(field string, desc bool) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	if desc {
		return "-" + field
	}
	return field
}

func enforceRangeLimit(ranges []dateRange, maxDays int) error {
	if maxDays <= 0 {
		return nil
	}
	for _, r := range ranges {
		start, ok := util.ParseDate(r.Start)
		if !ok {
			continue
		}
		end, ok := util.ParseDate(r.End)
		if !ok {
			continue
		}
		days := util.DaysBetween(start, end)
		if days > maxDays {
			return fmt.Errorf("date range for %s exceeds %d days", r.Field, maxDays)
		}
	}
	return nil
}

func enforceRangeLimitFromFlags(inputs filterInputs, maxDays int) error {
	_, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}
	return enforceRangeLimit(ranges, maxDays)
}

func newClient(cfg config.Config) *client.Client {
	httpClient := &http.Client{Timeout: cfg.Timeout}
	limiter := client.NewRateLimiter(cfg.RPS)
	return client.New(httpClient, limiter, 6, "mpr-cli/0.1")
}

// Import filter functions from filters.go
func buildQuery(inputs filterInputs, overrides map[string]dateRange) (*query.Builder, []dateRange, error) {
	builder := query.NewBuilder()
	ranges := make([]dateRange, 0)
	usedOverrides := map[string]bool{}

	for _, raw := range inputs.where {
		field, value, ok := splitFieldValue(raw)
		if !ok {
			return nil, nil, fmt.Errorf("invalid --where value: %s", raw)
		}
		builder.AddWhere(field, value)
	}

	for _, raw := range inputs.between {
		field, start, end, ok := splitBetween(raw)
		if !ok {
			return nil, nil, fmt.Errorf("invalid --between value: %s", raw)
		}
		if override, ok := getOverride(overrides, usedOverrides, field); ok {
			builder.AddBetween(field, override.Start, override.End)
			ranges = append(ranges, override)
			continue
		}
		builder.AddBetween(field, start, end)
		ranges = append(ranges, dateRange{Field: field, Start: start, End: end})
	}

	for _, raw := range inputs.inValues {
		field, value, ok := splitFieldValue(raw)
		if !ok {
			return nil, nil, fmt.Errorf("invalid --in value: %s", raw)
		}
		values := strings.Split(value, ",")
		builder.AddIn(field, values)
	}

	addConvenience(builder, &ranges, "report_date", inputs.reportDate, overrides, usedOverrides)
	addConvenience(builder, &ranges, "published_date", inputs.publishedDate, overrides, usedOverrides)
	addConvenience(builder, &ranges, "report_year", inputs.reportYear, overrides, usedOverrides)
	addConvenience(builder, &ranges, "report_month", inputs.reportMonth, overrides, usedOverrides)

	return builder, ranges, nil
}

func addConvenience(builder *query.Builder, ranges *[]dateRange, field, value string, overrides map[string]dateRange, usedOverrides map[string]bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if override, ok := getOverride(overrides, usedOverrides, field); ok {
		builder.AddBetween(field, override.Start, override.End)
		*ranges = append(*ranges, override)
		return
	}
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) == 2 {
			builder.AddBetween(field, parts[0], parts[1])
			*ranges = append(*ranges, dateRange{Field: field, Start: parts[0], End: parts[1]})
			return
		}
	}
	builder.AddWhere(field, value)
}

func getOverride(overrides map[string]dateRange, used map[string]bool, field string) (dateRange, bool) {
	if overrides == nil {
		return dateRange{}, false
	}
	if used[field] {
		return dateRange{}, false
	}
	override, ok := overrides[field]
	if !ok {
		return dateRange{}, false
	}
	used[field] = true
	return override, true
}

func splitFieldValue(input string) (string, string, bool) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	field := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if field == "" || value == "" {
		return "", "", false
	}
	return field, value, true
}

func splitBetween(input string) (string, string, string, bool) {
	field, value, ok := splitFieldValue(input)
	if !ok {
		return "", "", "", false
	}
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if start == "" || end == "" {
		return "", "", "", false
	}
	return field, start, end, true
}

// Data command helpers
func fetchWithOptions(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *CommonFlags, params map[string]string) error {
	inputs := flags.ToFilterInputs()

	format, err := util.ParseOutputFormat(flags.Format)
	if err != nil {
		return err
	}

	builder, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}

	if flags.ChunkDays > 0 {
		if format == util.FormatNDJSON && flags.Merge {
			fmt.Fprintln(os.Stderr, "--merge ignored for ndjson output")
		}
		if format == util.FormatNDJSON {
			return fetchChunkedNDJSON(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
		}
		if flags.Merge {
			return fetchChunkedMerged(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
		}
		return fetchChunked(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
	}

	if err := enforceRangeLimit(ranges, flags.LimitDays); err != nil {
		return err
	}

	req := query.Request{
		BaseURL:     cfg.BaseURL,
		APIVersion:  version,
		Endpoint:    endpoint,
		SlugID:      slugID,
		Section:     strings.TrimSpace(flags.Section),
		AllSections: flags.AllSections,
		Q:           builder.String(),
		Sort:        sortValue(flags.Sort, flags.Desc),
		Params:      params,
	}

	url, err := req.URL()
	if err != nil {
		return err
	}

	c := newClient(cfg)
	body, status, err := c.Get(ctx, url)
	if err != nil {
		return formatHTTPError(err, status)
	}

	return util.WriteFormatted(os.Stdout, body, format)
}

func fetchChunked(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *CommonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
	selected, err := pickDateRange(ranges)
	if err != nil {
		return err
	}

	startTime, layout, ok := util.ParseDateWithLayout(selected.Start)
	if !ok {
		return fmt.Errorf("invalid start date: %s", selected.Start)
	}
	endTime, _, ok := util.ParseDateWithLayout(selected.End)
	if !ok {
		return fmt.Errorf("invalid end date: %s", selected.End)
	}

	chunkDays := flags.ChunkDays
	if chunkDays <= 0 {
		chunkDays = 30
	}
	if chunkDays > flags.LimitDays && flags.LimitDays > 0 {
		chunkDays = flags.LimitDays
	}

	c := newClient(cfg)
	rangesToFetch := splitRange(startTime, endTime, chunkDays)
	for i := 0; i < len(rangesToFetch); {
		r := rangesToFetch[i]
		override := map[string]dateRange{
			selected.Field: {
				Field: selected.Field,
				Start: r.Start.Format(layout),
				End:   r.End.Format(layout),
			},
		}

		builder, _, err := buildQuery(inputs, override)
		if err != nil {
			return err
		}

		req := query.Request{
			BaseURL:     cfg.BaseURL,
			APIVersion:  version,
			Endpoint:    endpoint,
			SlugID:      slugID,
			Section:     strings.TrimSpace(flags.Section),
			AllSections: flags.AllSections,
			Q:           builder.String(),
			Sort:        sortValue(flags.Sort, flags.Desc),
			Params:      params,
		}

		url, err := req.URL()
		if err != nil {
			return err
		}

		body, status, err := c.Get(ctx, url)
		if err != nil {
			return formatHTTPError(err, status)
		}

		if truncated := util.IsResponseTruncated(body); truncated {
			days := util.DaysBetween(r.Start, r.End)
			if days <= 1 {
				return fmt.Errorf("row limit reached; add more filters to narrow results")
			}

			left, right := splitRangeOnce(r.Start, r.End)
			replacement := []rangeChunk{left, right}
			rangesToFetch = append(rangesToFetch[:i], append(replacement, rangesToFetch[i+1:]...)...)
			continue
		}

		if err := util.WriteFormatted(os.Stdout, body, format); err != nil {
			return err
		}
		i++
	}

	return nil
}

func fetchChunkedMerged(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *CommonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
	selected, err := pickDateRange(ranges)
	if err != nil {
		return err
	}

	startTime, layout, ok := util.ParseDateWithLayout(selected.Start)
	if !ok {
		return fmt.Errorf("invalid start date: %s", selected.Start)
	}
	endTime, _, ok := util.ParseDateWithLayout(selected.End)
	if !ok {
		return fmt.Errorf("invalid end date: %s", selected.End)
	}

	chunkDays := flags.ChunkDays
	if chunkDays <= 0 {
		chunkDays = 30
	}
	if chunkDays > flags.LimitDays && flags.LimitDays > 0 {
		chunkDays = flags.LimitDays
	}

	c := newClient(cfg)
	rangesToFetch := splitRange(startTime, endTime, chunkDays)
	bodies := make([][]byte, 0)
	for i := 0; i < len(rangesToFetch); {
		r := rangesToFetch[i]
		override := map[string]dateRange{
			selected.Field: {
				Field: selected.Field,
				Start: r.Start.Format(layout),
				End:   r.End.Format(layout),
			},
		}

		builder, _, err := buildQuery(inputs, override)
		if err != nil {
			return err
		}

		req := query.Request{
			BaseURL:     cfg.BaseURL,
			APIVersion:  version,
			Endpoint:    endpoint,
			SlugID:      slugID,
			Section:     strings.TrimSpace(flags.Section),
			AllSections: flags.AllSections,
			Q:           builder.String(),
			Sort:        sortValue(flags.Sort, flags.Desc),
			Params:      params,
		}

		url, err := req.URL()
		if err != nil {
			return err
		}

		body, status, err := c.Get(ctx, url)
		if err != nil {
			return formatHTTPError(err, status)
		}

		if truncated := util.IsResponseTruncated(body); truncated {
			days := util.DaysBetween(r.Start, r.End)
			if days <= 1 {
				return fmt.Errorf("row limit reached; add more filters to narrow results")
			}

			left, right := splitRangeOnce(r.Start, r.End)
			replacement := []rangeChunk{left, right}
			rangesToFetch = append(rangesToFetch[:i], append(replacement, rangesToFetch[i+1:]...)...)
			continue
		}

		bodies = append(bodies, body)
		i++
	}

	merged, err := util.MergeResponses(bodies)
	if err != nil {
		return err
	}
	return util.WriteFormatted(os.Stdout, merged, format)
}

func fetchChunkedNDJSON(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *CommonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
	return fetchChunked(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
}

func pickDateRange(ranges []dateRange) (dateRange, error) {
	var selected *dateRange
	for _, r := range ranges {
		field := strings.ToLower(r.Field)
		if strings.Contains(field, "date") {
			if selected != nil {
				return dateRange{}, fmt.Errorf("--chunk-days supports only one ranged date filter")
			}
			copy := r
			selected = &copy
		}
	}
	if selected != nil {
		return *selected, nil
	}
	if len(ranges) > 0 {
		return ranges[0], nil
	}
	return dateRange{}, fmt.Errorf("--chunk-days requires a date range filter")
}

type rangeChunk struct {
	Start time.Time
	End   time.Time
}

func splitRange(start, end time.Time, chunkDays int) []rangeChunk {
	if end.Before(start) {
		start, end = end, start
	}
	chunks := []rangeChunk{}
	current := start
	for !current.After(end) {
		nextEnd := current.AddDate(0, 0, chunkDays-1)
		if nextEnd.After(end) {
			nextEnd = end
		}
		chunks = append(chunks, rangeChunk{Start: current, End: nextEnd})
		current = nextEnd.AddDate(0, 0, 1)
	}
	return chunks
}

func splitRangeOnce(start, end time.Time) (rangeChunk, rangeChunk) {
	totalDays := util.DaysBetween(start, end)
	leftDays := totalDays / 2
	if leftDays < 1 {
		leftDays = 1
	}
	leftEnd := start.AddDate(0, 0, leftDays-1)
	rightStart := leftEnd.AddDate(0, 0, 1)
	return rangeChunk{Start: start, End: leftEnd}, rangeChunk{Start: rightStart, End: end}
}

func formatHTTPError(err error, status int) error {
	if status == 429 || status == 403 {
		return fmt.Errorf("%w; possible temporary IP block, reduce request rate or contact AMS support", err)
	}
	return err
}

// Reports helper functions
func filterReports(reports []models.ReportItem, market, office, marketType string) []models.ReportItem {
	market = strings.TrimSpace(strings.ToLower(market))
	office = strings.TrimSpace(strings.ToLower(office))
	marketType = strings.TrimSpace(strings.ToLower(marketType))

	filtered := make([]models.ReportItem, 0, len(reports))
	for _, report := range reports {
		if market != "" && !containsFold(report.Markets, market) {
			continue
		}
		if office != "" && !containsFold(report.Offices, office) {
			continue
		}
		if marketType != "" && !containsFold(report.MarketTypes, marketType) {
			continue
		}
		filtered = append(filtered, report)
	}
	return filtered
}

func searchReports(reports []models.ReportItem, term string, useRegex bool) ([]models.ReportItem, error) {
	matches := make([]models.ReportItem, 0)
	if useRegex {
		re, err := regexp.Compile(term)
		if err != nil {
			return nil, err
		}
		for _, report := range reports {
			if re.MatchString(report.ReportTitle) || re.MatchString(report.SlugName) {
				matches = append(matches, report)
			}
		}
		return matches, nil
	}

	term = strings.ToLower(term)
	for _, report := range reports {
		if strings.Contains(strings.ToLower(report.ReportTitle), term) || strings.Contains(strings.ToLower(report.SlugName), term) {
			matches = append(matches, report)
		}
	}
	return matches, nil
}

func findReport(reports []models.ReportItem, slugID string) (models.ReportItem, bool) {
	for _, report := range reports {
		if fmt.Sprintf("%d", report.SlugID) == slugID {
			return report, true
		}
	}
	return models.ReportItem{}, false
}

func containsFold(values []string, needle string) bool {
	for _, v := range values {
		if strings.ToLower(v) == needle {
			return true
		}
	}
	return false
}

// filterInputs mirrors the internal type
type filterInputs struct {
	where         []string
	between       []string
	inValues      []string
	reportDate    string
	publishedDate string
	reportYear    string
	reportMonth   string
}

// dateRange mirrors the internal type
type dateRange struct {
	Field string
	Start string
	End   string
}
