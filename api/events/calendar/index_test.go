package calendar

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeSyncEvents(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single timed event",
			body:      `[{"google_event_id":"abc","title":"Workshop","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"scheduled"}]`,
			wantCount: 1,
		},
		{
			name:      "all-day event",
			body:      `[{"google_event_id":"abc","title":"Cat's Back","start_time":"2026-01-22","end_time":"2026-01-23","status":"scheduled"}]`,
			wantCount: 1,
		},
		{
			name: "batch with mixed optional fields",
			body: `[{"google_event_id":"a","title":"A","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"scheduled","description":"has one"},
			        {"google_event_id":"b","title":"B","start_time":"2026-02-14T18:00:00Z","end_time":"2026-02-14T20:00:00Z","status":"cancelled"}]`,
			wantCount: 2,
		},
		{
			name:      "empty batch",
			body:      `[]`,
			wantCount: 0,
		},
		{
			name:    "not an array",
			body:    `{"google_event_id":"abc"}`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			body:    `[{"google_event_id":"abc",`,
			wantErr: true,
		},
		{
			// google_event_id is the conflict target; without it an upsert would
			// insert a duplicate row on every single poll.
			name:    "missing google_event_id",
			body:    `[{"title":"Workshop","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"scheduled"}]`,
			wantErr: true,
		},
		{
			name:    "missing title",
			body:    `[{"google_event_id":"abc","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"scheduled"}]`,
			wantErr: true,
		},
		{
			name:    "malformed start_time",
			body:    `[{"google_event_id":"abc","title":"Workshop","start_time":"soon","end_time":"2026-02-13T20:00:00Z","status":"scheduled"}]`,
			wantErr: true,
		},
		{
			name:    "unknown status",
			body:    `[{"google_event_id":"abc","title":"Workshop","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"confirmed"}]`,
			wantErr: true,
		},
		{
			// One bad row must fail the whole batch rather than silently syncing
			// a partial calendar.
			name:    "one bad row in a good batch",
			body:    `[{"google_event_id":"a","title":"A","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z","status":"scheduled"},
			           {"google_event_id":"b","title":"","start_time":"2026-02-14T18:00:00Z","end_time":"2026-02-14T20:00:00Z","status":"scheduled"}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeSyncEvents(strings.NewReader(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeSyncEvents(%q) = %+v, want error", tt.body, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeSyncEvents(%q) unexpected error: %v", tt.body, err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("decoded %d events, want %d", len(got), tt.wantCount)
			}
		})
	}
}

// PostgREST builds the upsert's SET clause from the union of keys across the
// batch, and fills a key that some rows omit with the column default. So every
// row must serialise the same key set, and that set must exclude the columns
// officers own — otherwise a calendar poll silently resets is_public,
// point_category and flyer_url on every event it touches.
func TestSyncEventSerialisesAConstantKeySet(t *testing.T) {
	description := "has one"

	withOptional := SyncEvent{
		GoogleEventID: "a",
		Title:         "A",
		Description:   &description,
		StartTime:     "2026-02-13T18:00:00Z",
		EndTime:       "2026-02-13T20:00:00Z",
		Status:        "scheduled",
	}
	withoutOptional := SyncEvent{
		GoogleEventID: "b",
		Title:         "B",
		StartTime:     "2026-02-14T18:00:00Z",
		EndTime:       "2026-02-14T20:00:00Z",
		Status:        "cancelled",
	}

	first := keysOf(t, withOptional)
	second := keysOf(t, withoutOptional)

	if len(first) != len(second) {
		t.Fatalf("key sets differ: %v vs %v", first, second)
	}
	for key := range first {
		if _, ok := second[key]; !ok {
			t.Errorf("key %q present on one row but not the other", key)
		}
	}

	want := []string{"google_event_id", "title", "description", "location", "start_time", "end_time", "status"}
	if len(first) != len(want) {
		t.Errorf("serialised %d keys %v, want %d", len(first), first, len(want))
	}
	for _, key := range want {
		if _, ok := first[key]; !ok {
			t.Errorf("missing expected key %q", key)
		}
	}

	for _, forbidden := range []string{"is_public", "point_category", "flyer_url", "id"} {
		if _, ok := first[forbidden]; ok {
			t.Errorf("officer-owned column %q must never appear in a sync payload", forbidden)
		}
	}
}

func keysOf(t *testing.T, event SyncEvent) map[string]struct{} {
	t.Helper()

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	keys := make(map[string]struct{}, len(decoded))
	for key := range decoded {
		keys[key] = struct{}{}
	}
	return keys
}
