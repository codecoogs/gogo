package discord

import (
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/codecoogs/gogo/constants"
	"github.com/supabase-community/supabase-go"
)

type UserDiscord struct {
	Discord *string `json:"discord,omitempty"`
}
type Response struct {
	Success bool          `json:"success"`
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

	email := r.URL.Query().Get("email")
	discordID := r.URL.Query().Get("discordId")
	if len(email) == 0 || len(discordID) == 0 {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide both 'email' and 'discordId'",
			},
		})
		return
	}

	switch r.Method {
	case "PATCH":
		userDiscord, err := getUserDiscordByEmail(client, email)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Failed to check user existence: " + err.Error(),
				},
			})
			return
		}

		if userDiscord == nil {
			crw.SendJSONResponse(http.StatusNotFound, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "User not found.",
				},
			})
			return
		}

		if userDiscord.Discord != nil {
			crw.SendJSONResponse(http.StatusConflict, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Discord ID is already associated with this user.",
				},
			})
			return
		}

		if err := updateUserDiscord(client, email, discordID); err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Failed to update user discord: " + err.Error(),
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

func getUserDiscordByEmail(client *supabase.Client, email string) (*UserDiscord, error) {
	var userDiscord []UserDiscord
	if _, err := client.From(constants.USER_TABLE).Select("discord", "exact", false).Eq("email", email).ExecuteTo(&userDiscord); err != nil {
		return nil, err
	}

	if len(userDiscord) == 0 {
		return nil, nil
	}
	return &userDiscord[0], nil
}

func updateUserDiscord(client *supabase.Client, email, discordID string) error {
	_, _, err := client.From(constants.USER_TABLE).Update(map[string]interface{}{"discord": discordID}, "", "exact").Eq("email", email).Execute()
	return err
}