package roles

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
)

type Role struct {
	ID          *int64  `json:"id,omitempty"`
	RoleName    string  `json:"role_name"`
	Description *string `json:"description,omitempty"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Role       `json:"data,omitempty"`
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
		case "GET":
			var roles []Role
			if _, err := client.From(constants.ROLE_TABLE).Select("*", "exact", false).ExecuteTo(&roles); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get roles: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    roles,
			})
		case "POST":
			var newRole Role
			if err := json.NewDecoder(r.Body).Decode(&newRole); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			// Validate required fields
			if newRole.RoleName == "" {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "role_name is required",
					},
				})
				return
			}

			// Create role (id will be auto-generated)
			roleToInsert := Role{
				RoleName:    newRole.RoleName,
				Description: newRole.Description,
			}

			if _, _, err := client.From(constants.ROLE_TABLE).Insert(roleToInsert, false, "", "", "exact").Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create role: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "OPTIONS":
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
		// Convert id to int64 for querying
		roleID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Invalid role id: " + err.Error(),
				},
			})
			return
		}

		switch r.Method {
		case "GET":
			var roles []Role
			if _, err := client.From(constants.ROLE_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&roles); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get role: " + err.Error(),
					},
				})
				return
			}

			if len(roles) == 0 {
				crw.SendJSONResponse(http.StatusNotFound, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Role not found",
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    roles,
			})
		case "PUT":
			var updatedRole Role
			if err := json.NewDecoder(r.Body).Decode(&updatedRole); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			// Validate required fields
			if updatedRole.RoleName == "" {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "role_name is required",
					},
				})
				return
			}

			// Set the ID for the update
			updatedRole.ID = &roleID

			// Update role
			if _, _, err := client.From(constants.ROLE_TABLE).Update(updatedRole, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update role: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			// Delete/deactivate role
			if _, _, err := client.From(constants.ROLE_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete role: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "OPTIONS":
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
