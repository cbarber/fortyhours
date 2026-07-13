package productive

import "context"

// ListPeople returns every person matching filter (nil for no filter). Used
// during `init` to resolve the caller's person_id from their email.
func (c *Client) ListPeople(ctx context.Context, filter *Filter) ([]Record[ResourcePerson], error) {
	return listAll[ResourcePerson](ctx, c, "/people", filter)
}
