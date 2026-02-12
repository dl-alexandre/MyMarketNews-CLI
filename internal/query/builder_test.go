package query

import "testing"

func TestBuilderQComposition(t *testing.T) {
	builder := NewBuilder()
	builder.AddBetween("report_date", "08/05/2019", "08/10/2019")
	builder.AddIn("class_description", []string{"STEER", "HEIFER"})
	builder.AddWhere("item_desc", "Chemical Lean, Fresh 50%")

	got := builder.String()
	want := "report_date=08/05/2019:08/10/2019;class_description=STEER,HEIFER;item_desc=Chemical Lean,, Fresh 50%"
	if got != want {
		t.Fatalf("unexpected q string\nwant: %s\n got: %s", want, got)
	}
}
