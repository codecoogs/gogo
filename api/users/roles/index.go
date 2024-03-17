package roles

import (
	"encoding/json"
	"github.com/codecoogs/gogo/wrappers/auth"
	"net/http"
	"strconv"

	"github.com/codecoogs/gogo/constants"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase-community/supabase-go"
)

type UsersRoles struct {
	ID     *string `json:"id,omitempty"`
	UserID string  `json:"user_id"`
	RoleID int8    `json:"role_id"`
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

	if r.Method != "POST" {
		crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Method not allowed for this resource",
			},
		})
		return
	}

	if token := r.Header.Get("Authorization"); !codecoogsauth.Authorize(token) {
		crw.SendJSONResponse(http.StatusUnauthorized, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Unauthorized access",
			},
		})
		return
	}

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

	if !isValidUser(client, newUserRole.UserID) {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Invalid user",
			},
		})
		return
	}

	if !isValidRole(client, strconv.Itoa(int(newUserRole.RoleID))) {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Invalid role",
			},
		})
		return
	}

	if err := addUserRole(client, newUserRole); err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Failed to add user role: " + err.Error(),
			},
		})
		return
	}

	crw.SendJSONResponse(http.StatusOK, Response{Success: true})
}

func isValidUser(client *supabase.Client, userID string) bool {
	_, count, err := client.From(constants.USER_TABLE).Select("*", "exact", false).Eq("id", userID).Execute()
	return err == nil && count > 0
}

func isValidRole(client *supabase.Client, roleID string) bool {
	_, count, err := client.From(constants.ROLE_TABLE).Select("*", "exact", false).Eq("id", roleID).Execute()
	return err == nil && count > 0
}

func addUserRole(client *supabase.Client, userRole UsersRoles) error {
	_, _, err := client.From(constants.USERS_ROLES_TABLE).Insert(userRole, false, "", "", "exact").Execute()
	return err
}
