package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/cbarber/fortyhours/internal/productive"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// toOpenAPIDate converts a date-only time.Time to the wire date type.
func toOpenAPIDate(t time.Time) openapi_types.Date {
	return openapi_types.Date{Time: t}
}

// dateStr renders a *openapi_types.Date, or "" for nil.
func dateStr(d *openapi_types.Date) string {
	if d == nil {
		return ""
	}
	return d.String()
}

// hoursStr renders minutes (as stored by Productive) as fractional hours,
// e.g. 480 -> "8".
func hoursStr(minutes *int) string {
	if minutes == nil {
		return ""
	}
	return fmt.Sprintf("%g", float64(*minutes)/60)
}

// createTimeEntry logs hours worked on serviceID for date, for the
// configured person.
func createTimeEntry(ctx context.Context, app *App, serviceID string, date time.Time, hours float64, note string) (productive.Record[productive.ResourceTimeEntry], error) {
	serviceIDInt, err := toIntID(serviceID)
	if err != nil {
		return productive.Record[productive.ResourceTimeEntry]{}, err
	}
	personIDInt, err := toIntID(app.Config.PersonID)
	if err != nil {
		return productive.Record[productive.ResourceTimeEntry]{}, fmt.Errorf("invalid configured person_id: %w", err)
	}

	minutes := int(hours * 60)
	d := toOpenAPIDate(date)
	attrs := productive.ResourceTimeEntry{
		Date:      &d,
		Time:      &minutes,
		ServiceId: &serviceIDInt,
		PersonId:  &personIDInt,
	}
	if note != "" {
		attrs.Note = &note
	}
	return app.Client.CreateTimeEntry(ctx, attrs)
}

// buildBookingAttrs resolves --project/--service (work) or --event
// (absence) into a booking payload for the configured person.
func buildBookingAttrs(ctx context.Context, app *App, projectName, serviceFlag, eventName string, startedOn, endedOn time.Time, hours float64, note string, draft bool) (productive.ResourceBooking, error) {
	if (projectName != "" || serviceFlag != "") == (eventName != "") {
		return productive.ResourceBooking{}, fmt.Errorf("specify either --project/--service (work) or --event (absence), not both")
	}

	personIDInt, err := toIntID(app.Config.PersonID)
	if err != nil {
		return productive.ResourceBooking{}, fmt.Errorf("invalid configured person_id: %w", err)
	}

	started := toOpenAPIDate(startedOn)
	ended := toOpenAPIDate(endedOn)
	minutes := int(hours * 60)
	attrs := productive.ResourceBooking{
		PersonId:  &personIDInt,
		StartedOn: &started,
		EndedOn:   &ended,
		Time:      &minutes,
		Draft:     &draft,
	}
	if note != "" {
		attrs.Note = &note
	}

	if eventName != "" {
		event, err := resolveEvent(ctx, app.Client, eventName)
		if err != nil {
			return productive.ResourceBooking{}, err
		}
		eventID, err := toIntID(event.ID)
		if err != nil {
			return productive.ResourceBooking{}, err
		}
		attrs.EventId = &eventID
		return attrs, nil
	}

	service, err := resolveServiceForCommand(ctx, app, projectName, serviceFlag)
	if err != nil {
		return productive.ResourceBooking{}, err
	}
	serviceID, err := toIntID(service.ID)
	if err != nil {
		return productive.ResourceBooking{}, err
	}
	attrs.ServiceId = &serviceID
	return attrs, nil
}

// toIntID parses a Productive resource id string into an int.
func toIntID(s string) (int, error) {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", s, err)
	}
	return n, nil
}
