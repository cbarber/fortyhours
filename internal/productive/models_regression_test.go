package productive

import (
	"encoding/json"
	"testing"
)

// These payloads are trimmed real responses (or exact shapes of them) that
// previously failed to decode because Productive's spec declares the
// filter-parameter's scalar type for a field the actual resource returns as
// a different shape. See spec/tools/update_spec.py's
// SCALAR_TYPE_OVERRIDES_TO_ANY. Regenerating models_gen.go without that
// override reintroduces these decode failures.

func TestResourceProjectDecodesRealPayload(t *testing.T) {
	raw := `{
		"name": "DreamFi",
		"project_type_id": 2,
		"project_color_id": 2,
		"tag_colors": {},
		"template": false,
		"duplication_status": "idle",
		"archived_at": null
	}`
	var p ResourceProject
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("decoding project: %v", err)
	}
	if p.Name == nil || *p.Name != "DreamFi" {
		t.Errorf("Name = %v, want DreamFi", p.Name)
	}
	if p.ArchivedAt != nil {
		t.Errorf("ArchivedAt = %v, want nil", p.ArchivedAt)
	}
}

func TestResourcePersonDecodesRealPayload(t *testing.T) {
	raw := `{"first_name": "Craig", "last_name": "Barber", "tag_list": ["remote", "backend"]}`
	var p ResourcePerson
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("decoding person: %v", err)
	}
	if p.FirstName == nil || *p.FirstName != "Craig" {
		t.Errorf("FirstName = %v, want Craig", p.FirstName)
	}
}

func TestResourceEventDecodesRealPayload(t *testing.T) {
	raw := `{"name": "Sick leave", "color_id": "red", "status": 1}`
	var e ResourceEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("decoding event: %v", err)
	}
	if e.Name == nil || *e.Name != "Sick leave" {
		t.Errorf("Name = %v, want %q", e.Name, "Sick leave")
	}
}
