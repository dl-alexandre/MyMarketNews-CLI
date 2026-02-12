package query

import "testing"

func TestRequestURL(t *testing.T) {
	tests := []struct {
		name string
		req  Request
		want string
	}{
		{
			name: "section encoding with q and sort",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "2466",
				Section:    "Butter Prices and Sales",
				Q:          "report_date=08/05/2019:08/10/2019;class_description=STEER,HEIFER;published_date=2020-07-04 12:30:00",
				Sort:       "-published_date",
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/2466/Butter%20Prices%20and%20Sales?q=report_date%3D08%2F05%2F2019%3A08%2F10%2F2019%3Bclass_description%3DSTEER%2CHEIFER%3Bpublished_date%3D2020-07-04+12%3A30%3A00&sort=-published_date",
		},
		{
			name: "section encoding with punctuation",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "9999",
				Section:    "National Weekly Direct Slaughter Cattle - Negotiated",
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/9999/National%20Weekly%20Direct%20Slaughter%20Cattle%20-%20Negotiated",
		},
		{
			name: "section encoding with parentheses and comma",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "9998",
				Section:    "Lamb (Carcass), Weekly Summary",
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/9998/Lamb%20%28Carcass%29%2C%20Weekly%20Summary",
		},
		{
			name: "corrections and recent params",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "2466",
				Params: map[string]string{
					"correctionsOnly": "true",
					"anyChangesSince": "7/4/2020",
					"lastDays":        "5",
				},
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/2466?anyChangesSince=7%2F4%2F2020&correctionsOnly=true&lastDays=5",
		},
		{
			name: "combined params with q and email",
			req: Request{
				BaseURL:     "https://mpr.datamart.ams.usda.gov",
				APIVersion:  "v1.1",
				Endpoint:    "reports",
				SlugID:      "2451",
				AllSections: true,
				Q:           "report_date=2026-02-01:2026-02-10;class_description=STEER,HEIFER;item_desc=Chemical Lean,, Fresh 50%",
				Sort:        "-published_date",
				Params: map[string]string{
					"correctionsOnly": "true",
					"anyChangesSince": "2026-02-10",
					"sendEmail":       "true",
					"email":           "you@example.com",
				},
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/2451?allSections=true&anyChangesSince=2026-02-10&correctionsOnly=true&email=you%40example.com&q=report_date%3D2026-02-01%3A2026-02-10%3Bclass_description%3DSTEER%2CHEIFER%3Bitem_desc%3DChemical+Lean%2C%2C+Fresh+50%25&sendEmail=true&sort=-published_date",
		},
		{
			name: "published date with time only",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "1111",
				Q:          "published_date=2026-02-10 14:30:00",
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/1111?q=published_date%3D2026-02-10+14%3A30%3A00",
		},
		{
			name: "sort only no q",
			req: Request{
				BaseURL:    "https://mpr.datamart.ams.usda.gov",
				APIVersion: "v1.1",
				Endpoint:   "reports",
				SlugID:     "2222",
				Sort:       "-published_date",
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/2222?sort=-published_date",
		},
		{
			name: "email params with all sections",
			req: Request{
				BaseURL:     "https://mpr.datamart.ams.usda.gov",
				APIVersion:  "v1.1",
				Endpoint:    "reports",
				SlugID:      "2451",
				AllSections: true,
				Params: map[string]string{
					"sendEmail": "true",
					"email":     "you@example.com",
				},
			},
			want: "https://mpr.datamart.ams.usda.gov/services/v1.1/reports/2451?allSections=true&email=you%40example.com&sendEmail=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.req.URL()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected url\nwant: %s\n got: %s", tt.want, got)
			}
		})
	}
}
