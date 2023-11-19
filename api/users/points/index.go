package points

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
)

type UserPoints struct {
	Points int `json:"points"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
  	crw.SetCors(r.Host)

	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create Supabase client: " + err.Error()})
		return
	}

  	id := r.URL.Query().Get("id")
  
	switch r.Method {
	case "GET":
		var userPoints []UserPoints
		if _, err := client.From("User").Select("points", "exact", false).Eq("id", id).ExecuteTo(&userPoints); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get users points: " + err.Error()})
			return
		}
		crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": userPoints})
	case "PATCH":
		var updatedUserPoints UserPoints
		if err := json.NewDecoder(r.Body).Decode(&updatedUserPoints); err != nil {
			crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
			return
		}
		// TODO: Perform validation checks on updatedUserPoints data
		
		if _, _, err := client.From("User").Update(updatedUserPoints, "", "exact").Eq("id", id).Execute(); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update user points: " + err.Error()})
			return
		}
		crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated user points with id: " + id})
	default:
		crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
	}
}