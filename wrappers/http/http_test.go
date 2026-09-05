package codecoogshttp

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetCache(t *testing.T) {
	recorder := httptest.NewRecorder()
	crw := &ResponseWriter{W: recorder}

	crw.SetCache(300)

	got := recorder.Header().Get(CacheControlHeader)
	for _, want := range []string{
		// The browser serves repeat page loads without reaching us at all.
		"max-age=300",
		// Vercel's CDN caches it too, so one origin hit covers every visitor.
		"s-maxage=300",
		// A stale response is fine while the refresh happens in the background.
		"stale-while-revalidate=600",
		"public",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Cache-Control = %q, missing %q", got, want)
		}
	}
}

// Anything that can change per request must not be cached, or one caller's
// response gets served to the next.
func TestSetNoCache(t *testing.T) {
	recorder := httptest.NewRecorder()
	crw := &ResponseWriter{W: recorder}

	crw.SetNoCache()

	got := recorder.Header().Get(CacheControlHeader)
	if !strings.Contains(got, "no-store") {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if strings.Contains(got, "public") {
		t.Errorf("Cache-Control = %q, must not be public", got)
	}
}

// SendJSONResponse writes the header block, so the cache directive has to be set
// before it is called or it is silently dropped.
func TestSetCacheSurvivesSendJSONResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	crw := &ResponseWriter{W: recorder}

	crw.SetCache(300)
	crw.SendJSONResponse(200, map[string]bool{"success": true})

	if got := recorder.Result().Header.Get(CacheControlHeader); got == "" {
		t.Error("Cache-Control was dropped by SendJSONResponse")
	}
}
