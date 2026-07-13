package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/cbarber/fortyhours/internal/productive"
)

// resolveTestServer serves just enough of /projects and /services for
// resolveAutofillSpec's project/service resolution, filtering services by
// filter[project_id] the way the real API does.
func resolveTestServer(t *testing.T, projects, services []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		var records []map[string]any
		switch r.URL.Path {
		case "/projects":
			records = projects
		case "/services":
			records = services
			if projectID := r.URL.Query().Get("filter[project_id]"); projectID != "" {
				var filtered []map[string]any
				for _, s := range records {
					if strconv.Itoa(int(s["project_id"].(float64))) == projectID {
						filtered = append(filtered, s)
					}
				}
				records = filtered
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		data := make([]map[string]any, len(records))
		for i, rec := range records {
			data[i] = map[string]any{"id": rec["id"], "type": "x", "attributes": attributesWithoutID(rec)}
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": data,
			"meta": map[string]any{"total_pages": 1, "current_page": 1},
		})
	}))
}

func TestResolveAutofillSpecPicksAmbiguousServiceInteractively(t *testing.T) {
	srv := resolveTestServer(t,
		[]map[string]any{{"id": "1", "name": "DreamFi"}},
		[]map[string]any{
			{"id": "10", "name": "Product Manager", "project_id": float64(1), "time_tracking_enabled": true},
			{"id": "20", "name": "Senior Software Engineer", "project_id": float64(1), "time_tracking_enabled": true},
		},
	)
	defer srv.Close()

	client := productive.NewClient("test-token", "test-org")
	client.BaseURL = srv.URL

	in := bufio.NewReader(strings.NewReader("2\n"))
	var out bytes.Buffer

	plan, err := resolveAutofillSpec(context.Background(), client, "DreamFi:7", in, &out)
	if err != nil {
		t.Fatalf("resolveAutofillSpec: %v\nprompted output: %s", err, out.String())
	}
	if len(plan) != 1 || plan[0].ServiceID != "20" {
		t.Fatalf("expected the interactively chosen service 20, got %+v", plan)
	}
	if !strings.Contains(out.String(), "Multiple trackable services found") {
		t.Errorf("expected a disambiguation prompt, got: %s", out.String())
	}
}

func TestResolveAutofillSpecFailsAmbiguousServiceWithoutPrompting(t *testing.T) {
	srv := resolveTestServer(t,
		[]map[string]any{{"id": "1", "name": "DreamFi"}},
		[]map[string]any{
			{"id": "10", "name": "Product Manager", "project_id": float64(1), "time_tracking_enabled": true},
			{"id": "20", "name": "Senior Software Engineer", "project_id": float64(1), "time_tracking_enabled": true},
		},
	)
	defer srv.Close()

	client := productive.NewClient("test-token", "test-org")
	client.BaseURL = srv.URL

	_, err := resolveAutofillSpec(context.Background(), client, "DreamFi:7", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "multiple trackable services found") {
		t.Fatalf("expected an ambiguous-service error, got: %v", err)
	}
}

func TestResolveAutofillSpecExplicitServiceSkipsPrompt(t *testing.T) {
	srv := resolveTestServer(t,
		[]map[string]any{{"id": "1", "name": "DreamFi"}},
		[]map[string]any{
			{"id": "10", "name": "Product Manager", "project_id": float64(1), "time_tracking_enabled": true},
			{"id": "20", "name": "Senior Software Engineer", "project_id": float64(1), "time_tracking_enabled": true},
		},
	)
	defer srv.Close()

	client := productive.NewClient("test-token", "test-org")
	client.BaseURL = srv.URL

	plan, err := resolveAutofillSpec(context.Background(), client, "DreamFi:7:20", nil, nil)
	if err != nil {
		t.Fatalf("resolveAutofillSpec: %v", err)
	}
	if len(plan) != 1 || plan[0].ServiceID != "20" {
		t.Fatalf("expected service 20 from explicit spec, got %+v", plan)
	}
}
