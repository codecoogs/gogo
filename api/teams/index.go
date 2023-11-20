package teams

import (
  "encoding/json"
  "net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
)

type Team struct {
  ID uuid.UUID `json:"id"`
  Name string `json:"name"`
  Points int `json:"points"`
  Leads uuid.UUID `json:"leads"`
  Members uuid.UUID `json:"members"`
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

  if id == "" {
    switch r.Method {
    case "POST":
      var updatedTeam Team
      if err := json.NewDecoder(r.Body).Decode(&updatedTeam); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }
      // TODO: Perform validation checks on updatedTeam data
      
      if _, _, err := client.From("Team").Insert(updatedTeam, false, "", "", "exact").Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create team: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully created team"})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  } else {
    switch r.Method {
    case "GET":
      var team []Team
      if _, err := client.From("Team").Select("*", "exact", false).Eq("id", id).ExecuteTo(&team); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get team: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": team})
    case "PUT":
      var updatedTeam Team
      if err := json.NewDecoder(r.Body).Decode(&updatedTeam); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }
      // TODO: Perform validation checks on updatedTeam data
      
      if _, _, err := client.From("Team").Update(updatedTeam, "", "exact").Eq("id", id).Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update team: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated team with id: " + id})
    case "DELETE":
      if _, _, err := client.From("Team").Delete("", "exact").Eq("id", id).Execute(); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to delete team: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully deleted team with id: " + id})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  }
}