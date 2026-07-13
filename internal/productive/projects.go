package productive

import "context"

// ListProjects returns every project matching filter (nil for no filter).
func (c *Client) ListProjects(ctx context.Context, filter *Filter) ([]Record[ResourceProject], error) {
	return listAll[ResourceProject](ctx, c, "/projects", filter)
}
