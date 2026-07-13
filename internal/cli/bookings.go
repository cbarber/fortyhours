package cli

import (
	"fmt"

	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newBookingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bookings",
		Short: "List, get, create, update, and delete bookings (scheduled work or absences)",
	}
	cmd.AddCommand(
		newBookingsListCommand(),
		newBookingsGetCommand(),
		newBookingsCreateCommand(),
		newBookingsUpdateCommand(),
		newBookingsDeleteCommand(),
	)
	return cmd
}

func printBookings(app *App, bookings []productive.Record[productive.ResourceBooking]) error {
	if app.JSON {
		return printJSON(app.Out, bookings)
	}
	rows := make([][]string, len(bookings))
	for i, b := range bookings {
		rows[i] = []string{
			b.ID,
			dateStr(b.Attributes.StartedOn),
			dateStr(b.Attributes.EndedOn),
			hoursStr(b.Attributes.Time),
			intStr(b.Attributes.EventId),
			intStr(b.Attributes.ServiceId),
			str(b.Attributes.Note),
		}
	}
	return printTable(app.Out, []string{"ID", "START", "END", "HOURS/DAY", "EVENT_ID", "SERVICE_ID", "NOTE"}, rows)
}

func newBookingsListCommand() *cobra.Command {
	var personID, from, to string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List bookings",
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
				filter.Op("started_on", "gt_eq", dates.Format(d))
			}
			if to != "" {
				d, err := dates.Parse(to)
				if err != nil {
					return err
				}
				filter.Op("ended_on", "lt_eq", dates.Format(d))
			}

			bookings, err := app.Client.ListBookings(ctx, filter)
			if err != nil {
				return err
			}
			return printBookings(app, bookings)
		},
	}
	cmd.Flags().StringVar(&personID, "person", "", "filter by person id (default: configured person)")
	cmd.Flags().StringVar(&from, "from", "", "only bookings starting on or after this date")
	cmd.Flags().StringVar(&to, "to", "", "only bookings ending on or before this date")
	return cmd
}

func newBookingsGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single booking",
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
			booking, err := app.Client.GetBooking(cmd.Context(), id)
			if err != nil {
				return err
			}
			return printBookings(app, []productive.Record[productive.ResourceBooking]{booking})
		},
	}
}

func newBookingsCreateCommand() *cobra.Command {
	var projectName, serviceFlag, eventName, from, to, note string
	var hours float64
	var draft bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a booking (scheduled work, or an absence when --event is set)",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			if app.Config.PersonID == "" {
				return fmt.Errorf("no person configured; run `fortyhours init`")
			}
			startedOn, err := dates.Parse(from)
			if err != nil {
				return err
			}
			endedOn := startedOn
			if to != "" {
				endedOn, err = dates.Parse(to)
				if err != nil {
					return err
				}
			}

			attrs, err := buildBookingAttrs(ctx, app, projectName, serviceFlag, eventName, startedOn, endedOn, hours, note, draft)
			if err != nil {
				return err
			}

			booking, err := app.Client.CreateBooking(ctx, attrs)
			if err != nil {
				return err
			}
			return printBookings(app, []productive.Record[productive.ResourceBooking]{booking})
		},
	}
	cmd.Flags().StringVar(&projectName, "project", "", "project name (work bookings; used to resolve the service)")
	cmd.Flags().StringVar(&serviceFlag, "service", "", "service id or name (work bookings)")
	cmd.Flags().StringVar(&eventName, "event", "", "absence event category name (absence bookings, e.g. Sick, PTO)")
	cmd.Flags().StringVar(&from, "from", "today", "start date (YYYY-MM-DD, today, tomorrow, yesterday)")
	cmd.Flags().StringVar(&to, "to", "", "end date (defaults to --from for a single day)")
	cmd.Flags().Float64Var(&hours, "hours", 8, "scheduled hours per day")
	cmd.Flags().StringVar(&note, "note", "", "note for the booking")
	cmd.Flags().BoolVar(&draft, "draft", false, "create as a tentative (draft) booking")
	return cmd
}

func newBookingsUpdateCommand() *cobra.Command {
	var hours float64
	var note string
	var from, to string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a booking",
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

			var attrs productive.ResourceBooking
			if cmd.Flags().Changed("hours") {
				minutes := int(hours * 60)
				attrs.Time = &minutes
			}
			if cmd.Flags().Changed("note") {
				attrs.Note = &note
			}
			if from != "" {
				d, err := dates.Parse(from)
				if err != nil {
					return err
				}
				od := toOpenAPIDate(d)
				attrs.StartedOn = &od
			}
			if to != "" {
				d, err := dates.Parse(to)
				if err != nil {
					return err
				}
				od := toOpenAPIDate(d)
				attrs.EndedOn = &od
			}

			booking, err := app.Client.UpdateBooking(cmd.Context(), id, attrs)
			if err != nil {
				return err
			}
			return printBookings(app, []productive.Record[productive.ResourceBooking]{booking})
		},
	}
	cmd.Flags().Float64Var(&hours, "hours", 0, "new scheduled hours per day")
	cmd.Flags().StringVar(&note, "note", "", "new note")
	cmd.Flags().StringVar(&from, "from", "", "new start date")
	cmd.Flags().StringVar(&to, "to", "", "new end date")
	return cmd
}

func newBookingsDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a booking",
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
			if err := app.Client.DeleteBooking(cmd.Context(), id); err != nil {
				return err
			}
			fmt.Fprintf(app.Out, "deleted booking %s\n", id)
			return nil
		},
	}
}
