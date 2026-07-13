package productive

import "context"

// ListServices returns every service matching filter (nil for no filter).
// Services live under projects (ResourceService.ProjectId) and are what
// time entries and bookings are actually tracked against.
func (c *Client) ListServices(ctx context.Context, filter *Filter) ([]Record[ResourceService], error) {
	return listAll[ResourceService](ctx, c, "/services", filter)
}
