package productive

import (
	"context"
	"fmt"
)

// ListBookings returns every booking matching filter (nil for no filter).
// A booking represents either scheduled project work (ServiceId set) or an
// absence such as sick leave or PTO (EventId set).
func (c *Client) ListBookings(ctx context.Context, filter *Filter) ([]Record[ResourceBooking], error) {
	return listAll[ResourceBooking](ctx, c, "/bookings", filter)
}

// GetBooking fetches a single booking by id.
func (c *Client) GetBooking(ctx context.Context, id string) (Record[ResourceBooking], error) {
	return get[ResourceBooking](ctx, c, "/bookings/"+id)
}

// CreateBooking creates a booking. attrs must set at least PersonId,
// StartedOn and EndedOn, plus either ServiceId (work) or EventId (absence).
func (c *Client) CreateBooking(ctx context.Context, attrs ResourceBooking) (Record[ResourceBooking], error) {
	return create(ctx, c, "/bookings", "bookings", attrs)
}

// UpdateBooking patches a booking. Only non-nil fields in attrs are sent.
func (c *Client) UpdateBooking(ctx context.Context, id string, attrs ResourceBooking) (Record[ResourceBooking], error) {
	return update(ctx, c, "/bookings/"+id, "bookings", id, attrs)
}

// DeleteBooking deletes a booking by id.
func (c *Client) DeleteBooking(ctx context.Context, id string) error {
	return destroy(ctx, c, fmt.Sprintf("/bookings/%s", id))
}
