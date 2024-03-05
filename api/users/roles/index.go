package roles

import (
	"encoding/json"
	"errors"
	"github.com/codecoogs/gogo/constants"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase-community/supabase-go"
	"net/http"
	"strconv"
)

type UsersRoles struct {
	UserID string `json:"user_id"`
	RoleID int8   `json:"role_id"`
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

	switch r.Method {
	case "POST":
		var newUserRole UsersRoles
		if err := json.NewDecoder(r.Body).Decode(&newUserRole); err != nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Invalid request body: " + err.Error(),
				},
			})
			return
		}

		if checkValidUser(client, newUserRole.UserID) != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Invalid user: " + err.Error(),
				},
			})
		}

		if checkValidRole(client, strconv.Itoa(int(newUserRole.RoleID))) != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Invalid role: " + err.Error(),
				},
			})
		}

		if addUserRole(client, newUserRole) != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Failed to add user role: " + err.Error(),
				},
			})
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

func checkValidUser(client *supabase.Client, user string) error {
	_, count, err := client.From(constants.USER_TABLE).Select("*", "exact", false).Eq("id", user).Execute()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("Unknown user " + user)
	}

	return nil
}

func checkValidRole(client *supabase.Client, role string) error {
	_, count, err := client.From(constants.ROLE_TABLE).Select("*", "exact", false).Eq("id", role).Execute()
	if err != nil {
		return err
	}

	if count == 0 {
		return errors.New("Unknown role " + role)
	}

	return nil
}

func addUserRole(client *supabase.Client, userRole UsersRoles) error {
	_, _, err := client.From(constants.USERS_ROLES_TABLE).Insert(userRole, false, "", "", "exact").Execute()
	return err
}
