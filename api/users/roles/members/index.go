package members

import (
	"github.com/codecoogs/gogo/constants"
	"github.com/codecoogs/gogo/wrappers/auth"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
	"net/http"
)

type UsersRoles struct {
	ID     *uuid.UUID `json:"id,omitempty"`
	UserID uuid.UUID  `json:"user_id"`
	RoleID int8       `json:"role_id"`
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

	id := r.URL.Query().Get("id")
	if len(id) == 0 {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide 'id'",
			},
		})
		return
	}
	
	if err := grantMembership(client, id); err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Failed to grant membership: " + err.Error(),
			},
		})
		return
	}

	crw.SendJSONResponse(http.StatusOK, Response{Success: true})
}

func grantMembership(client *supabase.Client, userIDString string) error {
	userID, err := uuid.Parse(userIDString)
	if err != nil {
		return err
	}
	newMemberRole := UsersRoles{
		UserID: userID,
		RoleID: 2,
	}
	_, _, err = client.From(constants.USERS_ROLES_TABLE).Insert(newMemberRole, false, "", "", "exact").Execute()
	return err
}
