package productive

import (
	"context"
	"fmt"
	"net/url"
)

// pageSize is the number of records requested per page when listing.
// Productive's collection endpoints are paginated; listAll walks every page.
const pageSize = 200

// Record pairs a resource's JSON:API id with its decoded attributes.
type Record[T any] struct {
	ID         string
	Attributes T
}

func list[T any](ctx context.Context, c *Client, path string, filter *Filter, page int) ([]resourceObject[T], *Meta, error) {
	query := filter.queryValues()
	query.Set("page[number]", fmt.Sprintf("%d", page))
	query.Set("page[size]", fmt.Sprintf("%d", pageSize))

	var doc collectionDocument[T]
	if err := c.do(ctx, "GET", path, query, nil, &doc); err != nil {
		return nil, nil, err
	}
	return doc.Data, doc.Meta, nil
}

// listAll fetches every page of a collection endpoint.
func listAll[T any](ctx context.Context, c *Client, path string, filter *Filter) ([]Record[T], error) {
	var out []Record[T]
	page := 1
	for {
		items, meta, err := list[T](ctx, c, path, filter, page)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			out = append(out, Record[T]{ID: item.ID, Attributes: item.Attributes})
		}
		if meta == nil || meta.TotalPages == nil || page >= *meta.TotalPages {
			break
		}
		page++
	}
	return out, nil
}

func get[T any](ctx context.Context, c *Client, path string) (Record[T], error) {
	var doc singleDocument[T]
	if err := c.do(ctx, "GET", path, url.Values{}, nil, &doc); err != nil {
		return Record[T]{}, err
	}
	return Record[T]{ID: doc.Data.ID, Attributes: doc.Data.Attributes}, nil
}

func create[T any](ctx context.Context, c *Client, path, resourceType string, attrs T) (Record[T], error) {
	var doc singleDocument[T]
	body := singleDocument[T]{Data: resourceObject[T]{Identifier: Identifier{Type: resourceType}, Attributes: attrs}}
	if err := c.do(ctx, "POST", path, url.Values{}, body, &doc); err != nil {
		return Record[T]{}, err
	}
	return Record[T]{ID: doc.Data.ID, Attributes: doc.Data.Attributes}, nil
}

func update[T any](ctx context.Context, c *Client, path, resourceType, id string, attrs T) (Record[T], error) {
	var doc singleDocument[T]
	body := singleDocument[T]{Data: resourceObject[T]{Identifier: Identifier{Type: resourceType, ID: id}, Attributes: attrs}}
	if err := c.do(ctx, "PATCH", path, url.Values{}, body, &doc); err != nil {
		return Record[T]{}, err
	}
	return Record[T]{ID: doc.Data.ID, Attributes: doc.Data.Attributes}, nil
}

func destroy(ctx context.Context, c *Client, path string) error {
	return c.do(ctx, "DELETE", path, url.Values{}, nil, nil)
}
