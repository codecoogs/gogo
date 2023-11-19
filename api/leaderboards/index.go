package leaderboards

import (
  "encoding/json"
  "net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
)

type Leaderboard struct {
  ID int `json:"id"`
  Name string `json:"name"`
  Teams uuid.UUID `json:"teams"`
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
      var leaderboard Leaderboard
      if err := json.NewDecoder(r.Body).Decode(&leaderboard); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }
      
      if _, _, err := client.From("Leaderboard").Insert(leaderboard, false, "", "", "exact").Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create leaderboard: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully created leaderboard"})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  } else {
    switch r.Method {
    case "GET":
      var leaderboard []Leaderboard
      if _, err := client.From("Leaderboard").Select("*", "exact", false).Eq("id", id).ExecuteTo(&leaderboard); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get leaderboard: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": leaderboard})
    case "PUT":
      var updatedLeaderboard Leaderboard
      if err := json.NewDecoder(r.Body).Decode(&updatedLeaderboard); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }

      if _, _, err := client.From("Leaderboard").Update(updatedLeaderboard, "", "exact").Eq("id", id).Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update leaderboard: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated leaderboard with id: " + id})
    case "DELETE":
      if _, _, err := client.From("Leaderboard").Delete("", "exact").Eq("id", id).Execute(); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to delete leaderboard: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully deleted leaderboard with id: " + id})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  }
}