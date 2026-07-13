package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cbarber/fortyhours/internal/config"
	"github.com/cbarber/fortyhours/internal/dates"
)

// fakeProductive is a minimal in-memory JSON:API server covering the
// endpoints the CLI calls, used to exercise autofill/sick's skip-day and
// delete-existing-timesheet logic without live Productive credentials.
type fakeProductive struct {
	mu          sync.Mutex
	nextID      int
	timeEntries map[string]map[string]any
	bookings    map[string]map[string]any
	events      []map[string]any
}

func newFakeProductive() *fakeProductive {
	return &fakeProductive{
		nextID:      100,
		timeEntries: map[string]map[string]any{},
		bookings:    map[string]map[string]any{},
		events: []map[string]any{
			{"id": "1", "name": "Sick"},
			{"id": "2", "name": "PTO"},
		},
	}
}

func (f *fakeProductive) newID() string {
	f.nextID++
	return strconv.Itoa(f.nextID)
}

func (f *fakeProductive) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.api+json")

		switch {
		case r.URL.Path == "/events" && r.Method == http.MethodGet:
			f.writeCollection(w, f.events)

		case r.URL.Path == "/time_entries" && r.Method == http.MethodGet:
			f.writeCollection(w, filterRecords(f.timeEntries, r, "date"))

		case r.URL.Path == "/time_entries" && r.Method == http.MethodPost:
			f.create(w, r, f.timeEntries)

		case strings.HasPrefix(r.URL.Path, "/time_entries/") && r.Method == http.MethodDelete:
			id := strings.TrimPrefix(r.URL.Path, "/time_entries/")
			delete(f.timeEntries, id)
			w.WriteHeader(http.StatusNoContent)

		case r.URL.Path == "/bookings" && r.Method == http.MethodGet:
			f.writeCollection(w, filterRecords(f.bookings, r, "started_on"))

		case r.URL.Path == "/bookings" && r.Method == http.MethodPost:
			f.create(w, r, f.bookings)

		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"errors":[{"status":"404","title":"not found","detail":%q}]}`, r.URL.Path)
		}
	}
}

func (f *fakeProductive) writeCollection(w http.ResponseWriter, records []map[string]any) {
	data := make([]map[string]any, len(records))
	for i, rec := range records {
		data[i] = map[string]any{"id": rec["id"], "type": "x", "attributes": attributesWithoutID(rec)}
	}
	json.NewEncoder(w).Encode(map[string]any{
		"data": data,
		"meta": map[string]any{"total_pages": 1, "current_page": 1},
	})
}

func (f *fakeProductive) create(w http.ResponseWriter, r *http.Request, table map[string]map[string]any) {
	var body struct {
		Data struct {
			Type       string         `json:"type"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := f.newID()
	body.Data.Attributes["id"] = id
	table[id] = body.Data.Attributes

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"data": map[string]any{"id": id, "type": body.Data.Type, "attributes": attributesWithoutID(body.Data.Attributes)},
	})
}

// attributesWithoutID drops the "id" key: fortyhours reads a record's id
// from the JSON:API envelope (Record.ID), and ResourceTimeEntry/
// ResourceBooking declare their "id" attribute as an integer, which would
// otherwise conflict with the string ids this fake server assigns.
func attributesWithoutID(rec map[string]any) map[string]any {
	out := make(map[string]any, len(rec))
	for k, v := range rec {
		if k == "id" {
			continue
		}
		out[k] = v
	}
	return out
}

// filterRecords applies the person_id/date-ish filters the CLI actually
// sends: filter[person_id], filter[<dateField>][gte|lte], and (for
// bookings) filter[started_on][lte]/filter[ended_on][gte].
func filterRecords(table map[string]map[string]any, r *http.Request, primaryDateField string) []map[string]any {
	q := r.URL.Query()
	var out []map[string]any
	for _, rec := range table {
		if v := q.Get("filter[person_id]"); v != "" && fmt.Sprint(rec["person_id"]) != v {
			continue
		}
		if v := q.Get(fmt.Sprintf("filter[%s][gte]", primaryDateField)); v != "" && fmt.Sprint(rec[primaryDateField]) < v {
			continue
		}
		if v := q.Get(fmt.Sprintf("filter[%s][lte]", primaryDateField)); v != "" && fmt.Sprint(rec[primaryDateField]) > v {
			continue
		}
		if v := q.Get("filter[ended_on][gte]"); v != "" && fmt.Sprint(rec["ended_on"]) < v {
			continue
		}
		if v := q.Get("filter[started_on][lte]"); v != "" && fmt.Sprint(rec["started_on"]) > v {
			continue
		}
		out = append(out, rec)
	}
	return out
}

// runCLI executes the root command against a fake Productive server backed
// by an isolated config file, returning combined stdout/stderr.
func runCLI(t *testing.T, srv *httptest.Server, cfg *config.Config, args ...string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("FORTYHOURS_CONFIG", filepath.Join(dir, "config.yaml"))
	t.Setenv("FORTYHOURS_BASE_URL", srv.URL)
	t.Setenv(config.EnvAPIToken, "test-token")
	t.Setenv(config.EnvOrgID, "test-org")

	if cfg.PersonID == "" {
		cfg.PersonID = "1"
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("saving test config: %v", err)
	}

	root := NewRootCommand()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestSickDeletesExistingTimeEntry(t *testing.T) {
	fake := newFakeProductive()
	fake.timeEntries["50"] = map[string]any{"id": "50", "person_id": float64(1), "date": "2024-03-04", "time": float64(480)}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	out, err := runCLI(t, srv, &config.Config{SickEvent: "Sick"}, "sick", "2024-03-04")
	if err != nil {
		t.Fatalf("sick: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "removed 1 existing time entry") {
		t.Errorf("expected existing time entry to be reported removed, got: %s", out)
	}
	if len(fake.timeEntries) != 0 {
		t.Errorf("expected existing time entry to be deleted, got %v", fake.timeEntries)
	}
	if len(fake.bookings) != 1 {
		t.Fatalf("expected 1 sick booking to be created, got %d", len(fake.bookings))
	}
}

func TestAutofillSkipsFilledAndAbsenceDays(t *testing.T) {
	monday, _, err := dates.AutofillRange("week")
	if err != nil {
		t.Fatalf("AutofillRange(week): %v", err)
	}
	tuesday := monday.AddDate(0, 0, 1)
	wednesday := monday.AddDate(0, 0, 2)

	fake := newFakeProductive()
	// Monday already has a time entry; Tuesday has a PTO booking. Both
	// should be skipped; Wednesday should be filled.
	fake.timeEntries["50"] = map[string]any{"id": "50", "person_id": float64(1), "date": dates.Format(monday), "time": float64(480)}
	fake.bookings["60"] = map[string]any{"id": "60", "person_id": float64(1), "event_id": float64(2), "started_on": dates.Format(tuesday), "ended_on": dates.Format(tuesday), "time": float64(480)}
	srv := httptest.NewServer(fake.handler())
	defer srv.Close()

	cfg := &config.Config{
		Autofill: []config.AutofillProject{{Project: "dreamfi", ServiceID: "10", Hours: 8}},
	}
	out, err := runCLI(t, srv, cfg, "autofill", "week")
	if err != nil {
		t.Fatalf("autofill: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, dates.Format(monday)+": skipped (already has time entries)") {
		t.Errorf("expected Monday to be skipped as already filled, got: %s", out)
	}
	if !strings.Contains(out, dates.Format(tuesday)+": skipped (absence booked)") {
		t.Errorf("expected Tuesday to be skipped as absence booked, got: %s", out)
	}
	if !strings.Contains(out, dates.Format(wednesday)+": filled") {
		t.Errorf("expected Wednesday to be filled, got: %s", out)
	}

	var created int
	for _, rec := range fake.timeEntries {
		if rec["date"] == dates.Format(wednesday) {
			created++
		}
	}
	if created != 1 {
		t.Errorf("expected 1 new time entry on %s, got %d (all: %v)", dates.Format(wednesday), created, fake.timeEntries)
	}
}
