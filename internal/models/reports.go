package models

type ReportItem struct {
	SlugID        int      `json:"slug_id"`
	SlugName      string   `json:"slug_name"`
	ReportTitle   string   `json:"report_title"`
	PublishedDate string   `json:"published_date"`
	Markets       []string `json:"markets"`
	MarketTypes   []string `json:"market_types"`
	Offices       []string `json:"offices"`
	SectionNames  []string `json:"sectionNames"`
}
