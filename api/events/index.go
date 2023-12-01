package events

import (
  "encoding/json"
  "net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
)

type Event struct {
  ID int `json:"id"`
  Type string `json:"type"`
  StartTime string `json:"start_time"`
  EndTime string `json:"end_time"`
  Location string `json:"location"`
  Name string `json:"name"`
  Description string `json:"description"`
  Points int `json:"points"`
  Leaderboard *int `json:"leaderboard"`
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
      var event Event
      if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }
      
      if _, _, err := client.From("Event").Insert(event, false, "", "", "exact").Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create event: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully created event"})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  } else {
    switch r.Method {
    case "GET":
      var event []Event
      if _, err := client.From("Event").Select("*", "exact", false).Eq("id", id).ExecuteTo(&event); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get event: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": event})
    case "PUT":
      var updatedLeaderboard Event
      if err := json.NewDecoder(r.Body).Decode(&updatedLeaderboard); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }

      if _, _, err := client.From("Event").Update(updatedLeaderboard, "", "exact").Eq("id", id).Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update event: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated event with id: " + id})
    case "DELETE":
      if _, _, err := client.From("Event").Delete("", "exact").Eq("id", id).Execute(); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to delete event: " + err.Error()})
        return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully deleted event with id: " + id})
    default:
      crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
    }
  }
}