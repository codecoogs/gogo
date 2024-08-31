package users

import (
	"net/http"
	"strconv"

	"github.com/codecoogs/gogo/constants"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase-community/supabase-go"
	"github.com/supabase/postgrest-go"
)

type LeaderboardUser struct {
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Points    int     `json:"points"`
	Discord   *string `json:"discord"`
}

type Response struct {
	Success bool              `json:"success"`
	Data    []LeaderboardUser `json:"data,omitempty"`
	Error   *ErrorDetails     `json:"error,omitempty"`
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

	top := r.URL.Query().Get("top")
	if top == "" {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide 'top' as a parameter to retrive the 'top' users",
			},
		})
		return
	}

	switch r.Method {
	case "GET":
		leaderboard, err := getLeaderboard(client, top)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: err.Error(),
				},
			})
			return
		}

		if leaderboard == nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Not enough users exist",
				},
			})
			return
		}

		crw.SendJSONResponse(http.StatusOK, Response{
			Success: true,
			Data:    leaderboard,
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

func getLeaderboard(client *supabase.Client, top string) ([]LeaderboardUser, error) {
	var leaderboard []LeaderboardUser
	count, err := strconv.Atoi(top)
	if err != nil {
		return nil, err
	}

	var MyOrderOpts = &postgrest.OrderOpts{
		Ascending:    false,
		NullsFirst:   false,
		ForeignTable: "",
	}

	if _, err := client.From(constants.USER_TABLE).Select("first_name, last_name, points, discord", "exact", false).Order("points", MyOrderOpts).Limit(count, "").ExecuteTo(&leaderboard); err != nil {
		return nil, err
	}
	if len(leaderboard) == 0 {
		return nil, nil
	}

	return leaderboard, nil
}
