package events

import (
	"net/url"
	"strings"
	"testing"
)

func TestParseListFilters(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    listFilters
		wantErr bool
	}{
		{
			name:  "no filters",
			query: "",
			want:  listFilters{},
		},
		{
			name:  "is_public true",
			query: "is_public=true",
			want:  listFilters{IsPublic: boolPtr(true)},
		},
		{
			name:  "is_public false",
			query: "is_public=false",
			want:  listFilters{IsPublic: boolPtr(false)},
		},
		{
			name:  "status",
			query: "status=scheduled",
			want:  listFilters{Status: "scheduled"},
		},
		{
			name:  "start_after",
			query: "start_after=2026-01-01T00:00:00Z",
			want:  listFilters{StartAfter: "2026-01-01T00:00:00Z"},
		},
		{
			name:  "start_before",
			query: "start_before=2026-12-31T00:00:00Z",
			want:  listFilters{StartBefore: "2026-12-31T00:00:00Z"},
		},
		{
			// What the website actually sends.
			name:  "all filters combined",
			query: "is_public=true&status=scheduled&start_after=2026-01-01T00:00:00Z&start_before=2026-06-01T00:00:00Z",
			want: listFilters{
				IsPublic:    boolPtr(true),
				Status:      "scheduled",
				StartAfter:  "2026-01-01T00:00:00Z",
				StartBefore: "2026-06-01T00:00:00Z",
			},
		},
		{
			name:    "is_public not a bool",
			query:   "is_public=yes",
			wantErr: true,
		},
		{
			name:    "is_public present but empty",
			query:   "is_public=",
			wantErr: true,
		},
		{
			// A bad timestamp must be rejected here rather than handed to
			// PostgREST, which would fail with a 500 describing the schema.
			name:    "start_after not a timestamp",
			query:   "start_after=next tuesday",
			wantErr: true,
		},
		{
			name:    "start_before not a timestamp",
			query:   "start_before=2026-13-45",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("bad test query %q: %v", tt.query, err)
			}

			got, err := parseListFilters(q)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseListFilters(%q) = %+v, want error", tt.query, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseListFilters(%q) unexpected error: %v", tt.query, err)
			}

			if !equalBoolPtr(got.IsPublic, tt.want.IsPublic) {
				t.Errorf("IsPublic = %v, want %v", fmtBoolPtr(got.IsPublic), fmtBoolPtr(tt.want.IsPublic))
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.StartAfter != tt.want.StartAfter {
				t.Errorf("StartAfter = %q, want %q", got.StartAfter, tt.want.StartAfter)
			}
			if got.StartBefore != tt.want.StartBefore {
				t.Errorf("StartBefore = %q, want %q", got.StartBefore, tt.want.StartBefore)
			}
		})
	}
}

func TestDecodeEvent(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "minimal valid",
			body: `{"title":"Intro to Python Workshop","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z"}`,
		},
		{
			name: "full valid",
			body: `{"title":"Intro to Python Workshop","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z",
			        "description":"Fundamentals of Python","location":"PGH 563","google_event_id":"abc123",
			        "point_category":"Workshop Attendance","flyer_url":"https://example.com/f.png",
			        "is_public":true,"status":"scheduled"}`,
		},
		{
			name:    "malformed json",
			body:    `{"title":"Intro to Python",`,
			wantErr: true,
		},
		{
			name:    "missing title",
			body:    `{"start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z"}`,
			wantErr: true,
		},
		{
			name:    "blank title",
			body:    `{"title":"   ","start_time":"2026-02-13T18:00:00Z","end_time":"2026-02-13T20:00:00Z"}`,
			wantErr: true,
		},
		{
			name:    "missing start_time",
			body:    `{"title":"Intro to Python","end_time":"2026-02-13T20:00:00Z"}`,
			wantErr: true,
		},
		{
			name:    "missing end_time",
			body:    `{"title":"Intro to Python","start_time":"2026-02-13T18:00:00Z"}`,
			wantErr: true,
		},
		{
			name:    "malformed start_time",
			body:    `{"title":"Intro to Python","start_time":"tomorrow","end_time":"2026-02-13T20:00:00Z"}`,
			wantErr: true,
		},
		{
			// The calendar is the source of truth for ordering; a backwards event
			// would sort into the list in a way nobody can explain.
			name:    "end before start",
			body:    `{"title":"Intro to Python","start_time":"2026-02-13T20:00:00Z","end_time":"2026-02-13T18:00:00Z"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeEvent(strings.NewReader(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeEvent(%q) = %+v, want error", tt.body, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeEvent(%q) unexpected error: %v", tt.body, err)
			}
		})
	}
}

// An all-day event arrives from Google as a date with no time. It must survive
// decoding, because the sync route feeds the same validation.
func TestDecodeEventAcceptsDateOnlyTimes(t *testing.T) {
	body := `{"title":"Cat's Back","start_time":"2026-01-22","end_time":"2026-01-23"}`

	if _, err := decodeEvent(strings.NewReader(body)); err != nil {
		t.Fatalf("decodeEvent rejected a date-only event: %v", err)
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func equalBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fmtBoolPtr(b *bool) string {
	if b == nil {
		return "<nil>"
	}
	if *b {
		return "true"
	}
	return "false"
}
