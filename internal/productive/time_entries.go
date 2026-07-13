package productive

import (
	"context"
	"fmt"
)

// ListTimeEntries returns every time entry matching filter (nil for no
// filter).
func (c *Client) ListTimeEntries(ctx context.Context, filter *Filter) ([]Record[ResourceTimeEntry], error) {
	return listAll[ResourceTimeEntry](ctx, c, "/time_entries", filter)
}

// GetTimeEntry fetches a single time entry by id.
func (c *Client) GetTimeEntry(ctx context.Context, id string) (Record[ResourceTimeEntry], error) {
	return get[ResourceTimeEntry](ctx, c, "/time_entries/"+id)
}

// CreateTimeEntry creates a time entry. attrs must set at least Date, Time,
// PersonId and ServiceId, per Productive's requirements.
func (c *Client) CreateTimeEntry(ctx context.Context, attrs ResourceTimeEntry) (Record[ResourceTimeEntry], error) {
	return create(ctx, c, "/time_entries", "time_entries", attrs)
}

// UpdateTimeEntry patches a time entry. Only non-nil fields in attrs are
// sent.
func (c *Client) UpdateTimeEntry(ctx context.Context, id string, attrs ResourceTimeEntry) (Record[ResourceTimeEntry], error) {
	return update(ctx, c, "/time_entries/"+id, "time_entries", id, attrs)
}

// DeleteTimeEntry deletes a time entry by id.
func (c *Client) DeleteTimeEntry(ctx context.Context, id string) error {
	return destroy(ctx, c, fmt.Sprintf("/time_entries/%s", id))
}
