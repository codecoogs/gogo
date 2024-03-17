package users

import (
	"github.com/codecoogs/gogo/constants"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase-community/supabase-go"
	"net/http"
)

type User struct {
	ID      string  `json:"id,omitempty"`
	Discord *string `json:"discord,omitempty"`
}

type UserRole struct {
	UserID int8 `json:"user_id"`
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
		user, err := getUserByEmail(client, email)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Failed to check user existence: " + err.Error(),
				},
			})
			return
		}

		if user == nil {
			crw.SendJSONResponse(http.StatusNotFound, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "User not found.",
				},
			})
			return
		}

		if user.Discord != nil {
			crw.SendJSONResponse(http.StatusConflict, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Discord ID is already associated with this user.",
				},
			})
			return
		}

		if !eligibleUser(client, user.ID) {
			crw.SendJSONResponse(http.StatusForbidden, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "User is not a member.",
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

func getUserByEmail(client *supabase.Client, email string) (*User, error) {
	var user []User
	if _, err := client.From(constants.USER_TABLE).Select("id, discord", "exact", false).Eq("email", email).ExecuteTo(&user); err != nil {
		return nil, err
	}

	if len(user) == 0 {
		return nil, nil
	}
	return &user[0], nil
}

func eligibleUser(client *supabase.Client, userID string) bool {
	memberRole := "2"
	_, count, err := client.From(constants.USERS_ROLES_TABLE).Select("id", "exact", false).Eq("role_id", memberRole).Eq("user_id", userID).Execute()

	return !(err != nil || count == 0)
}

func updateUserDiscord(client *supabase.Client, email, discordID string) error {
	_, _, err := client.From(constants.USER_TABLE).Update(map[string]interface{}{"discord": discordID}, "", "exact").Eq("email", email).Execute()
	return err
}
