package cli

import (
	"bufio"
	"context"
	"errors"
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
			client := newProductiveClient(cmd, cfg.APIToken, cfg.OrgID)
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
			autofill, err := resolveAutofillSpec(ctx, client, autofillSpec, in, out)
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
	cmd.Flags().StringVar(&autofillSpec, "autofill", "", `autofill defaults as "project:hours[:service],project:hours[:service]" (e.g. "dreamfi:7,internal:1"); skips the prompt`)
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

// chooseService prompts the user to pick one of candidates by number or id,
// used when a project has more than one trackable service.
func chooseService(in *bufio.Reader, out interface{ Write([]byte) (int, error) }, projectName string, candidates []productive.Record[productive.ResourceService]) (productive.Record[productive.ResourceService], error) {
	fmt.Fprintf(out, "Multiple trackable services found for %q:\n", projectName)
	for i, s := range candidates {
		fmt.Fprintf(out, "  %d) %s (id=%s)\n", i+1, str(s.Attributes.Name), s.ID)
	}
	for {
		choice := prompt(in, out, "Which service? (number or id): ")
		if n, err := strconv.Atoi(choice); err == nil && n >= 1 && n <= len(candidates) {
			return candidates[n-1], nil
		}
		for _, s := range candidates {
			if s.ID == choice {
				return s, nil
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
	return prompt(in, out, `Autofill defaults as "project:hours[:service],project:hours[:service]" (e.g. "dreamfi:7,internal:1"), or blank to skip: `)
}

// resolveAutofillSpec parses "project:hours" or "project:hours:service"
// (comma-separated), resolving each project to a service: its single
// trackable one, the given service id/name when specified, or (when in is
// non-nil) one chosen interactively when a project has more than one
// trackable service (common when a project rolls services over across
// budget periods). Pass a nil in to disable prompting and instead fail with
// the list of candidates, e.g. for non-interactive callers like --fill.
func resolveAutofillSpec(ctx context.Context, client *productive.Client, spec string, in *bufio.Reader, w interface{ Write([]byte) (int, error) }) ([]config.AutofillProject, error) {
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
		fields := strings.SplitN(part, ":", 3)
		if len(fields) < 2 {
			return nil, fmt.Errorf("invalid autofill entry %q: expected project:hours or project:hours:service", part)
		}
		name := strings.TrimSpace(fields[0])
		hours, err := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid hours in %q: %w", part, err)
		}
		var serviceFlag string
		if len(fields) == 3 {
			serviceFlag = strings.TrimSpace(fields[2])
		}

		project, err := resolveProject(ctx, client, name)
		if err != nil {
			return nil, err
		}
		service, err := resolveService(ctx, client, project.ID, serviceFlag)
		var ambiguous *ambiguousServiceError
		if errors.As(err, &ambiguous) && in != nil {
			service, err = chooseService(in, w, name, ambiguous.Candidates)
		}
		if err != nil {
			return nil, fmt.Errorf("resolving service for project %q: %w", name, err)
		}
		out = append(out, config.AutofillProject{Project: str(project.Attributes.Name), ServiceID: service.ID, Hours: hours})
	}
	return out, nil
}
