package util

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type OutputFormat string

const (
	FormatJSON   OutputFormat = "json"
	FormatPretty OutputFormat = "pretty"
	FormatNDJSON OutputFormat = "ndjson"
)

func ParseOutputFormat(value string) (OutputFormat, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == string(FormatJSON) {
		return FormatJSON, nil
	}
	if value == string(FormatPretty) {
		return FormatPretty, nil
	}
	if value == string(FormatNDJSON) {
		return FormatNDJSON, nil
	}
	return "", fmt.Errorf("unsupported format: %s", value)
}

func WriteFormatted(w io.Writer, body []byte, format OutputFormat) error {
	switch format {
	case FormatJSON:
		if _, err := w.Write(body); err != nil {
			return err
		}
		return ensureNewline(w, body)
	case FormatPretty:
		var out bytes.Buffer
		if err := json.Indent(&out, body, "", "  "); err != nil {
			return err
		}
		if _, err := w.Write(out.Bytes()); err != nil {
			return err
		}
		return ensureNewline(w, out.Bytes())
	case FormatNDJSON:
		return writeNDJSON(w, body)
	default:
		return errors.New("unknown output format")
	}
}

func writeNDJSON(w io.Writer, body []byte) error {
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return err
	}
	results, ok := obj["results"].([]any)
	if !ok {
		return errors.New("response missing results array")
	}
	for _, item := range results {
		line, err := json.Marshal(item)
		if err != nil {
			return err
		}
		if _, err := w.Write(line); err != nil {
			return err
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}

func ensureNewline(w io.Writer, body []byte) error {
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, err := w.Write([]byte("\n"))
		return err
	}
	return nil
}
