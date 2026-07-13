package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newTimesheetsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timesheets",
		Short: "Submit or unsubmit timesheets (the unit of time-entry approval)",
		Long: `A timesheet locks one person's time entries for one day, submitting them
for approval. Productive allows at most one timesheet per person per day,
and a timesheet can only be deleted (unsubmitted) before any of its time
entries have been approved.`,
	}
	cmd.AddCommand(
		newTimesheetsSubmitCommand(),
		newTimesheetsUnsubmitCommand(),
	)
	return cmd
}

func newTimesheetsSubmitCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "submit <day|week|last-week|month|YYYY-MM-DD>",
		Short: "Submit timesheets for every weekday in range, locking that day's time entries for approval",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if app.Config.PersonID == "" {
				return fmt.Errorf("no person configured; run `fortyhours init`")
			}

			start, end, err := dates.SubmitRange(args[0])
			if err != nil {
				return err
			}
			existing, err := timesheetsByDate(ctx, app, start, end)
			if err != nil {
				return err
			}
			personID, err := toIntID(app.Config.PersonID)
			if err != nil {
				return fmt.Errorf("invalid configured person_id: %w", err)
			}

			for _, day := range dates.Weekdays(start, end) {
				label := dates.Format(day)
				if _, ok := existing[label]; ok {
					fmt.Fprintf(app.Out, "%s: skipped (already submitted)\n", label)
					continue
				}
				if dryRun {
					fmt.Fprintf(app.Out, "%s: would submit\n", label)
					continue
				}
				d := toOpenAPIDate(day)
				if _, err := app.Client.CreateTimesheet(ctx, productive.ResourceTimesheet{Date: &d, PersonId: &personID}); err != nil {
					return fmt.Errorf("%s: submitting: %w", label, err)
				}
				fmt.Fprintf(app.Out, "%s: submitted\n", label)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be submitted without creating timesheets")
	return cmd
}

func newTimesheetsUnsubmitCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "unsubmit <day|week|last-week|month|YYYY-MM-DD>",
		Short: "Delete timesheets for every weekday in range, unlocking that day's time entries",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()
			if app.Config.PersonID == "" {
				return fmt.Errorf("no person configured; run `fortyhours init`")
			}

			start, end, err := dates.SubmitRange(args[0])
			if err != nil {
				return err
			}
			existing, err := timesheetsByDate(ctx, app, start, end)
			if err != nil {
				return err
			}

			for _, day := range dates.Weekdays(start, end) {
				label := dates.Format(day)
				id, ok := existing[label]
				if !ok {
					fmt.Fprintf(app.Out, "%s: skipped (not submitted)\n", label)
					continue
				}
				if dryRun {
					fmt.Fprintf(app.Out, "%s: would unsubmit\n", label)
					continue
				}
				if err := app.Client.DeleteTimesheet(ctx, id); err != nil {
					return fmt.Errorf("%s: unsubmitting: %w", label, err)
				}
				fmt.Fprintf(app.Out, "%s: unsubmitted\n", label)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be unsubmitted without deleting timesheets")
	return cmd
}

// timesheetsByDate returns the configured person's existing timesheet ids
// in [start, end], keyed by YYYY-MM-DD date.
func timesheetsByDate(ctx context.Context, app *App, start, end time.Time) (map[string]string, error) {
	filter := productive.NewFilter().
		Eq("person_id", app.Config.PersonID).
		Op("date", "gt_eq", dates.Format(start)).
		Op("date", "lt_eq", dates.Format(end))
	timesheets, err := app.Client.ListTimesheets(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("checking for existing timesheets: %w", err)
	}
	out := make(map[string]string, len(timesheets))
	for _, t := range timesheets {
		out[dateStr(t.Attributes.Date)] = t.ID
	}
	return out, nil
}
