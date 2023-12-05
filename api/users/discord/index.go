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
var column string
var value string

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
  	crw.SetCors(r.Host)

	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create Supabase client: " + err.Error()})
		return
	}

  	id := r.URL.Query().Get("id")
	email := r.URL.Query().Get("email")
	if (len(id) > 0 && len(email) > 0) {
		crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Provide either 'id' or 'email', not both"})
		return
	} else if len(email) > 0 {
		column = "email"
		value = email
	} else {
		column = "id"
		value = id
	}
  
	switch r.Method {
	case "GET":
		var userDiscord []UserDiscord
		if _, err := client.From("User").Select("discord", "exact", false).Eq(column, value).ExecuteTo(&userDiscord); err != nil {
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
		
		if _, _, err := client.From("User").Update(updatedUserDiscord, "", "exact").Eq(column, value).Execute(); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update users discord: " + err.Error()})
			return
		}
		crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated users discord"})
	default:
		crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
	}
}