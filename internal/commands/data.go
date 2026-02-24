package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"mpr/internal/config"
	"mpr/internal/query"
	"mpr/internal/util"

	"github.com/spf13/cobra"
)

type commonFlags struct {
	section     string
	allSections bool
	sort        string
	desc        bool
	where       []string
	between     []string
	inValues    []string
	reportDate  string
	pubDate     string
	reportYear  string
	reportMonth string
	limitDays   int
	chunkDays   int
	merge       bool
	format      string
}

func addCommonFlags(cmd *cobra.Command, defaults defaultFlags) *commonFlags {
	flags := &commonFlags{}
	cmd.Flags().StringVar(&flags.section, "section", defaults.section, "Report section name")
	cmd.Flags().BoolVar(&flags.allSections, "all-sections", false, "Request all sections")
	cmd.Flags().StringVar(&flags.sort, "sort", "", "Sort field")
	cmd.Flags().BoolVar(&flags.desc, "desc", false, "Sort descending")
	cmd.Flags().StringArrayVar(&flags.where, "where", nil, "Filter condition field=value")
	cmd.Flags().StringArrayVar(&flags.between, "between", nil, "Range filter field=start:end")
	cmd.Flags().StringArrayVar(&flags.inValues, "in", nil, "Multi-value filter field=v1,v2")
	cmd.Flags().StringVar(&flags.reportDate, "report-date", "", "Filter report_date (value or start:end)")
	cmd.Flags().StringVar(&flags.pubDate, "published-date", "", "Filter published_date (value or start:end)")
	cmd.Flags().StringVar(&flags.reportYear, "report-year", "", "Filter report_year (value or start:end)")
	cmd.Flags().StringVar(&flags.reportMonth, "report-month", "", "Filter report_month")
	cmd.Flags().IntVar(&flags.limitDays, "limit-days", defaults.limitDays, "Max allowed date range in days")
	cmd.Flags().IntVar(&flags.chunkDays, "chunk-days", defaults.chunkDays, "Split requests into N-day chunks")
	cmd.Flags().BoolVar(&flags.merge, "merge", false, "Merge chunked responses into single JSON")
	return flags
}

func addOutputFlags(cmd *cobra.Command, flags *commonFlags) {
	cmd.Flags().StringVar(&flags.format, "format", "json", "Output format: json, pretty, ndjson")
}

type defaultFlags struct {
	section   string
	limitDays int
	chunkDays int
}

func urlCmd() *cobra.Command {
	flags := &commonFlags{}
	extra := &extraParamsFlags{}

	cmd := &cobra.Command{
		Use:   "url <slug_id>",
		Short: "Print the composed API URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}

			if err := resolveSectionFlags(cmd, flags); err != nil {
				return err
			}

			inputs := filterInputs{
				where:         flags.where,
				between:       flags.between,
				inValues:      flags.inValues,
				reportDate:    flags.reportDate,
				publishedDate: flags.pubDate,
				reportYear:    flags.reportYear,
				reportMonth:   flags.reportMonth,
			}

			builder, ranges, err := buildQuery(inputs, nil)
			if err != nil {
				return err
			}

			if err := enforceRangeLimit(ranges, flags.limitDays); err != nil {
				return err
			}

			params, err := extra.buildParams()
			if err != nil {
				return err
			}

			req := query.Request{
				BaseURL:     cfg.BaseURL,
				APIVersion:  "v1.1",
				Endpoint:    "reports",
				SlugID:      args[0],
				Section:     strings.TrimSpace(flags.section),
				AllSections: flags.allSections,
				Q:           builder.String(),
				Sort:        sortValue(flags.sort, flags.desc),
				Params:      params,
			}

			url, err := req.URL()
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintln(os.Stdout, url)
			return nil
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "Summary", limitDays: 180, chunkDays: 0})
	addExtraFlags(cmd, extra)
	return cmd
}

func curlCmd() *cobra.Command {
	flags := &commonFlags{}
	extra := &extraParamsFlags{}

	cmd := &cobra.Command{
		Use:   "curl <slug_id>",
		Short: "Print an equivalent curl command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}

			if err := resolveSectionFlags(cmd, flags); err != nil {
				return err
			}

			inputs := filterInputs{
				where:         flags.where,
				between:       flags.between,
				inValues:      flags.inValues,
				reportDate:    flags.reportDate,
				publishedDate: flags.pubDate,
				reportYear:    flags.reportYear,
				reportMonth:   flags.reportMonth,
			}

			builder, ranges, err := buildQuery(inputs, nil)
			if err != nil {
				return err
			}

			if err := enforceRangeLimit(ranges, flags.limitDays); err != nil {
				return err
			}

			params, err := extra.buildParams()
			if err != nil {
				return err
			}

			req := query.Request{
				BaseURL:     cfg.BaseURL,
				APIVersion:  "v1.1",
				Endpoint:    "reports",
				SlugID:      args[0],
				Section:     strings.TrimSpace(flags.section),
				AllSections: flags.allSections,
				Q:           builder.String(),
				Sort:        sortValue(flags.sort, flags.desc),
				Params:      params,
			}
			url, err := req.URL()
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(os.Stdout, "curl -s '%s'\n", url)
			return nil
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "Summary", limitDays: 180, chunkDays: 0})
	addExtraFlags(cmd, extra)
	return cmd
}

func getCmd() *cobra.Command {
	flags := &commonFlags{}

	cmd := &cobra.Command{
		Use:   "get <slug_id>",
		Short: "Fetch report data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if err := resolveSectionFlags(cmd, flags); err != nil {
				return err
			}

			return fetchWithOptions(cmd.Context(), cfg, args[0], "v1.1", "reports", flags, map[string]string{})
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "Summary", limitDays: 180, chunkDays: 0})
	addOutputFlags(cmd, flags)
	return cmd
}

func correctionsCmd() *cobra.Command {
	flags := &commonFlags{}
	var since string

	cmd := &cobra.Command{
		Use:   "corrections <slug_id>",
		Short: "Fetch corrections for a report section",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if flags.section == "" {
				return errors.New("--section is required")
			}

			params := map[string]string{"correctionsOnly": "true"}
			if strings.TrimSpace(since) != "" {
				params["anyChangesSince"] = strings.TrimSpace(since)
			}

			return fetchWithOptions(cmd.Context(), cfg, args[0], "v1.1", "reports", flags, params)
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "", limitDays: 180, chunkDays: 0})
	_ = cmd.MarkFlagRequired("section")
	cmd.Flags().StringVar(&since, "since", "", "Only include corrections since date")
	addOutputFlags(cmd, flags)
	return cmd
}

func recentCmd() *cobra.Command {
	flags := &commonFlags{}
	var lastDays int
	var lastReports int

	cmd := &cobra.Command{
		Use:   "recent <slug_id>",
		Short: "Fetch recent report data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if flags.section == "" {
				return errors.New("--section is required")
			}
			if (lastDays == 0 && lastReports == 0) || (lastDays > 0 && lastReports > 0) {
				return errors.New("use either --last-days or --last-reports")
			}

			params := map[string]string{}
			if lastDays > 0 {
				params["lastDays"] = fmt.Sprintf("%d", lastDays)
			}
			if lastReports > 0 {
				params["lastReports"] = fmt.Sprintf("%d", lastReports)
			}

			return fetchWithOptions(cmd.Context(), cfg, args[0], "v1.1", "reports", flags, params)
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "", limitDays: 180, chunkDays: 0})
	_ = cmd.MarkFlagRequired("section")
	cmd.Flags().IntVar(&lastDays, "last-days", 0, "Include last N days")
	cmd.Flags().IntVar(&lastReports, "last-reports", 0, "Include last N reports")
	addOutputFlags(cmd, flags)
	return cmd
}

func emailCmd() *cobra.Command {
	flags := &commonFlags{}
	var email string

	cmd := &cobra.Command{
		Use:   "email <slug_id>",
		Short: "Request an emailed ZIP of CSV data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if email == "" {
				return errors.New("--to is required")
			}
			if err := resolveSectionFlags(cmd, flags); err != nil {
				return err
			}
			if flags.chunkDays > 0 {
				return errors.New("--chunk-days is not supported with email delivery")
			}

			params := map[string]string{
				"sendEmail": "true",
				"email":     email,
			}

			if err := enforceRangeLimitFromFlags(flags, 30); err != nil {
				return err
			}

			return fetchWithOptions(cmd.Context(), cfg, args[0], "v1.1", "reports", flags, params)
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "Summary", limitDays: 30, chunkDays: 0})
	cmd.Flags().StringVar(&email, "to", "", "Email address for ZIP")
	_ = cmd.MarkFlagRequired("to")
	addOutputFlags(cmd, flags)
	return cmd
}

func timeseriesCmd() *cobra.Command {
	flags := &commonFlags{}

	cmd := &cobra.Command{
		Use:   "timeseries <slug_id>",
		Short: "Fetch time series data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := readConfig(cmd.Flags())
			if err != nil {
				return err
			}
			if flags.allSections {
				return errors.New("--all-sections is not supported for timeseries")
			}

			return fetchWithOptions(cmd.Context(), cfg, args[0], "v1.2", "timeseries", flags, map[string]string{})
		},
	}
	cmd.ValidArgsFunction = slugCompletion
	_ = cmd.RegisterFlagCompletionFunc("section", sectionCompletion)

	flags = addCommonFlags(cmd, defaultFlags{section: "", limitDays: 180, chunkDays: 0})
	addOutputFlags(cmd, flags)
	return cmd
}

type extraParamsFlags struct {
	correctionsOnly bool
	anyChangesSince string
	lastDays        int
	lastReports     int
	sendEmail       bool
	email           string
}

func addExtraFlags(cmd *cobra.Command, extra *extraParamsFlags) {
	cmd.Flags().BoolVar(&extra.correctionsOnly, "corrections-only", false, "Only include corrections")
	cmd.Flags().StringVar(&extra.anyChangesSince, "any-changes-since", "", "Only include changes since date")
	cmd.Flags().IntVar(&extra.lastDays, "last-days", 0, "Include last N days")
	cmd.Flags().IntVar(&extra.lastReports, "last-reports", 0, "Include last N reports")
	cmd.Flags().BoolVar(&extra.sendEmail, "send-email", false, "Send results via email")
	cmd.Flags().StringVar(&extra.email, "email", "", "Email address for send-email")
}

func (e *extraParamsFlags) buildParams() (map[string]string, error) {
	params := map[string]string{}
	if e.correctionsOnly {
		params["correctionsOnly"] = "true"
	}
	if strings.TrimSpace(e.anyChangesSince) != "" {
		params["anyChangesSince"] = strings.TrimSpace(e.anyChangesSince)
	}
	if e.lastDays > 0 && e.lastReports > 0 {
		return nil, errors.New("use either --last-days or --last-reports")
	}
	if e.lastDays > 0 {
		params["lastDays"] = fmt.Sprintf("%d", e.lastDays)
	}
	if e.lastReports > 0 {
		params["lastReports"] = fmt.Sprintf("%d", e.lastReports)
	}
	if e.sendEmail || strings.TrimSpace(e.email) != "" {
		if strings.TrimSpace(e.email) == "" {
			return nil, errors.New("--email is required with --send-email")
		}
		params["sendEmail"] = "true"
		params["email"] = strings.TrimSpace(e.email)
	}
	return params, nil
}

func fetchWithOptions(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *commonFlags, params map[string]string) error {
	inputs := filterInputs{
		where:         flags.where,
		between:       flags.between,
		inValues:      flags.inValues,
		reportDate:    flags.reportDate,
		publishedDate: flags.pubDate,
		reportYear:    flags.reportYear,
		reportMonth:   flags.reportMonth,
	}

	format, err := util.ParseOutputFormat(flags.format)
	if err != nil {
		return err
	}

	builder, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}

	if flags.chunkDays > 0 {
		if format == util.FormatNDJSON && flags.merge {
			fmt.Fprintln(os.Stderr, "--merge ignored for ndjson output")
		}
		if format == util.FormatNDJSON {
			return fetchChunkedNDJSON(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
		}
		if flags.merge {
			return fetchChunkedMerged(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
		}
		return fetchChunked(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
	}

	if err := enforceRangeLimit(ranges, flags.limitDays); err != nil {
		return err
	}

	req := query.Request{
		BaseURL:     cfg.BaseURL,
		APIVersion:  version,
		Endpoint:    endpoint,
		SlugID:      slugID,
		Section:     strings.TrimSpace(flags.section),
		AllSections: flags.allSections,
		Q:           builder.String(),
		Sort:        sortValue(flags.sort, flags.desc),
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

func fetchChunked(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *commonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
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

	chunkDays := flags.chunkDays
	if chunkDays <= 0 {
		chunkDays = 30
	}
	if chunkDays > flags.limitDays && flags.limitDays > 0 {
		chunkDays = flags.limitDays
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
			Section:     strings.TrimSpace(flags.section),
			AllSections: flags.allSections,
			Q:           builder.String(),
			Sort:        sortValue(flags.sort, flags.desc),
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
				return errors.New("row limit reached; add more filters to narrow results")
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

func fetchChunkedMerged(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *commonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
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

	chunkDays := flags.chunkDays
	if chunkDays <= 0 {
		chunkDays = 30
	}
	if chunkDays > flags.limitDays && flags.limitDays > 0 {
		chunkDays = flags.limitDays
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
			Section:     strings.TrimSpace(flags.section),
			AllSections: flags.allSections,
			Q:           builder.String(),
			Sort:        sortValue(flags.sort, flags.desc),
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
				return errors.New("row limit reached; add more filters to narrow results")
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

func fetchChunkedNDJSON(ctx context.Context, cfg config.Config, slugID string, version string, endpoint string, flags *commonFlags, params map[string]string, inputs filterInputs, ranges []dateRange, format util.OutputFormat) error {
	return fetchChunked(ctx, cfg, slugID, version, endpoint, flags, params, inputs, ranges, format)
}

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

func resolveSectionFlags(cmd *cobra.Command, flags *commonFlags) error {
	if !flags.allSections {
		return nil
	}
	if cmd.Flags().Changed("section") {
		return errors.New("--section cannot be used with --all-sections")
	}
	flags.section = ""
	return nil
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

func enforceRangeLimitFromFlags(flags *commonFlags, maxDays int) error {
	inputs := filterInputs{
		where:         flags.where,
		between:       flags.between,
		inValues:      flags.inValues,
		reportDate:    flags.reportDate,
		publishedDate: flags.pubDate,
		reportYear:    flags.reportYear,
		reportMonth:   flags.reportMonth,
	}
	_, ranges, err := buildQuery(inputs, nil)
	if err != nil {
		return err
	}
	return enforceRangeLimit(ranges, maxDays)
}

func pickDateRange(ranges []dateRange) (dateRange, error) {
	var selected *dateRange
	for _, r := range ranges {
		field := strings.ToLower(r.Field)
		if strings.Contains(field, "date") {
			if selected != nil {
				return dateRange{}, errors.New("--chunk-days supports only one ranged date filter")
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
	return dateRange{}, errors.New("--chunk-days requires a date range filter")
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
