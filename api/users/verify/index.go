package verify

import (
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
)

type VerifyResponse struct {
	Exists bool `json:"exists"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
  	crw.SetCors(r.Host)

	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create Supabase client: " + err.Error()})
		return
	}

  	email := r.URL.Query().Get("email")
  
	switch r.Method {
	case "GET":
		var temp []interface{}
		if _, err := client.From("User").Select("*", "exact", false).Eq("email", email).ExecuteTo(&temp); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get users email: " + err.Error()})
			return
		}
		
		var verifyResponse VerifyResponse
		if len(temp) > 0 {
			verifyResponse = VerifyResponse{Exists: true}
		} else {
			verifyResponse = VerifyResponse{Exists: false}
		}
		crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": verifyResponse})
	default:
		crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
	}
}