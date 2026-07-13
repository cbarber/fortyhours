package cli

import (
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X ...cli.Version=...".
var Version = "dev"

// NewRootCommand builds the fortyhours root command with every subcommand
// wired in.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "fortyhours",
		Short:         "Fill out a Productive.io timesheet and time off without the clicking",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().Bool("json", false, "output machine-readable JSON instead of a table")
	root.PersistentFlags().Bool("debug", false, "print Productive API request/response details to stderr (or set FORTYHOURS_DEBUG=1)")

	root.AddCommand(
		newInitCommand(),
		newProjectsCommand(),
		newServicesCommand(),
		newTimeCommand(),
		newBookingsCommand(),
		newSickCommand(),
		newPTOCommand(),
		newAutofillCommand(),
		newTimesheetsCommand(),
	)
	return root
}
