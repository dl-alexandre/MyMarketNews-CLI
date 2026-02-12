package commands

import (
	"fmt"
	"strings"

	"mpr/internal/query"
)

type filterInputs struct {
	where         []string
	between       []string
	inValues      []string
	reportDate    string
	publishedDate string
	reportYear    string
	reportMonth   string
}

type dateRange struct {
	Field string
	Start string
	End   string
}

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
