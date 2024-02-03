package todos

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/codecoogs/gogo/constants"
	"github.com/supabase/postgrest-go"
)

type Todo struct {
	ID        int    `json:"id,omitempty"`
	Title     string `json:"title"`
	Deadline  string `json:"deadline"`
	Completed bool   `json:"completed"`
}

type TodoUser struct {
	TodoID    int    `json:"todo_id"`
	DiscordID string `json:"discord_id"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Todo        `json:"data,omitempty"`
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
	discord_id := r.URL.Query().Get("discord_id")

	if id == "" && discord_id == "" {
		switch r.Method {
		case "GET":
			var MyOrderOpts = &postgrest.OrderOpts{
				Ascending:    true,
				NullsFirst:   false,
				ForeignTable: "",
			}
			var todo []Todo
			if _, err := client.From(constants.TODO_TABLE).Select("*", "exact", false).Order("deadline", MyOrderOpts).ExecuteTo(&todo); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get todo: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    todo,
			})
		default:
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed for this resource",
				},
			})
		}
	} else if id != "" {
		switch r.Method {
		case "GET":
			var todo []Todo
			if _, err := client.From(constants.TODO_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&todo); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get todo: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    todo,
			})
		case "PUT":
			var updatedTodo Todo
			if err := json.NewDecoder(r.Body).Decode(&updatedTodo); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.TODO_TABLE).Update(updatedTodo, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update todo: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			if _, _, err := client.From(constants.TODO_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete todo: " + err.Error(),
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
	} else if discord_id != "" {
		switch r.Method {
		case "GET":
			var MyOrderOpts = &postgrest.OrderOpts{
				Ascending:    true,
				NullsFirst:   false,
				ForeignTable: "",
			}
			var todo []Todo
			if _, err := client.From(constants.TODO_TABLE).Select("*, " + constants.TODO_USER_TABLE + " !inner(todo_id, discord_id)(id)", "exact", false).Filter(constants.TODO_USER_TABLE + ".discord_id", "eq", discord_id).Order("deadline", MyOrderOpts).ExecuteTo(&todo); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get todo from user: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    todo,
			})
		case "POST":
			var newTodo Todo
			if err := json.NewDecoder(r.Body).Decode(&newTodo); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			res, _, err := client.From(constants.TODO_TABLE).Insert(newTodo, false, "", "", "exact").Execute()
			if err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create todo: " + err.Error(),
					},
				})
				return
			}

			var todos []map[string]interface{}
			err = json.Unmarshal(res, &todos)
			if err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to unmarshal response: " + err.Error(),
					},
				})
				return
			}

			var todo_id = int(todos[0]["id"].(float64))
			todoUser := TodoUser{
				TodoID:    todo_id,
				DiscordID: discord_id,
			}
			if _, _, err := client.From(constants.TODO_USER_TABLE).Insert(todoUser, false, "", "", "exact").Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to assing todo to a user: " + err.Error(),
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
