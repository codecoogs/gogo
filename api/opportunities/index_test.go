package opportunities

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
			name:  "website_viewable true",
			query: "website_viewable=true",
			want:  listFilters{WebsiteViewable: boolPtr(true)},
		},
		{
			name:  "is_active false",
			query: "is_active=false",
			want:  listFilters{IsActive: boolPtr(false)},
		},
		{
			name:  "both filters combined",
			query: "website_viewable=true&is_active=true",
			want: listFilters{
				WebsiteViewable: boolPtr(true),
				IsActive:        boolPtr(true),
			},
		},
		{
			name:    "website_viewable not a bool",
			query:   "website_viewable=maybe",
			wantErr: true,
		},
		{
			name:    "is_active not a bool",
			query:   "is_active=2",
			wantErr: true,
		},
		{
			name:    "is_active present but empty",
			query:   "is_active=",
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

			if !equalBoolPtr(got.WebsiteViewable, tt.want.WebsiteViewable) {
				t.Errorf("WebsiteViewable = %v, want %v", fmtBoolPtr(got.WebsiteViewable), fmtBoolPtr(tt.want.WebsiteViewable))
			}
			if !equalBoolPtr(got.IsActive, tt.want.IsActive) {
				t.Errorf("IsActive = %v, want %v", fmtBoolPtr(got.IsActive), fmtBoolPtr(tt.want.IsActive))
			}
		})
	}
}

func TestDecodeOpportunity(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "minimal valid",
			body: `{"title":"Intern","link_url":"https://forms.gle/abc"}`,
		},
		{
			name: "full valid",
			body: `{"title":"Intern","description":"Work under a director","icon_url":"/assets/intern.svg",
			        "category":"Club Role","term":"Spring 2026","link_url":"https://forms.gle/abc",
			        "opens_on":"2026-01-05","closes_on":"2026-02-01","display_order":1,
			        "website_viewable":true,"is_active":true}`,
		},
		{
			name:    "malformed json",
			body:    `{"title":"Intern"`,
			wantErr: true,
		},
		{
			name:    "not an object",
			body:    `"Intern"`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
		{
			name:    "missing title",
			body:    `{"term":"Spring 2026","link_url":"https://forms.gle/abc"}`,
			wantErr: true,
		},
		{
			name:    "title is whitespace only",
			body:    `{"title":"  ","link_url":"https://forms.gle/abc"}`,
			wantErr: true,
		},
		{
			// link_url is NOT NULL on the table, so the API rejects it up front
			// rather than surfacing a Postgres constraint error.
			name:    "missing link_url",
			body:    `{"title":"Intern","term":"Spring 2026"}`,
			wantErr: true,
		},
		{
			name:    "link_url is whitespace only",
			body:    `{"title":"Intern","link_url":"   "}`,
			wantErr: true,
		},
		{
			name:    "wrong type for display_order",
			body:    `{"title":"Intern","link_url":"https://forms.gle/abc","display_order":"first"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeOpportunity(strings.NewReader(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeOpportunity(%q) = %+v, want error", tt.body, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeOpportunity(%q) unexpected error: %v", tt.body, err)
			}
		})
	}
}

// A POST that omits is_active/website_viewable must not send them, so the
// column defaults in the migration apply instead of Go's zero value.
func TestDecodeOpportunityOmittedFlagsStayNil(t *testing.T) {
	got, err := decodeOpportunity(strings.NewReader(
		`{"title":"Intern","link_url":"https://forms.gle/abc"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.WebsiteViewable != nil {
		t.Errorf("WebsiteViewable = %v, want nil when omitted", *got.WebsiteViewable)
	}
	if got.IsActive != nil {
		t.Errorf("IsActive = %v, want nil when omitted", *got.IsActive)
	}
}

func TestDecodeOpportunityExplicitFalseIsKept(t *testing.T) {
	got, err := decodeOpportunity(strings.NewReader(
		`{"title":"Intern","link_url":"https://forms.gle/abc","website_viewable":false,"is_active":false}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.WebsiteViewable == nil || *got.WebsiteViewable {
		t.Errorf("WebsiteViewable = %v, want explicit false", fmtBoolPtr(got.WebsiteViewable))
	}
	if got.IsActive == nil || *got.IsActive {
		t.Errorf("IsActive = %v, want explicit false", fmtBoolPtr(got.IsActive))
	}
}

func boolPtr(b bool) *bool { return &b }

func equalBoolPtr(a, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func fmtBoolPtr(b *bool) string {
	if b == nil {
		return "nil"
	}
	if *b {
		return "true"
	}
	return "false"
}
