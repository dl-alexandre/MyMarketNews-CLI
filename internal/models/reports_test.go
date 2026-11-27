package models

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReportItemJSONRoundTrip(t *testing.T) {
	want := ReportItem{
		SlugID: 42, SlugName: "weekly-cattle", ReportTitle: "Weekly Cattle Summary",
		PublishedDate: "2026-11-25", Markets: []string{"Oklahoma", "Texas"},
		MarketTypes: []string{"Cattle"}, Offices: []string{"Southwest"}, SectionNames: []string{"Summary", "Prices"},
	}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got ReportItem
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestReportItemUsesAPIJSONFieldNames(t *testing.T) {
	var got ReportItem
	if err := json.Unmarshal([]byte(`{"slug_id":7,"sectionNames":["Summary"]}`), &got); err != nil {
		t.Fatal(err)
	}
	if got.SlugID != 7 || len(got.SectionNames) != 1 || got.SectionNames[0] != "Summary" {
		t.Fatalf("unexpected decoded report: %+v", got)
	}
}
