package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cbarber/fortyhours/internal/config"
	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newAutofillCommand() *cobra.Command {
	var fillSpec string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "autofill <day|week|month|YYYY-MM-DD>",
		Short: "Fill in missing weekday time entries with the configured project defaults",
		Long: `autofill logs the configured default hours (see "fortyhours init") against
every Monday-Friday in the given range. It skips weekends, days that
already have time entries, and days covered by a sick/PTO booking. Run
the sick/pto/time commands first for exceptions, then autofill for
everything else.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			start, end, err := dates.AutofillRange(args[0])
			if err != nil {
				return err
			}

			plan := app.Config.Autofill
			if fillSpec != "" {
				// nil disables interactive service picking: --fill is meant
				// for non-interactive/scripted overrides, so an ambiguous
				// service must be spelled out as "project:hours:service".
				plan, err = resolveAutofillSpec(ctx, app.Client, fillSpec, nil, nil)
				if err != nil {
					return err
				}
			}
			if len(plan) == 0 {
				return fmt.Errorf("no autofill defaults configured; run `fortyhours init` or pass --fill")
			}
			if app.Config.PersonID == "" {
				return fmt.Errorf("no person configured; run `fortyhours init`")
			}

			absences, err := absenceBookingsInRange(ctx, app, start, end)
			if err != nil {
				return err
			}
			filledDates, err := datesWithTimeEntries(ctx, app, start, end)
			if err != nil {
				return err
			}

			goalMinutes := app.Config.DailyGoalMinutes
			if goalMinutes == 0 {
				goalMinutes = config.DefaultDailyGoalMinutes
			}

			for _, day := range dates.Weekdays(start, end) {
				label := dates.Format(day)

				if coversDate(absences, day) {
					fmt.Fprintf(app.Out, "%s: skipped (absence booked)\n", label)
					continue
				}
				if filledDates[label] {
					fmt.Fprintf(app.Out, "%s: skipped (already has time entries)\n", label)
					continue
				}

				var totalMinutes int
				for _, p := range plan {
					totalMinutes += int(p.Hours * 60)
					if dryRun {
						continue
					}
					if _, err := createTimeEntry(ctx, app, p.ServiceID, day, p.Hours, "autofill"); err != nil {
						return fmt.Errorf("%s: creating time entry for %s: %w", label, p.Project, err)
					}
				}

				verb := "filled"
				if dryRun {
					verb = "would fill"
				}
				fmt.Fprintf(app.Out, "%s: %s %s\n", label, verb, planSummary(plan))
				if totalMinutes != goalMinutes {
					fmt.Fprintf(app.Out, "%s: warning: totals %s, expected %s\n", label, hoursStr(&totalMinutes), hoursStr(&goalMinutes))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&fillSpec, "fill", "", `override the configured autofill defaults, as "project:hours,project:hours"`)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be filled without creating time entries")
	return cmd
}

func planSummary(plan []config.AutofillProject) string {
	out := ""
	for i, p := range plan {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%s %gh", p.Project, p.Hours)
	}
	return out
}

// absenceBookingsInRange returns every absence (EventId != nil) booking for
// the configured person overlapping [start, end].
func absenceBookingsInRange(ctx context.Context, app *App, start, end time.Time) ([]productive.Record[productive.ResourceBooking], error) {
	filter := productive.NewFilter().
		Eq("person_id", app.Config.PersonID).
		Op("started_on", "lt_eq", ymd(end)).
		Op("ended_on", "gt_eq", ymd(start))
	bookings, err := app.Client.ListBookings(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("checking for existing absence bookings: %w", err)
	}
	var absences []productive.Record[productive.ResourceBooking]
	for _, b := range bookings {
		if b.Attributes.EventId != nil {
			absences = append(absences, b)
		}
	}
	return absences, nil
}

// coversDate reports whether any booking in bookings spans day. Comparisons
// are done on the YYYY-MM-DD string, not time.Time instants: Productive
// dates parse as UTC midnight while day is built from the local "today",
// and comparing those as instants misclassifies dates near a UTC offset
// boundary.
func coversDate(bookings []productive.Record[productive.ResourceBooking], day time.Time) bool {
	d := ymd(day)
	for _, b := range bookings {
		if b.Attributes.StartedOn == nil || b.Attributes.EndedOn == nil {
			continue
		}
		if ymd(b.Attributes.StartedOn.Time) <= d && d <= ymd(b.Attributes.EndedOn.Time) {
			return true
		}
	}
	return false
}

// datesWithTimeEntries returns the set of YYYY-MM-DD dates (formatted via
// dates.Format) that already have at least one time entry for the
// configured person in [start, end].
func datesWithTimeEntries(ctx context.Context, app *App, start, end time.Time) (map[string]bool, error) {
	filter := productive.NewFilter().
		Eq("person_id", app.Config.PersonID).
		Op("date", "gt_eq", ymd(start)).
		Op("date", "lt_eq", ymd(end))
	entries, err := app.Client.ListTimeEntries(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("checking for existing time entries: %w", err)
	}
	filled := map[string]bool{}
	for _, e := range entries {
		if e.Attributes.Date != nil {
			filled[e.Attributes.Date.String()] = true
		}
	}
	return filled, nil
}
