// Package dates provides the small set of date helpers shared by the sick,
// pto and autofill commands: parsing "today"-style shorthand, and computing
// the weekday range autofill should fill.
package dates

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

// Parse parses "today", "tomorrow", "yesterday", or a YYYY-MM-DD date,
// returning a date-only time.Time (midnight, local time).
func Parse(s string) (time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch s {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	}
	t, err := time.ParseInLocation(layout, s, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD, today, tomorrow, or yesterday", s)
	}
	return t, nil
}

// Format renders a date-only time.Time as YYYY-MM-DD.
func Format(t time.Time) string {
	return t.Format(layout)
}

// Weekdays returns every Monday-Friday date in [start, end], inclusive.
func Weekdays(start, end time.Time) []time.Time {
	var out []time.Time
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		out = append(out, d)
	}
	return out
}

// AutofillRange resolves autofill's range argument into a concrete [start,
// end] date span:
//
//   - "day":   today only
//   - "week":  Monday-Friday of the current week
//   - "month": the 1st through the last day of the current month
//   - any other value: parsed as an upper-bound date (today..date)
func AutofillRange(arg string) (start, end time.Time, err error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch arg {
	case "day":
		return today, today, nil
	case "week":
		monday := today.AddDate(0, 0, -weekdayOffset(today))
		friday := monday.AddDate(0, 0, 4)
		return monday, friday, nil
	case "month":
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		last := first.AddDate(0, 1, -1)
		return first, last, nil
	default:
		upper, err := Parse(arg)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid range %q: expected day, week, month, or an upper-bound date: %w", arg, err)
		}
		if upper.Before(today) {
			return time.Time{}, time.Time{}, fmt.Errorf("upper-bound date %s is before today (%s)", Format(upper), Format(today))
		}
		return today, upper, nil
	}
}

// weekdayOffset returns how many days to subtract from t to reach the
// Monday of its week (0 if t is already Monday).
func weekdayOffset(t time.Time) int {
	switch t.Weekday() {
	case time.Sunday:
		return 6
	default:
		return int(t.Weekday()) - int(time.Monday)
	}
}

// SubmitRange resolves timesheets submit/unsubmit's range argument into a
// concrete [start, end] date span:
//
//   - "day":        today only
//   - "week":       Monday-Friday of the current week
//   - "last-week":  Monday-Friday of the previous week
//   - "month":      the 1st through the last day of the current month
//   - any other value: parsed as a single date (that day only)
//
// Unlike AutofillRange, past dates are allowed: timesheets are submitted
// for time already logged, typically after the fact.
func SubmitRange(arg string) (start, end time.Time, err error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch arg {
	case "day":
		return today, today, nil
	case "week":
		monday := today.AddDate(0, 0, -weekdayOffset(today))
		return monday, monday.AddDate(0, 0, 4), nil
	case "last-week":
		monday := today.AddDate(0, 0, -weekdayOffset(today)-7)
		return monday, monday.AddDate(0, 0, 4), nil
	case "month":
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
		return first, first.AddDate(0, 1, -1), nil
	default:
		d, err := Parse(arg)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid range %q: expected day, week, last-week, month, or a date: %w", arg, err)
		}
		return d, d, nil
	}
}
