package cli

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cbarber/fortyhours/internal/config"
	"github.com/cbarber/fortyhours/internal/productive"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	var email, sickEvent, ptoEvent, autofillSpec string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Discover your person id, absence events, and autofill defaults, and save them to config",
		Long: `init authenticates with PRODUCTIVE_API_KEY/PRODUCTIVE_ORG_ID, resolves your
Productive person id from your email, and helps you pick which absence
events mean "sick" and "pto" and which projects/hours autofill should use
by default. Flags allow running it non-interactively.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.APIToken == "" || cfg.OrgID == "" {
				return fmt.Errorf("set %s and %s before running init", config.EnvAPIToken, config.EnvOrgID)
			}
			client := productive.NewClient(cfg.APIToken, cfg.OrgID)
			ctx := cmd.Context()
			in := bufio.NewReader(cmd.InOrStdin())
			out := cmd.OutOrStdout()

			if email == "" {
				email = cfg.PersonEmail
			}
			if email == "" {
				email = prompt(in, out, "Your Productive email: ")
			}
			person, err := resolvePersonByEmail(ctx, client, email)
			if err != nil {
				return err
			}
			cfg.PersonEmail = email
			cfg.PersonID = person.ID
			fmt.Fprintf(out, "resolved person: %s %s (id=%s)\n", str(person.Attributes.FirstName), str(person.Attributes.LastName), person.ID)

			events, err := client.ListEvents(ctx, nil)
			if err != nil {
				return err
			}
			if sickEvent == "" {
				sickEvent = choose(in, out, "sick", events)
			}
			if ptoEvent == "" {
				ptoEvent = choose(in, out, "pto", events)
			}
			cfg.SickEvent = sickEvent
			cfg.PTOEvent = ptoEvent

			projects, err := client.ListProjects(ctx, nil)
			if err != nil {
				return err
			}
			if autofillSpec == "" {
				autofillSpec = promptAutofillSpec(in, out, projects)
			}
			autofill, err := resolveAutofillSpec(ctx, client, autofillSpec)
			if err != nil {
				return err
			}
			cfg.Autofill = autofill

			if err := cfg.Save(); err != nil {
				return err
			}
			path, _ := config.Path()
			fmt.Fprintf(out, "saved config to %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "your Productive email (skips the prompt)")
	cmd.Flags().StringVar(&sickEvent, "sick-event", "", "absence event name for sick days (skips the prompt)")
	cmd.Flags().StringVar(&ptoEvent, "pto-event", "", "absence event name for PTO (skips the prompt)")
	cmd.Flags().StringVar(&autofillSpec, "autofill", "", `autofill defaults as "project:hours,project:hours" (e.g. "dreamfi:7,internal:1"); skips the prompt`)
	return cmd
}

func prompt(in *bufio.Reader, out interface{ Write([]byte) (int, error) }, label string) string {
	fmt.Fprint(out, label)
	line, _ := in.ReadString('\n')
	return strings.TrimSpace(line)
}

func choose(in *bufio.Reader, out interface{ Write([]byte) (int, error) }, purpose string, events []productive.Record[productive.ResourceEvent]) string {
	fmt.Fprintf(out, "Available absence events:\n")
	for i, e := range events {
		fmt.Fprintf(out, "  %d) %s\n", i+1, str(e.Attributes.Name))
	}
	for {
		choice := prompt(in, out, fmt.Sprintf("Which one is %q? (number or name): ", purpose))
		if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(events) {
			return str(events[n-1].Attributes.Name)
		}
		for _, e := range events {
			if strings.EqualFold(str(e.Attributes.Name), choice) {
				return choice
			}
		}
		fmt.Fprintln(out, "not a valid choice, try again")
	}
}

func promptAutofillSpec(in *bufio.Reader, out interface{ Write([]byte) (int, error) }, projects []productive.Record[productive.ResourceProject]) string {
	fmt.Fprintf(out, "Active projects:\n")
	for _, p := range projects {
		if p.Attributes.ArchivedAt != nil {
			continue
		}
		fmt.Fprintf(out, "  - %s\n", str(p.Attributes.Name))
	}
	return prompt(in, out, `Autofill defaults as "project:hours,project:hours" (e.g. "dreamfi:7,internal:1"), or blank to skip: `)
}

// resolveAutofillSpec parses "project:hours,project:hours" and resolves
// each project to its single trackable service.
func resolveAutofillSpec(ctx context.Context, client *productive.Client, spec string) ([]config.AutofillProject, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}

	var out []config.AutofillProject
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, hoursStr, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("invalid autofill entry %q: expected project:hours", part)
		}
		hours, err := strconv.ParseFloat(strings.TrimSpace(hoursStr), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid hours in %q: %w", part, err)
		}
		project, err := resolveProject(ctx, client, strings.TrimSpace(name))
		if err != nil {
			return nil, err
		}
		service, err := resolveService(ctx, client, project.ID, "")
		if err != nil {
			return nil, fmt.Errorf("resolving service for project %q: %w", name, err)
		}
		out = append(out, config.AutofillProject{Project: str(project.Attributes.Name), ServiceID: service.ID, Hours: hours})
	}
	return out, nil
}
