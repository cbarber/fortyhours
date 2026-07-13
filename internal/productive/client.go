package productive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const defaultBaseURL = "https://api.productive.io/api/v2"

// Client is a minimal JSON:API client for the Productive.io time-tracking
// API. It authenticates every request with the headers documented at
// https://developer.productive.io/guides/authorization.
type Client struct {
	BaseURL    string
	APIToken   string
	OrgID      string
	HTTPClient *http.Client

	// Debug, when true, writes each request's method/URL/body and each
	// response's status/body to DebugWriter (os.Stderr if nil). The
	// X-Auth-Token header value is never printed.
	Debug       bool
	DebugWriter io.Writer
}

// NewClient builds a Client authenticated with the given API token and
// organization ID.
func NewClient(apiToken, orgID string) *Client {
	return &Client{
		BaseURL:    defaultBaseURL,
		APIToken:   apiToken,
		OrgID:      orgID,
		HTTPClient: http.DefaultClient,
	}
}

func (c *Client) debugWriter() io.Writer {
	if c.DebugWriter != nil {
		return c.DebugWriter
	}
	return os.Stderr
}

// Identifier is a JSON:API resource identifier (the "type"/"id" pair used in
// both resource objects and relationships).
type Identifier struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

type resourceObject[T any] struct {
	Identifier
	Attributes T `json:"attributes"`
}

type singleDocument[T any] struct {
	Data resourceObject[T] `json:"data"`
}

type collectionDocument[T any] struct {
	Data []resourceObject[T] `json:"data"`
	Meta *Meta               `json:"meta,omitempty"`
}

type apiError struct {
	Status string `json:"status"`
	Code   string `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type errorDocument struct {
	Errors []apiError `json:"errors"`
}

// APIError is returned when Productive responds with a JSON:API error
// document.
type APIError struct {
	StatusCode int
	Errors     []apiError
}

func (e *APIError) Error() string {
	if len(e.Errors) == 0 {
		return fmt.Sprintf("productive: request failed with status %d", e.StatusCode)
	}
	parts := make([]string, len(e.Errors))
	for i, apiErr := range e.Errors {
		if apiErr.Detail != "" {
			parts[i] = fmt.Sprintf("%s: %s", apiErr.Title, apiErr.Detail)
		} else {
			parts[i] = apiErr.Title
		}
	}
	return fmt.Sprintf("productive: %s", strings.Join(parts, "; "))
}

// Filter builds Productive's JSON:API `filter[field]`/`filter[field][op]`
// query parameters.
type Filter struct {
	values url.Values
}

// NewFilter returns an empty Filter.
func NewFilter() *Filter {
	return &Filter{values: url.Values{}}
}

// Eq adds an exact-match filter, e.g. filter[person_id]=123.
func (f *Filter) Eq(field, value string) *Filter {
	f.values.Set(fmt.Sprintf("filter[%s]", field), value)
	return f
}

// Op adds an operator filter, e.g. filter[date][gte]=2024-01-01.
func (f *Filter) Op(field, op, value string) *Filter {
	f.values.Set(fmt.Sprintf("filter[%s][%s]", field, op), value)
	return f
}

func (f *Filter) queryValues() url.Values {
	if f == nil {
		return url.Values{}
	}
	return f.values
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("productive: encoding request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return fmt.Errorf("productive: building request: %w", err)
	}
	req.Header.Set("X-Auth-Token", c.APIToken)
	req.Header.Set("X-Organization-Id", c.OrgID)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	if c.Debug {
		reqBodyStr := ""
		if body != nil {
			b, _ := json.Marshal(body)
			reqBodyStr = string(b)
		}
		fmt.Fprintf(c.debugWriter(), "--> %s %s\n    X-Organization-Id: %s\n    body: %s\n", method, u, c.OrgID, reqBodyStr)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("productive: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("productive: reading response: %w", err)
	}

	if c.Debug {
		fmt.Fprintf(c.debugWriter(), "<-- %d %s\n    body: %s\n", resp.StatusCode, u, respBody)
	}

	if resp.StatusCode >= 400 {
		var errDoc errorDocument
		_ = json.Unmarshal(respBody, &errDoc)
		return &APIError{StatusCode: resp.StatusCode, Errors: errDoc.Errors}
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("productive: decoding response: %w", err)
	}
	return nil
}
