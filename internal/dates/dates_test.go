package dates

import "testing"

func TestWeekdays(t *testing.T) {
	start, _ := Parse("2024-03-01") // Friday
	end, _ := Parse("2024-03-10")   // Sunday
	got := Weekdays(start, end)
	want := []string{"2024-03-01", "2024-03-04", "2024-03-05", "2024-03-06", "2024-03-07", "2024-03-08"}
	if len(got) != len(want) {
		t.Fatalf("Weekdays() returned %d dates, want %d", len(got), len(want))
	}
	for i, d := range got {
		if Format(d) != want[i] {
			t.Errorf("Weekdays()[%d] = %s, want %s", i, Format(d), want[i])
		}
	}
}

func TestParse(t *testing.T) {
	if _, err := Parse("today"); err != nil {
		t.Errorf("Parse(today): %v", err)
	}
	if _, err := Parse("2024-03-01"); err != nil {
		t.Errorf("Parse(date): %v", err)
	}
	if _, err := Parse("not-a-date"); err == nil {
		t.Error("Parse(not-a-date): expected error, got nil")
	}
}

func TestAutofillRangeKeywords(t *testing.T) {
	start, end, err := AutofillRange("week")
	if err != nil {
		t.Fatalf("AutofillRange(week): %v", err)
	}
	if start.Weekday().String() != "Monday" {
		t.Errorf("week start = %s, want Monday", start.Weekday())
	}
	if end.Weekday().String() != "Friday" {
		t.Errorf("week end = %s, want Friday", end.Weekday())
	}

	start, end, err = AutofillRange("month")
	if err != nil {
		t.Fatalf("AutofillRange(month): %v", err)
	}
	if start.Day() != 1 {
		t.Errorf("month start day = %d, want 1", start.Day())
	}
	if end.AddDate(0, 0, 1).Day() != 1 {
		t.Errorf("month end = %s is not the last day of its month", Format(end))
	}

	start, end, err = AutofillRange("day")
	if err != nil {
		t.Fatalf("AutofillRange(day): %v", err)
	}
	if !start.Equal(end) {
		t.Errorf("day range = [%s, %s], want a single day", Format(start), Format(end))
	}
}

func TestAutofillRangeUpperBoundDate(t *testing.T) {
	_, end, err := AutofillRange("2099-12-31")
	if err != nil {
		t.Fatalf("AutofillRange(date): %v", err)
	}
	if Format(end) != "2099-12-31" {
		t.Errorf("end = %s, want 2099-12-31", Format(end))
	}
}

func TestAutofillRangeRejectsPastUpperBound(t *testing.T) {
	if _, _, err := AutofillRange("2000-01-01"); err == nil {
		t.Error("expected error for an upper-bound date in the past")
	}
}
