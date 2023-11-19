package users

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
)

type User struct {
  ID uuid.UUID `json:"id"`
  Role string `json:"role"`
  FirstName string `json:"first_name"`
  LastName string `json:"last_name"`
  Email string `json:"email"`
  Phone string `json:"phone"`
  Password string `json:"password"`
  Classification string `json:"classification"`
  ExpectedGraduation string `json:"expected_graduation"`
  Discord *string `json:"discord"`
  Team *uuid.UUID `json:"team"`
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
    var user []User
    if _, err := client.From("User").Select("*", "exact", false).Eq("id", id).ExecuteTo(&user); err != nil {
      crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get users: " + err.Error()})
      return
    }
    crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": user})
  case "PUT":
    var updatedUser User
    if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
      crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
      return
    }
    // TODO: Perform validation checks on updatedUser data
    
    if _, _, err := client.From("User").Update(updatedUser, "", "exact").Eq("id", id).Execute(); err != nil {
        crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update user: " + err.Error()})
        return
    }
    crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated user with id: " + id})
  case "DELETE":
    if _, _, err := client.From("User").Delete("", "exact").Eq("id", id).Execute(); err != nil {
      crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to delete users: " + err.Error()})
      return
    }
    crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully deleted user with id: " + id})
  default:
    crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
  }
}