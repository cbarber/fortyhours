package cli

import (
	"fmt"

	"github.com/cbarber/fortyhours/internal/dates"
	"github.com/spf13/cobra"
)

func newPTOCommand() *cobra.Command {
	var hours float64
	var note string
	cmd := &cobra.Command{
		Use:   "pto <from> [to]",
		Short: "Book a PTO range (default: a single day if [to] is omitted)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			from, err := dates.Parse(args[0])
			if err != nil {
				return err
			}
			to := from
			if len(args) == 2 {
				to, err = dates.Parse(args[1])
				if err != nil {
					return err
				}
			}
			if to.Before(from) {
				return fmt.Errorf("end date %s is before start date %s", dates.Format(to), dates.Format(from))
			}

			booking, deleted, err := createAbsenceBooking(cmd.Context(), app, app.Config.PTOEvent, from, to, hours, note)
			if err != nil {
				return err
			}
			if deleted > 0 {
				fmt.Fprintf(app.Out, "removed %d existing time entry(s) between %s and %s\n", deleted, dates.Format(from), dates.Format(to))
			}
			fmt.Fprintf(app.Out, "booked PTO %s to %s (booking %s)\n", dates.Format(from), dates.Format(to), booking.ID)
			return nil
		},
	}
	cmd.Flags().Float64Var(&hours, "hours", 8, "hours per day to book as PTO")
	cmd.Flags().StringVar(&note, "note", "", "note for the booking")
	return cmd
}
