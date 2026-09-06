package codecoogshttp

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	ContentTypeHeader  = "Content-Type"
	JSONContentType    = "application/json"
	CacheControlHeader = "Cache-Control"
)

type ResponseWriter struct {
	W http.ResponseWriter
}

func (crw *ResponseWriter) SetCors(origin string) {
	crw.W.Header().Set("Access-Control-Allow-Origin", "*")
	crw.W.Header().Set("Access-Control-Allow-Methods", "*")
	crw.W.Header().Set("Access-Control-Allow-Headers", "*")
}

// SetCache marks a response as cacheable for maxAge seconds by the browser and
// by any CDN in front of it, so repeat page loads are served without reaching
// the API. Must be called before SendJSONResponse, which writes the headers.
func (crw *ResponseWriter) SetCache(maxAge int) {
	crw.W.Header().Set(CacheControlHeader, fmt.Sprintf(
		"public, max-age=%d, s-maxage=%d, stale-while-revalidate=%d",
		maxAge, maxAge, maxAge*2,
	))
}

// SetNoCache marks a response as uncacheable, for anything that varies per
// request or must not be shared between callers.
func (crw *ResponseWriter) SetNoCache() {
	crw.W.Header().Set(CacheControlHeader, "no-store")
}

func (crw *ResponseWriter) SendJSONResponse(status int, payload interface{}) {
	crw.W.Header().Set(ContentTypeHeader, JSONContentType)
	crw.W.WriteHeader(status)

	jsonResp, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	if _, err := crw.W.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
