package cli

import (
	"github.com/spf13/cobra"
)

func newProjectsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Work with Productive projects",
	}
	cmd.AddCommand(newProjectsListCommand())
	return cmd
}

func newProjectsListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			projects, err := app.Client.ListProjects(ctx, nil)
			if err != nil {
				return err
			}

			if app.JSON {
				return printJSON(app.Out, projects)
			}

			rows := make([][]string, len(projects))
			for i, p := range projects {
				rows[i] = []string{p.ID, str(p.Attributes.Name), archivedLabel(p.Attributes.ArchivedAt != nil)}
			}
			return printTable(app.Out, []string{"ID", "NAME", "ARCHIVED"}, rows)
		},
	}
}

func archivedLabel(archived bool) string {
	if archived {
		return "yes"
	}
	return "no"
}
