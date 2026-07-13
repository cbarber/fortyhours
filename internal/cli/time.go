package cli

import (
	"fmt"

	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newTimeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "time",
		Short: "List, get, create, update, and delete time entries",
	}
	cmd.AddCommand(
		newTimeListCommand(),
		newTimeGetCommand(),
		newTimeCreateCommand(),
		newTimeUpdateCommand(),
		newTimeDeleteCommand(),
	)
	return cmd
}

func printTimeEntries(app *App, entries []productive.Record[productive.ResourceTimeEntry]) error {
	if app.JSON {
		return printJSON(app.Out, entries)
	}
	rows := make([][]string, len(entries))
	for i, e := range entries {
		rows[i] = []string{e.ID, dateStr(e.Attributes.Date), hoursStr(e.Attributes.Time), str(e.Attributes.Note), intStr(e.Attributes.ServiceId)}
	}
	return printTable(app.Out, []string{"ID", "DATE", "HOURS", "NOTE", "SERVICE_ID"}, rows)
}

func newTimeListCommand() *cobra.Command {
	var personID, from, to, projectName, serviceFlag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List time entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if personID == "" {
				personID = app.Config.PersonID
			}
			filter := productive.NewFilter()
			if personID != "" {
				filter.Eq("person_id", personID)
			}
			if from != "" {
				d, err := dates.Parse(from)
				if err != nil {
					return err
				}
				filter.Op("date", "gte", dates.Format(d))
			}
			if to != "" {
				d, err := dates.Parse(to)
				if err != nil {
					return err
				}
				filter.Op("date", "lte", dates.Format(d))
			}
			if projectName != "" {
				project, err := resolveProject(ctx, app.Client, projectName)
				if err != nil {
					return err
				}
				filter.Eq("project_id", project.ID)
			}
			if serviceFlag != "" {
				filter.Eq("service_id", serviceFlag)
			}

			entries, err := app.Client.ListTimeEntries(ctx, filter)
			if err != nil {
				return err
			}
			return printTimeEntries(app, entries)
		},
	}
	cmd.Flags().StringVar(&personID, "person", "", "filter by person id (default: configured person)")
	cmd.Flags().StringVar(&from, "from", "", "only entries on or after this date")
	cmd.Flags().StringVar(&to, "to", "", "only entries on or before this date")
	cmd.Flags().StringVar(&projectName, "project", "", "filter by project name")
	cmd.Flags().StringVar(&serviceFlag, "service", "", "filter by service id")
	return cmd
}

func newTimeGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			entry, err := app.Client.GetTimeEntry(cmd.Context(), id)
			if err != nil {
				return err
			}
			return printTimeEntries(app, []productive.Record[productive.ResourceTimeEntry]{entry})
		},
	}
}

func newTimeCreateCommand() *cobra.Command {
	var projectName, serviceFlag, date, note string
	var hours float64
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a time entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if app.Config.PersonID == "" {
				return fmt.Errorf("no person configured; run `fortyhours init`")
			}
			service, err := resolveServiceForCommand(ctx, app, projectName, serviceFlag)
			if err != nil {
				return err
			}
			d, err := dates.Parse(date)
			if err != nil {
				return err
			}

			entry, err := createTimeEntry(ctx, app, service.ID, d, hours, note)
			if err != nil {
				return err
			}
			return printTimeEntries(app, []productive.Record[productive.ResourceTimeEntry]{entry})
		},
	}
	cmd.Flags().StringVar(&projectName, "project", "", "project name (used to resolve the service, if --service is omitted)")
	cmd.Flags().StringVar(&serviceFlag, "service", "", "service id or name to track time against")
	cmd.Flags().StringVar(&date, "date", "today", "date to log time for (YYYY-MM-DD, today, tomorrow, yesterday)")
	cmd.Flags().Float64Var(&hours, "hours", 0, "hours worked (required)")
	cmd.Flags().StringVar(&note, "note", "", "note for the time entry")
	cmd.MarkFlagRequired("hours")
	return cmd
}

func newTimeUpdateCommand() *cobra.Command {
	var hours float64
	var hoursSet bool
	var note string
	var noteSet bool
	var date string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			id, err := parseID(args[0])
			if err != nil {
				return err
			}

			var attrs productive.ResourceTimeEntry
			if cmd.Flags().Changed("hours") {
				hoursSet = true
			}
			if cmd.Flags().Changed("note") {
				noteSet = true
			}
			if hoursSet {
				minutes := int(hours * 60)
				attrs.Time = &minutes
			}
			if noteSet {
				attrs.Note = &note
			}
			if date != "" {
				d, err := dates.Parse(date)
				if err != nil {
					return err
				}
				openAPIDate := toOpenAPIDate(d)
				attrs.Date = &openAPIDate
			}

			entry, err := app.Client.UpdateTimeEntry(cmd.Context(), id, attrs)
			if err != nil {
				return err
			}
			return printTimeEntries(app, []productive.Record[productive.ResourceTimeEntry]{entry})
		},
	}
	cmd.Flags().Float64Var(&hours, "hours", 0, "new hours worked")
	cmd.Flags().StringVar(&note, "note", "", "new note")
	cmd.Flags().StringVar(&date, "date", "", "new date (YYYY-MM-DD, today, tomorrow, yesterday)")
	return cmd
}

func newTimeDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a time entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			if err := app.Client.DeleteTimeEntry(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "deleted time entry %s\n", id)
			return nil
		},
	}
}
