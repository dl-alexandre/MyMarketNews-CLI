package query

import (
	"fmt"
	"strings"
)

type Builder struct {
	parts []string
}

func NewBuilder() *Builder {
	return &Builder{parts: []string{}}
}

func (b *Builder) AddRaw(part string) {
	if strings.TrimSpace(part) == "" {
		return
	}
	b.parts = append(b.parts, part)
}

func (b *Builder) AddWhere(field, value string) {
	if field == "" || value == "" {
		return
	}
	if strings.Contains(value, ",") {
		value = strings.ReplaceAll(value, ",", ",,")
	}
	b.parts = append(b.parts, fmt.Sprintf("%s=%s", field, value))
}

func (b *Builder) AddBetween(field, start, end string) {
	if field == "" || start == "" || end == "" {
		return
	}
	b.parts = append(b.parts, fmt.Sprintf("%s=%s:%s", field, start, end))
}

func (b *Builder) AddIn(field string, values []string) {
	if field == "" || len(values) == 0 {
		return
	}
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	if len(cleaned) == 0 {
		return
	}
	b.parts = append(b.parts, fmt.Sprintf("%s=%s", field, strings.Join(cleaned, ",")))
}

func (b *Builder) String() string {
	if len(b.parts) == 0 {
		return ""
	}
	return strings.Join(b.parts, ";")
}
