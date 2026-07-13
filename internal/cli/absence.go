package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cbarber/fortyhours/internal/productive"
)

// deleteTimeEntriesInRange deletes every time entry the configured person
// has between start and end (inclusive), returning how many were removed.
//
// This implements the edge case where booking sick/PTO on a day that
// already has a timesheet should replace it: Productive doesn't reject
// overlapping bookings and time entries, so fortyhours enforces it itself.
func deleteTimeEntriesInRange(ctx context.Context, app *App, start, end time.Time) (int, error) {
	filter := productive.NewFilter().
		Eq("person_id", app.Config.PersonID).
		Op("date", "gte", ymd(start)).
		Op("date", "lte", ymd(end))

	entries, err := app.Client.ListTimeEntries(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("checking for existing time entries: %w", err)
	}
	for _, e := range entries {
		if err := app.Client.DeleteTimeEntry(ctx, e.ID); err != nil {
			return 0, fmt.Errorf("deleting existing time entry %s: %w", e.ID, err)
		}
	}
	return len(entries), nil
}

func ymd(t time.Time) string {
	return t.Format("2006-01-02")
}

// createAbsenceBooking books eventName (sick/PTO) for the configured person
// from start to end (inclusive), first deleting any conflicting time
// entries in that range.
func createAbsenceBooking(ctx context.Context, app *App, eventName string, start, end time.Time, hours float64, note string) (productive.Record[productive.ResourceBooking], int, error) {
	if eventName == "" {
		return productive.Record[productive.ResourceBooking]{}, 0, fmt.Errorf("no absence event configured; run `fortyhours init`")
	}
	if app.Config.PersonID == "" {
		return productive.Record[productive.ResourceBooking]{}, 0, fmt.Errorf("no person configured; run `fortyhours init`")
	}

	deleted, err := deleteTimeEntriesInRange(ctx, app, start, end)
	if err != nil {
		return productive.Record[productive.ResourceBooking]{}, 0, err
	}

	attrs, err := buildBookingAttrs(ctx, app, "", "", eventName, start, end, hours, note, false)
	if err != nil {
		return productive.Record[productive.ResourceBooking]{}, 0, err
	}
	booking, err := app.Client.CreateBooking(ctx, attrs)
	if err != nil {
		return productive.Record[productive.ResourceBooking]{}, 0, err
	}
	return booking, deleted, nil
}
