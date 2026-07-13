package cli

import (
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newServicesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Work with Productive services (what time is tracked against within a project)",
	}
	cmd.AddCommand(newServicesListCommand())
	return cmd
}

func newServicesListCommand() *cobra.Command {
	var projectName string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List services, optionally scoped to one project",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(cmd)
			if err != nil {
				return err
			}
			ctx := cmd.Context()

			var filter *productive.Filter
			if projectName != "" {
				project, err := resolveProject(ctx, app.Client, projectName)
				if err != nil {
					return err
				}
				filter = productive.NewFilter().Eq("project_id", project.ID)
			}

			services, err := app.Client.ListServices(ctx, filter)
			if err != nil {
				return err
			}

			if app.JSON {
				return printJSON(app.Out, services)
			}

			rows := make([][]string, len(services))
			for i, s := range services {
				rows[i] = []string{s.ID, str(s.Attributes.Name), intStr(s.Attributes.ProjectId), boolStr(s.Attributes.ForTracking)}
			}
			return printTable(app.Out, []string{"ID", "NAME", "PROJECT_ID", "TRACKABLE"}, rows)
		},
	}
	cmd.Flags().StringVar(&projectName, "project", "", "only list services under this project (by name)")
	return cmd
}
