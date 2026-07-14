package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cbarber/fortyhours/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect fortyhours' resolved configuration",
	}
	cmd.AddCommand(newConfigShowCommand())
	return cmd
}

// configView is config.Config as displayed by `config show`: APIToken is
// replaced with a bool so the secret itself is never printed.
type configView struct {
	ConfigPath       string                   `json:"config_path"`
	APITokenSet      bool                     `json:"api_token_set"`
	OrgID            string                   `json:"organization_id"`
	PersonID         string                   `json:"person_id"`
	PersonEmail      string                   `json:"person_email"`
	SickEvent        string                   `json:"sick_event"`
	PTOEvent         string                   `json:"pto_event"`
	DailyGoalMinutes int                      `json:"daily_goal_minutes"`
	Autofill         []config.AutofillProject `json:"autofill"`
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the resolved config (file + env overrides), redacting the API token",
		Long: `show prints the same config autofill/timesheets would use: the config
file overlaid with PRODUCTIVE_API_KEY/PRODUCTIVE_ORG_ID/PRODUCTIVE_PERSON_ID
from the environment. The API token is never printed, only whether one is
set, so this is safe to run in CI logs to confirm secrets resolved.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			path, err := config.Path()
			if err != nil {
				return err
			}

			jsonOut, _ := cmd.Flags().GetBool("json")
			view := configView{
				ConfigPath:       path,
				APITokenSet:      cfg.APIToken != "",
				OrgID:            cfg.OrgID,
				PersonID:         cfg.PersonID,
				PersonEmail:      cfg.PersonEmail,
				SickEvent:        cfg.SickEvent,
				PTOEvent:         cfg.PTOEvent,
				DailyGoalMinutes: cfg.DailyGoalMinutes,
				Autofill:         cfg.Autofill,
			}
			if jsonOut {
				return printJSON(cmd.OutOrStdout(), view)
			}

			rows := [][]string{
				{"config_path", view.ConfigPath},
				{"api_token_set", strconv.FormatBool(view.APITokenSet)},
				{"organization_id", view.OrgID},
				{"person_id", view.PersonID},
				{"person_email", view.PersonEmail},
				{"sick_event", view.SickEvent},
				{"pto_event", view.PTOEvent},
				{"daily_goal_minutes", strconv.Itoa(view.DailyGoalMinutes)},
				{"autofill", formatAutofill(view.Autofill)},
			}
			return printTable(cmd.OutOrStdout(), []string{"FIELD", "VALUE"}, rows)
		},
	}
}

// formatAutofill renders cfg.Autofill as "project:hours:service" pairs,
// matching the --fill/--autofill flag syntax so it can be copy-pasted.
func formatAutofill(projects []config.AutofillProject) string {
	parts := make([]string, len(projects))
	for i, p := range projects {
		parts[i] = fmt.Sprintf("%s:%s:%s", p.Project, strconv.FormatFloat(p.Hours, 'f', -1, 64), p.ServiceID)
	}
	return strings.Join(parts, ",")
}
