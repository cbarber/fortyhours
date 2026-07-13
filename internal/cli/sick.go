package cli

import (
	"fmt"

	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/spf13/cobra"
)

func newSickCommand() *cobra.Command {
	var hours float64
	var note string
	cmd := &cobra.Command{
		Use:   "sick [date]",
		Short: "Book a sick day (default: today)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			arg := "today"
			if len(args) > 0 {
				arg = args[0]
			}
			day, err := dates.Parse(arg)
			if err != nil {
				return err
			}

			booking, deleted, err := createAbsenceBooking(cmd.Context(), app, app.Config.SickEvent, day, day, hours, note)
			if err != nil {
				return err
			}
			if deleted > 0 {
				fmt.Fprintf(app.Out, "removed %d existing time entry(s) for %s\n", deleted, dates.Format(day))
			}
			fmt.Fprintf(app.Out, "booked sick day %s (booking %s)\n", dates.Format(day), booking.ID)
			return nil
		},
	}
	cmd.Flags().Float64Var(&hours, "hours", 8, "hours to book as sick")
	cmd.Flags().StringVar(&note, "note", "", "note for the booking")
	return cmd
}
