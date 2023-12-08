package points

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase-community/supabase-go"
)

type UserPoints struct {
	Points int `json:"points"`
}
type Response struct {
    Success bool `json:"success"`
	Data *UserPoints `json:"data,omitempty"`
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

	var column string
	var value string
	if id == "" && email == "" && discordId == "" {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide either 'id', 'email' or 'discordId' as a parameter",
			},
		})
		return
	} else if id != "" {
		column = "id"
		value = id
	} else if email != "" {
		column = "email"
		value = email
	} else if discordId != "" {
		column = "discord"
		value = discordId
	} else {
		column = "id"
		value = id
	}

	switch r.Method {
	case "GET":
		userPoints, err := getUserPointsByColumn(client, column, value)
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
			Data: userPoints,
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

func getUserPointsByColumn(client *supabase.Client, column string, value string) (*UserPoints, error) {
	var userPoints []UserPoints
	if _, err := client.From("User").Select("points", "exact", false).Eq(column, value).ExecuteTo(&userPoints); err != nil {
		return nil, err
	}
	if len(userPoints) == 0 {
		return nil, nil
	}

	return &userPoints[0], nil
}

func updateUserPoints(client *supabase.Client, column string, value string, userPoints UserPoints) (int64, error) {
	_, count, err := client.From("User").Update(userPoints, "", "exact").Eq(column, value).Execute()
	if err != nil{
		return 0, err
	}
	return count, nil
}