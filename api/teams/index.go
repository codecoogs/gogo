package teams

import (
	"encoding/json"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/codecoogs/gogo/constants"
	"github.com/google/uuid"
	"net/http"
)

type Team struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Points int       `json:"points"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Team        `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`
}

type ErrorDetails struct {
	Message string `json:"message"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
	crw.SetCors(r.Host)

	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Failed to create Supabase client: " + err.Error(),
			},
		})
		return
	}

	id := r.URL.Query().Get("id")

	if id == "" {
		switch r.Method {
		case "POST":
			var updatedTeam Team
			if err := json.NewDecoder(r.Body).Decode(&updatedTeam); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}
			// TODO: Perform validation checks on updatedTeam data

			if _, _, err := client.From(constants.TEAM_TABLE).Insert(updatedTeam, false, "", "", "exact").Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create team: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		default:
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed for this resource",
				},
			})
		}
	} else {
		switch r.Method {
		case "GET":
			var team []Team
			if _, err := client.From(constants.TEAM_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&team); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get team: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    team,
			})
		case "PUT":
			var updatedTeam Team
			if err := json.NewDecoder(r.Body).Decode(&updatedTeam); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}
			// TODO: Perform validation checks on updatedTeam data

			if _, _, err := client.From(constants.TEAM_TABLE).Update(updatedTeam, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update team: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			if _, _, err := client.From(constants.TEAM_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete team: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		default:
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed for this resource",
				},
			})
		}
	}
}