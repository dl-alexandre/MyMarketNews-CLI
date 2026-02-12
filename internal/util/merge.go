package util

import (
	"encoding/json"
	"errors"
)

type mergeStats struct {
	TotalRows       int
	ReturnedRows    int
	UserAllowedRows int
}

func MergeResponses(bodies [][]byte) ([]byte, error) {
	if len(bodies) == 0 {
		return nil, errors.New("no responses to merge")
	}

	var merged map[string]any
	results := make([]any, 0)
	stats := mergeStats{}

	for _, body := range bodies {
		var obj map[string]any
		if err := json.Unmarshal(body, &obj); err != nil {
			return nil, err
		}

		res, ok := obj["results"].([]any)
		if !ok {
			return nil, errors.New("response missing results array")
		}
		results = append(results, res...)

		parsed := parseStats(obj["stats"])
		if parsed.UserAllowedRows > 0 && stats.UserAllowedRows == 0 {
			stats.UserAllowedRows = parsed.UserAllowedRows
		}
		if parsed.TotalRows > 0 {
			stats.TotalRows += parsed.TotalRows
		}
		stats.ReturnedRows += len(res)

		if merged == nil {
			merged = obj
		}
	}

	if merged == nil {
		return nil, errors.New("no mergeable responses")
	}

	merged["results"] = results
	if stats.TotalRows == 0 {
		stats.TotalRows = stats.ReturnedRows
	}
	merged["stats"] = map[string]any{
		"totalRows":       stats.TotalRows,
		"returnedRows":    stats.ReturnedRows,
		"userAllowedRows": stats.UserAllowedRows,
	}

	return json.Marshal(merged)
}

func parseStats(value any) mergeStats {
	stats := mergeStats{}
	obj, ok := value.(map[string]any)
	if !ok {
		return stats
	}
	stats.TotalRows = readNumber(obj["totalRows"])
	stats.ReturnedRows = readNumber(obj["returnedRows"])
	stats.UserAllowedRows = readNumber(obj["userAllowedRows"])
	return stats
}

func readNumber(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return int(n)
		}
	}
	return 0
}
