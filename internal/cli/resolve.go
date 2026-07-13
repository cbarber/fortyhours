package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cbarber/fortyhours/internal/productive"
)

// resolveProject finds the single project whose name matches name
// case-insensitively.
func resolveProject(ctx context.Context, c *productive.Client, name string) (productive.Record[productive.ResourceProject], error) {
	projects, err := c.ListProjects(ctx, nil)
	if err != nil {
		return productive.Record[productive.ResourceProject]{}, err
	}
	var matches []productive.Record[productive.ResourceProject]
	for _, p := range projects {
		if strings.EqualFold(str(p.Attributes.Name), name) {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return productive.Record[productive.ResourceProject]{}, fmt.Errorf("no project named %q found; run `fortyhours projects list`", name)
	default:
		return productive.Record[productive.ResourceProject]{}, fmt.Errorf("multiple projects named %q found; disambiguate with --service", name)
	}
}

// resolveService resolves which service to track time/bookings against for
// a project. If serviceFlag is set, it's matched by id or name. Otherwise
// the project must have exactly one trackable service.
func resolveService(ctx context.Context, c *productive.Client, projectID, serviceFlag string) (productive.Record[productive.ResourceService], error) {
	filter := productive.NewFilter().Eq("project_id", projectID)
	services, err := c.ListServices(ctx, filter)
	if err != nil {
		return productive.Record[productive.ResourceService]{}, err
	}

	if serviceFlag != "" {
		for _, s := range services {
			if s.ID == serviceFlag || strings.EqualFold(str(s.Attributes.Name), serviceFlag) {
				return s, nil
			}
		}
		return productive.Record[productive.ResourceService]{}, fmt.Errorf("no service %q found on this project; run `fortyhours services list --project <name>`", serviceFlag)
	}

	var trackable []productive.Record[productive.ResourceService]
	for _, s := range services {
		if isTrackable(s.Attributes) {
			trackable = append(trackable, s)
		}
	}
	switch len(trackable) {
	case 1:
		return trackable[0], nil
	case 0:
		return productive.Record[productive.ResourceService]{}, fmt.Errorf("no trackable service found on this project; run `fortyhours services list --project <name>` and pass --service")
	default:
		return productive.Record[productive.ResourceService]{}, &ambiguousServiceError{Candidates: trackable}
	}
}

// ambiguousServiceError means a project has more than one trackable
// service, so the caller must disambiguate: via --service, or (if it has a
// terminal to prompt on) by choosing from Candidates.
type ambiguousServiceError struct {
	Candidates []productive.Record[productive.ResourceService]
}

func (e *ambiguousServiceError) Error() string {
	names := make([]string, len(e.Candidates))
	for i, s := range e.Candidates {
		names[i] = fmt.Sprintf("%s (id=%s)", str(s.Attributes.Name), s.ID)
	}
	return fmt.Sprintf("multiple trackable services found, pass --service: %s", strings.Join(names, ", "))
}

// isTrackable reports whether time can be tracked against a service.
// Productive's default (sparse) service response includes
// time_tracking_enabled but omits the filter-only for_tracking field
// entirely, so prefer the former and only fall back to the latter if it's
// ever present instead.
func isTrackable(s productive.ResourceService) bool {
	if s.TimeTrackingEnabled != nil {
		return *s.TimeTrackingEnabled
	}
	return s.ForTracking != nil && *s.ForTracking
}

// resolveEvent finds the single absence event category matching name
// case-insensitively.
func resolveEvent(ctx context.Context, c *productive.Client, name string) (productive.Record[productive.ResourceEvent], error) {
	events, err := c.ListEvents(ctx, nil)
	if err != nil {
		return productive.Record[productive.ResourceEvent]{}, err
	}
	var matches []productive.Record[productive.ResourceEvent]
	for _, e := range events {
		if strings.EqualFold(str(e.Attributes.Name), name) {
			matches = append(matches, e)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return productive.Record[productive.ResourceEvent]{}, fmt.Errorf("no absence event named %q found; run `fortyhours events list`", name)
	default:
		return productive.Record[productive.ResourceEvent]{}, fmt.Errorf("multiple absence events named %q found", name)
	}
}

// resolvePersonByEmail finds the single person matching email.
func resolvePersonByEmail(ctx context.Context, c *productive.Client, email string) (productive.Record[productive.ResourcePerson], error) {
	filter := productive.NewFilter().Eq("email", email)
	people, err := c.ListPeople(ctx, filter)
	if err != nil {
		return productive.Record[productive.ResourcePerson]{}, err
	}
	switch len(people) {
	case 1:
		return people[0], nil
	case 0:
		return productive.Record[productive.ResourcePerson]{}, fmt.Errorf("no person with email %q found", email)
	default:
		return productive.Record[productive.ResourcePerson]{}, fmt.Errorf("multiple people with email %q found", email)
	}
}

// resolveServiceForCommand resolves the service to track time/bookings
// against, given a command's --project and/or --service flags.
func resolveServiceForCommand(ctx context.Context, app *App, projectName, serviceFlag string) (productive.Record[productive.ResourceService], error) {
	if projectName != "" {
		project, err := resolveProject(ctx, app.Client, projectName)
		if err != nil {
			return productive.Record[productive.ResourceService]{}, err
		}
		return resolveService(ctx, app.Client, project.ID, serviceFlag)
	}
	if serviceFlag == "" {
		return productive.Record[productive.ResourceService]{}, fmt.Errorf("either --project or --service is required")
	}

	services, err := app.Client.ListServices(ctx, nil)
	if err != nil {
		return productive.Record[productive.ResourceService]{}, err
	}
	for _, s := range services {
		if s.ID == serviceFlag || strings.EqualFold(str(s.Attributes.Name), serviceFlag) {
			return s, nil
		}
	}
	return productive.Record[productive.ResourceService]{}, fmt.Errorf("no service %q found", serviceFlag)
}

// parseID validates that s looks like a Productive resource id.
func parseID(s string) (string, error) {
	if _, err := strconv.Atoi(s); err != nil {
		return "", fmt.Errorf("invalid id %q: expected a number", s)
	}
	return s, nil
}
