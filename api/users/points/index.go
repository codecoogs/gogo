package users

import (
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

		// GET user name + total points (total derived from point_transactions only; frontend can also sum from ?transactions=true)
		if column == "" || value == "" {
			crw.SendJSONResponse(http.StatusBadRequest, Response{
				Success: false,
				Error:   &ErrorDetails{Message: "Provide either 'id', 'email' or 'discordId' as a parameter."},
			})
			return
		}
		userPoints, err := getNameAndPointsFromTransactions(client, column, value)
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

// getNameAndPointsFromTransactions returns user name and total points using only point_transactions (no points table).
// Total is the sum of points_earned for that user; frontend can also fetch ?transactions=true and sum there.
func getNameAndPointsFromTransactions(client *supabase.Client, column string, value string) (*UserPoints, error) {
	userID, firstName, lastName, err := getUserIDAndName(client, column, value)
	if err != nil {
		return nil, err
	}
	if userID == nil {
		return nil, nil
	}

	var txns []PointTransaction
	_, err = client.From(constants.POINT_TRANSACTIONS_TABLE).
		Select("points_earned", "exact", false).
		Eq("user_id", userID.String()).
		ExecuteTo(&txns)
	if err != nil {
		return nil, err
	}

	total := 0
	for _, t := range txns {
		if t.PointsEarned != nil {
			total += *t.PointsEarned
		}
	}

	return &UserPoints{
		FirstName: firstName,
		LastName:  lastName,
		Points:    total,
	}, nil
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
