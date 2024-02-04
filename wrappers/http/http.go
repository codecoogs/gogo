package codecoogshttp

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const (
	ContentTypeHeader   = "Content-Type"
	JSONContentType     = "application/json"
	ControlOriginHeader = "Access-Control-Allow-Origin"
)

type ResponseWriter struct {
	W http.ResponseWriter
}

func (crw *ResponseWriter) SetCors(origin string) {
	if strings.Contains(origin, "www.codecoogs.com") {
		crw.W.Header().Set(ControlOriginHeader, "https://codecoogs.com")
	}

	if strings.Contains(origin, "127.0.0.1") {
		crw.W.Header().Set(ControlOriginHeader, "http://localhost:3001")
	}

	crw.W.Header().Set("Access-Control-Allow-Methods", "*")
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
