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

// PointTransaction matches point_transactions table
type PointTransaction struct {
	ID             *uuid.UUID `json:"id,omitempty"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	CategoryID     *uuid.UUID `json:"category_id,omitempty"`
	EventID        *int64     `json:"event_id,omitempty"`
	PointsEarned   *int       `json:"points_earned,omitempty"`
	CreatedAt      string     `json:"created_at,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	AcademicYearID *uuid.UUID `json:"academic_year_id,omitempty"`
}

// PointCategory matches point_categories table
type PointCategory struct {
	ID          *uuid.UUID `json:"id,omitempty"`
	Name        string     `json:"name"`
	PointsValue int        `json:"points_value"`
	Description *string    `json:"description,omitempty"`
}

type Response struct {
	Success            bool               `json:"success"`
	Data               *UserPoints        `json:"data,omitempty"`
	PointTransactions  []PointTransaction `json:"point_transactions,omitempty"`
	PointCategories    []PointCategory    `json:"point_categories,omitempty"`
	Error              *ErrorDetails      `json:"error,omitempty"`
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
	categories := r.URL.Query().Get("categories")
	transactions := r.URL.Query().Get("transactions")

	column, value := getColumnAndValue(id, email, discordId)

	switch r.Method {
	case "GET":
		// GET point categories (no user id required)
		if categories == "true" {
			cats, err := getPointCategories(client)
			if err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error:   &ErrorDetails{Message: err.Error()},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success:         true,
				PointCategories: cats,
			})
			return
		}

		// GET user's point transactions (requires id, email, or discordId)
		if transactions == "true" {
			if column == "" || value == "" {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error:   &ErrorDetails{Message: "Provide 'id', 'email', or 'discordId' to get point transactions."},
				})
				return
			}
			txns, err := getUserPointTransactions(client, column, value)
			if err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error:   &ErrorDetails{Message: err.Error()},
				})
				return
			}
			if txns == nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error:   &ErrorDetails{Message: "User not found."},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success:           true,
				PointTransactions: txns,
			})
			return
		}

		// GET user name + total points (existing behavior)
		if column == "" || value == "" {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error:   &ErrorDetails{Message: "Provide either 'id', 'email' or 'discordId' as a parameter."},
			})
			return
		}
		userPoints, err := getNameAndPointsByColumn(client, column, value)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error:   &ErrorDetails{Message: err.Error()},
			})
			return
		}
		if userPoints == nil {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error:   &ErrorDetails{Message: "User not found"},
			})
			return
		}
		crw.SendJSONResponse(http.StatusOK, Response{
			Success: true,
			Data:    userPoints,
		})
	case "PATCH":
		if column == "" || value == "" {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error:   &ErrorDetails{Message: "Provide either 'id', 'email' or 'discordId' as a parameter."},
			})
			return
		}
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

func getPointCategories(client *supabase.Client) ([]PointCategory, error) {
	var cats []PointCategory
	_, err := client.From(constants.POINT_CATEGORIES_TABLE).Select("id, name, points_value, description", "exact", false).ExecuteTo(&cats)
	if err != nil {
		return nil, err
	}
	if cats == nil {
		return []PointCategory{}, nil
	}
	return cats, nil
}

// getUserPointTransactions returns point_transactions for the user identified by id, email, or discord.
// We explicitly join users and point_transactions: first resolve the user from users by the given
// identifier (id, email, or discord), then fetch only transactions where point_transactions.user_id
// equals that user's id. This ensures we never return transactions for a different user when
// looking up by email or discord.
func getUserPointTransactions(client *supabase.Client, column string, value string) ([]PointTransaction, error) {
	// Step 1: Resolve user from users table by the provided identifier (id, email, or discord).
	// This is the "join" condition: we only consider the single user that matches.
	if column != "id" && column != "email" && column != "discord" {
		return nil, nil
	}
	userID, _, _, err := getUserIDAndName(client, column, value)
	if err != nil {
		return nil, err
	}
	if userID == nil {
		return nil, nil
	}
	// Step 2: Get point_transactions only for this user (join on user_id = users.id).
	var txns []PointTransaction
	_, err = client.From(constants.POINT_TRANSACTIONS_TABLE).
		Select("id, user_id, category_id, event_id, points_earned, created_at, created_by, academic_year_id", "exact", false).
		Eq("user_id", userID.String()).
		ExecuteTo(&txns)
	if err != nil {
		return nil, err
	}
	if txns == nil {
		return []PointTransaction{}, nil
	}
	return txns, nil
}
