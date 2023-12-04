package discord

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
)

type UserDiscord struct {
	Discord *string `json:"discord"`
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
		var userDiscord []UserDiscord
		if _, err := client.From("User").Select("discord", "exact", false).Eq("id", id).ExecuteTo(&userDiscord); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get users discord: " + err.Error()})
			return
		}
		crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": userDiscord})
	case "PATCH":
		var updatedUserDiscord UserDiscord
		if err := json.NewDecoder(r.Body).Decode(&updatedUserDiscord); err != nil {
			crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
			return
		}
		
		if _, _, err := client.From("User").Update(updatedUserDiscord, "", "exact").Eq("id", id).Execute(); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update user discord: " + err.Error()})
			return
		}
		crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated user discord with id: " + id})
	default:
		crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
	}
}