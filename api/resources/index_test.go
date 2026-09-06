package resources

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
			name:  "website_viewable false",
			query: "website_viewable=false",
			want:  listFilters{WebsiteViewable: boolPtr(false)},
		},
		{
			name:  "is_active true",
			query: "is_active=true",
			want:  listFilters{IsActive: boolPtr(true)},
		},
		{
			name:  "category",
			query: "category=Workshops",
			want:  listFilters{Category: "Workshops"},
		},
		{
			name:  "all filters combined",
			query: "website_viewable=true&is_active=false&category=Collaborations",
			want: listFilters{
				WebsiteViewable: boolPtr(true),
				IsActive:        boolPtr(false),
				Category:        "Collaborations",
			},
		},
		{
			name:    "website_viewable not a bool",
			query:   "website_viewable=yes",
			wantErr: true,
		},
		{
			name:    "is_active not a bool",
			query:   "is_active=1.5",
			wantErr: true,
		},
		{
			name:    "website_viewable present but empty",
			query:   "website_viewable=",
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
			if got.Category != tt.want.Category {
				t.Errorf("Category = %q, want %q", got.Category, tt.want.Category)
			}
		})
	}
}

func TestDecodeResource(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "minimal valid",
			body: `{"title":"Intro to Go","category":"Workshops","link_url":"https://example.com/a.pdf"}`,
		},
		{
			name: "full valid",
			body: `{"title":"Intro to Go","category":"Workshops","link_url":"https://example.com/a.pdf",
			        "description":"slides","extension":"pdf","thumbnail_url":"https://example.com/t.png",
			        "resource_date":"2026-02-13","display_order":3,
			        "website_viewable":true,"is_active":true}`,
		},
		{
			name:    "malformed json",
			body:    `{"title":"Intro to Go",`,
			wantErr: true,
		},
		{
			name:    "not an object",
			body:    `["nope"]`,
			wantErr: true,
		},
		{
			name:    "empty body",
			body:    ``,
			wantErr: true,
		},
		{
			name:    "missing title",
			body:    `{"category":"Workshops","link_url":"https://example.com/a.pdf"}`,
			wantErr: true,
		},
		{
			name:    "missing category",
			body:    `{"title":"Intro to Go","link_url":"https://example.com/a.pdf"}`,
			wantErr: true,
		},
		{
			name:    "missing link_url",
			body:    `{"title":"Intro to Go","category":"Workshops"}`,
			wantErr: true,
		},
		{
			name:    "title is whitespace only",
			body:    `{"title":"   ","category":"Workshops","link_url":"https://example.com/a.pdf"}`,
			wantErr: true,
		},
		{
			name:    "wrong type for display_order",
			body:    `{"title":"a","category":"b","link_url":"c","display_order":"third"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeResource(strings.NewReader(tt.body))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeResource(%q) = %+v, want error", tt.body, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeResource(%q) unexpected error: %v", tt.body, err)
			}
		})
	}
}

// A POST that omits is_active/website_viewable must not send them, so the
// column defaults in the migration apply instead of Go's zero value.
func TestDecodeResourceOmittedFlagsStayNil(t *testing.T) {
	got, err := decodeResource(strings.NewReader(
		`{"title":"a","category":"b","link_url":"c"}`))
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

func TestDecodeResourceExplicitFalseIsKept(t *testing.T) {
	got, err := decodeResource(strings.NewReader(
		`{"title":"a","category":"b","link_url":"c","website_viewable":false,"is_active":false}`))
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
