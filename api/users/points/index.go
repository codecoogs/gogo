package users

import (
	"encoding/json"
	"net/http"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type UserPoints struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Points    int    `json:"points"`
}

// PointsRow matches the points table (column "userID" per schema)
type PointsRow struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	UserID    uuid.UUID  `json:"userID"`
	Points    *int       `json:"points"`
	CreatedAt string     `json:"created_at"`
}

type Response struct {
	Success bool        `json:"success"`
	Data    *UserPoints `json:"data,omitempty"`
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

	column, value := getColumnAndValue(id, email, discordId)
	if column == "" && value == "" {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Provide either 'id', 'email' or 'discordId' as a parameter",
			},
		})
		return
	}

	switch r.Method {
	case "GET":
		userPoints, err := getNameAndPointsByColumn(client, column, value)
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
			Data:    userPoints,
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

		count, err := updateUserPoints(client, column, value, updatedUserPoints.Points)
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

func getColumnAndValue(id string, email string, discordId string) (string, string) {
	if id != "" {
		return "id", id
	}
	if email != "" {
		return "email", email
	}
	if discordId != "" {
		return "discord", discordId
	}
	return "", ""
}

// getUserIDAndName fetches user id, first_name, last_name from users table by column (id, email, or discord)
func getUserIDAndName(client *supabase.Client, column string, value string) (userID *uuid.UUID, firstName string, lastName string, err error) {
	var rows []struct {
		ID        *uuid.UUID `json:"id"`
		FirstName string     `json:"first_name"`
		LastName  string     `json:"last_name"`
	}
	_, err = client.From(constants.USER_TABLE).Select("id, first_name, last_name", "exact", false).Eq(column, value).ExecuteTo(&rows)
	if err != nil {
		return nil, "", "", err
	}
	if len(rows) == 0 {
		return nil, "", "", nil
	}
	return rows[0].ID, rows[0].FirstName, rows[0].LastName, nil
}

func getNameAndPointsByColumn(client *supabase.Client, column string, value string) (*UserPoints, error) {
	userID, firstName, lastName, err := getUserIDAndName(client, column, value)
	if err != nil {
		return nil, err
	}
	if userID == nil {
		return nil, nil
	}

	// Query points table for this user (column "userID" per schema)
	var pointsRows []PointsRow
	_, err = client.From(constants.POINTS_TABLE).Select("id, userID, points, created_at", "exact", false).Eq("userID", userID.String()).ExecuteTo(&pointsRows)
	if err != nil {
		return nil, err
	}

	pointsVal := 0
	if len(pointsRows) > 0 {
		// Use latest row by created_at (last in list if ordered; otherwise take first)
		latest := pointsRows[0]
		for i := 1; i < len(pointsRows); i++ {
			if pointsRows[i].CreatedAt > latest.CreatedAt {
				latest = pointsRows[i]
			}
		}
		if latest.Points != nil {
			pointsVal = *latest.Points
		}
	}

	return &UserPoints{
		FirstName: firstName,
		LastName:  lastName,
		Points:    pointsVal,
	}, nil
}

func updateUserPoints(client *supabase.Client, column string, value string, points int) (int64, error) {
	userID, _, _, err := getUserIDAndName(client, column, value)
	if err != nil {
		return 0, err
	}
	if userID == nil {
		return 0, nil
	}

	// Get existing points row(s) for this user
	var pointsRows []PointsRow
	_, err = client.From(constants.POINTS_TABLE).Select("id, userID, points, created_at", "exact", false).Eq("userID", userID.String()).ExecuteTo(&pointsRows)
	if err != nil {
		return 0, err
	}

	if len(pointsRows) > 0 {
		// Update the latest row
		latest := pointsRows[0]
		for i := 1; i < len(pointsRows); i++ {
			if pointsRows[i].CreatedAt > latest.CreatedAt {
				latest = pointsRows[i]
			}
		}
		if latest.ID != nil {
			_, count, err := client.From(constants.POINTS_TABLE).Update(map[string]interface{}{"points": points}, "", "exact").Eq("id", latest.ID.String()).Execute()
			if err != nil {
				return 0, err
			}
			return count, nil
		}
	}

	// No row exists: insert new points row
	row := map[string]interface{}{
		"userID": userID.String(),
		"points": points,
	}
	_, _, err = client.From(constants.POINTS_TABLE).Insert(row, false, "", "", "exact").Execute()
	if err != nil {
		return 0, err
	}
	return 1, nil
}
