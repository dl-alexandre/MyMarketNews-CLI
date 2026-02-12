package query

import (
	"net/url"
	"path"
	"strings"
)

type Request struct {
	BaseURL     string
	APIVersion  string
	Endpoint    string
	SlugID      string
	Section     string
	AllSections bool
	Q           string
	Sort        string
	Params      map[string]string
}

func (r Request) URL() (string, error) {
	u, err := url.Parse(r.BaseURL)
	if err != nil {
		return "", err
	}

	segments := []string{"services", r.APIVersion, r.Endpoint}
	if r.SlugID != "" {
		segments = append(segments, r.SlugID)
	}
	if r.Section != "" {
		segments = append(segments, r.Section)
	}

	escapedSegments := make([]string, 0, len(segments))
	for _, seg := range segments {
		escapedSegments = append(escapedSegments, url.PathEscape(seg))
	}

	u.Path = path.Join(segments...)
	u.RawPath = path.Join(escapedSegments...)

	q := u.Query()
	if r.Q != "" {
		q.Set("q", r.Q)
	}
	if r.Sort != "" {
		q.Set("sort", r.Sort)
	}
	if r.AllSections {
		q.Set("allSections", "true")
	}
	for k, v := range r.Params {
		if strings.TrimSpace(v) == "" {
			continue
		}
		q.Set(k, v)
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}
