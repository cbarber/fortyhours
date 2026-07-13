package productive

import (
	"context"
	"fmt"
)

// ListTimesheets returns every timesheet matching filter (nil for no
// filter). Creating a timesheet for a person/date submits that person's
// time entries for that date for approval; Productive enforces at most one
// timesheet per person per date.
func (c *Client) ListTimesheets(ctx context.Context, filter *Filter) ([]Record[ResourceTimesheet], error) {
	return listAll[ResourceTimesheet](ctx, c, "/timesheets", filter)
}

// GetTimesheet fetches a single timesheet by id.
func (c *Client) GetTimesheet(ctx context.Context, id string) (Record[ResourceTimesheet], error) {
	return get[ResourceTimesheet](ctx, c, "/timesheets/"+id)
}

// CreateTimesheet creates a timesheet. attrs must set PersonId and Date.
func (c *Client) CreateTimesheet(ctx context.Context, attrs ResourceTimesheet) (Record[ResourceTimesheet], error) {
	return create(ctx, c, "/timesheets", "timesheets", attrs)
}

// DeleteTimesheet deletes a timesheet by id, un-submitting its time
// entries. Productive rejects this once any of those entries are approved.
func (c *Client) DeleteTimesheet(ctx context.Context, id string) error {
	return destroy(ctx, c, fmt.Sprintf("/timesheets/%s", id))
}
