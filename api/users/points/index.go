package points

import (
	"encoding/json"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/codecoogs/gogo/constants"
	"github.com/supabase-community/supabase-go"
	"net/http"
)

type UserPoints struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Points    int    `json:"points"`
}
type Response struct {
	Success bool          `json:"success"`
	Data    *UserPoints   `json:"data,omitempty"`
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
	email := r.URL.Query().Get("email")
	discordId := r.URL.Query().Get("discordId")

	column, value := getColumnAndValue(id, email, discordId)
	if column == "" && value == "" {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide either 'id', 'email' or 'discordId' as a parameter",
			},
		})
		return
	}

	switch r.Method {
	case "GET":
		userPoints, err := getNameAndPointsByColumn(client, column, value)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: err.Error(),
				},
			})
			return
		}

		if userPoints == nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "User not found",
				},
			})
			return
		}

		crw.SendJSONResponse(http.StatusOK, Response{
			Success: true,
			Data:    userPoints,
		})
	case "PATCH":
		var updatedUserPoints UserPoints
		if err := json.NewDecoder(r.Body).Decode(&updatedUserPoints); err != nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: err.Error(),
				},
			})
			return
		}

		count, err := updateUserPoints(client, column, value, updatedUserPoints)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: err.Error(),
				},
			})
			return
		}

		if count == 0 {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "User not found",
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

func getColumnAndValue(id string, email string, discordId string) (string, string) {
	if id != "" {
		return "id", id
	}
	if email != "" {
		return "email", email
	}
	if discordId != "" {
		return "discord", discordId
	}
	return "", ""
}

func getNameAndPointsByColumn(client *supabase.Client, column string, value string) (*UserPoints, error) {
	var userPoints []UserPoints
	if _, err := client.From(constants.USER_TABLE).Select("first_name, last_name, points", "exact", false).Eq(column, value).ExecuteTo(&userPoints); err != nil {
		return nil, err
	}
	if len(userPoints) == 0 {
		return nil, nil
	}

	return &userPoints[0], nil
}

func updateUserPoints(client *supabase.Client, column string, value string, userPoints UserPoints) (int64, error) {
	_, count, err := client.From(constants.USER_TABLE).Update(userPoints, "", "exact").Eq(column, value).Execute()
	if err != nil {
		return 0, err
	}
	return count, nil
}