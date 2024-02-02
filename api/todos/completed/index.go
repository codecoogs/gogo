package completed

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/codecoogs/gogo/constants"
	"github.com/supabase-community/supabase-go"
)

type TodoStatus struct {
	Completed bool      `json:"completed"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    []TodoStatus        `json:"data,omitempty"`
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
		crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "An 'id' paramter is required",
			},
		})
	} else {
		switch r.Method {
		case "PATCH":
			var updatedTodoStatus TodoStatus
			if err := json.NewDecoder(r.Body).Decode(&updatedTodoStatus); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: err.Error(),
					},
				})
				return
			}

			count, err := updateTodoStatus(client, id, updatedTodoStatus)
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
						Message: "Todo not found",
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

func updateTodoStatus(client *supabase.Client, id string, todoStatus TodoStatus) (int64, error) {
	_, count, err := client.From(constants.TODO_TABLE).Update(todoStatus, "", "exact").Eq("id", id).Execute()
	if err != nil {
		return 0, err
	}
	return count, nil
}