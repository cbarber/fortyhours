package productive

import "context"

// ListEvents returns every absence event category (e.g. "Sick", "PTO")
// matching filter (nil for no filter). Events are referenced by bookings
// via ResourceBooking.EventId to represent time off.
func (c *Client) ListEvents(ctx context.Context, filter *Filter) ([]Record[ResourceEvent], error) {
	return listAll[ResourceEvent](ctx, c, "/events", filter)
}
