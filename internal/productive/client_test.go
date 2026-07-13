package productive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient("test-token", "test-org")
	c.BaseURL = srv.URL
	return c
}

func TestClientSetsAuthHeaders(t *testing.T) {
	var gotToken, gotOrg, gotAccept string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Auth-Token")
		gotOrg = r.Header.Get("X-Organization-Id")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.Write([]byte(`{"data":[]}`))
	})

	if _, err := c.ListProjects(context.Background(), nil); err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if gotToken != "test-token" {
		t.Errorf("X-Auth-Token = %q, want test-token", gotToken)
	}
	if gotOrg != "test-org" {
		t.Errorf("X-Organization-Id = %q, want test-org", gotOrg)
	}
	if gotAccept != "application/vnd.api+json" {
		t.Errorf("Accept = %q, want application/vnd.api+json", gotAccept)
	}
}

func TestListAllWalksPagination(t *testing.T) {
	pages := [][]byte{
		[]byte(`{"data":[{"id":"1","type":"projects","attributes":{"name":"dreamfi"}}],"meta":{"total_pages":2,"current_page":1}}`),
		[]byte(`{"data":[{"id":"2","type":"projects","attributes":{"name":"internal"}}],"meta":{"total_pages":2,"current_page":2}}`),
	}
	var calls int
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page[number]")
		if page == "2" {
			w.Write(pages[1])
		} else {
			w.Write(pages[0])
		}
		calls++
	})

	got, err := c.ListProjects(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 requests (one per page), got %d", calls)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(got))
	}
	if got[0].ID != "1" || *got[0].Attributes.Name != "dreamfi" {
		t.Errorf("unexpected first record: %+v", got[0])
	}
	if got[1].ID != "2" || *got[1].Attributes.Name != "internal" {
		t.Errorf("unexpected second record: %+v", got[1])
	}
}

func TestCreateTimeEntrySendsJSONAPIEnvelope(t *testing.T) {
	var body map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"42","type":"time_entries","attributes":{"time":480}}}`))
	})

	minutes := 480
	personID := 7
	serviceID := 3
	entry, err := c.CreateTimeEntry(context.Background(), ResourceTimeEntry{
		Time:      &minutes,
		PersonId:  &personID,
		ServiceId: &serviceID,
	})
	if err != nil {
		t.Fatalf("CreateTimeEntry: %v", err)
	}
	if entry.ID != "42" {
		t.Errorf("entry.ID = %q, want 42", entry.ID)
	}
	if entry.Attributes.Time == nil || *entry.Attributes.Time != 480 {
		t.Errorf("entry.Attributes.Time = %v, want 480", entry.Attributes.Time)
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("request body missing data object: %v", body)
	}
	if data["type"] != "time_entries" {
		t.Errorf("data.type = %v, want time_entries", data["type"])
	}
	if _, hasID := data["id"]; hasID {
		t.Errorf("create request body should not set id, got %v", data["id"])
	}
	attrs, ok := data["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("request body missing data.attributes: %v", body)
	}
	if attrs["time"] != float64(480) {
		t.Errorf("attributes.time = %v, want 480", attrs["time"])
	}
}

func TestAPIErrorSurfacesJSONAPIError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"status":"401","code":"invalid_token_signature","title":"Unauthenticated","detail":"Invalid signature"}]}`))
	})

	_, err := c.ListProjects(context.Background(), nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if len(apiErr.Errors) != 1 || apiErr.Errors[0].Code != "invalid_token_signature" {
		t.Errorf("unexpected errors: %+v", apiErr.Errors)
	}
}

func TestFilterBuildsJSONAPIQuery(t *testing.T) {
	var gotQuery string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"data":[]}`))
	})

	f := NewFilter().Eq("person_id", "7").Op("date", "gte", "2024-01-01")
	if _, err := c.ListTimeEntries(context.Background(), f); err != nil {
		t.Fatalf("ListTimeEntries: %v", err)
	}
	if got := gotQuery; !contains(got, "filter%5Bperson_id%5D=7") || !contains(got, "filter%5Bdate%5D%5Bgte%5D=2024-01-01") {
		t.Errorf("query = %q, missing expected filter params", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
