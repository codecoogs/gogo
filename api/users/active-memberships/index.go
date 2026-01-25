package users

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"
)

type ActiveMember struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership         string     `json:"membership"`
	Discord            string     `json:"discord"`
	Team               *uuid.UUID `json:"team"`
	Points             int        `json:"points"`
	DueDate            string     `json:"due_date"`
	LastPaymentDate    string     `json:"last_payment_date,omitempty"`
}

type PaymentWithDate struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	Payer     uuid.UUID  `json:"payer"`
	Created   string     `json:"created"`
	Expiration *string   `json:"expiration,omitempty"`
}

type Response struct {
	Success bool            `json:"success"`
	Data    []ActiveMember  `json:"data,omitempty"`
	Error   *ErrorDetails   `json:"error,omitempty"`
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
	case "GET":
		activeMembers, err := getActiveMembers(client)
		if err != nil {
			crw.SendJSONResponse(http.StatusInternalServerError, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Failed to get active members: " + err.Error(),
				},
			})
			return
		}

		crw.SendJSONResponse(http.StatusOK, Response{
			Success: true,
			Data:    activeMembers,
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

type UserQuery struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership         string     `json:"membership"`
	Paid               bool       `json:"paid"`
	Discord            string     `json:"discord"`
	Team               *uuid.UUID `json:"team"`
	Points             int        `json:"points"`
	Updated            string     `json:"updated"`
}

func getActiveMembers(client *supabase.Client) ([]ActiveMember, error) {
	// Get all users with Yearly or Semester membership
	var allUsers []UserQuery
	_, err := client.From(constants.USER_TABLE).
		Select("id, first_name, last_name, email, phone, major, classification, expected_graduation, membership, paid, discord, team, points, updated", "exact", false).
		In("membership", []string{"Yearly", "Semester"}).
		ExecuteTo(&allUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(allUsers) == 0 {
		return []ActiveMember{}, nil
	}

	// Get all payments to find most recent payment for each user
	var allPayments []PaymentWithDate
	_, err = client.From(constants.PAYMENT_TABLE).
		Select("id, payer, created, expiration", "exact", false).
		ExecuteTo(&allPayments)
	if err != nil {
		// If payments table doesn't have created field, try without it
		_, err = client.From(constants.PAYMENT_TABLE).
			Select("id, payer, expiration", "exact", false).
			ExecuteTo(&allPayments)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch payments: %w", err)
		}
	}

	// Create a map of user ID to most recent payment date
	userPaymentMap := make(map[uuid.UUID]time.Time)
	for _, payment := range allPayments {
		var paymentDate time.Time
		var err error

		// Try to use created field first, then expiration, then skip if neither works
		if payment.Created != "" {
			paymentDate, err = parsePaymentDate(payment.Created)
			if err != nil {
				continue
			}
		} else if payment.Expiration != nil && *payment.Expiration != "" {
			// If expiration exists, it might be the payment date
			paymentDate, err = parsePaymentDate(*payment.Expiration)
			if err != nil {
				continue
			}
		} else {
			continue
		}

		// Keep the most recent payment date for each user
		if existingDate, exists := userPaymentMap[payment.Payer]; !exists || paymentDate.After(existingDate) {
			userPaymentMap[payment.Payer] = paymentDate
		}
	}

	now := time.Now().UTC()
	var activeMembers []ActiveMember

	for _, user := range allUsers {
		if user.ID == nil {
			continue
		}

		// Get the most recent payment date for this user
		lastPaymentDate, hasPayment := userPaymentMap[*user.ID]
		
		// If user has no payment record but has paid=true, use updated timestamp as fallback
		if !hasPayment && user.Paid {
			updatedTime, err := parsePaymentDate(user.Updated)
			if err != nil {
				// Skip users where we can't parse the updated timestamp
				continue
			}
			lastPaymentDate = updatedTime
		} else if !hasPayment {
			// Skip users with no payment record
			continue
		}

		// Calculate due date based on membership type
		var dueDate time.Time
		if user.Membership == "Yearly" {
			dueDate = lastPaymentDate.AddDate(1, 0, 0) // Add 1 year
		} else if user.Membership == "Semester" {
			dueDate = lastPaymentDate.AddDate(0, 6, 0) // Add 6 months
		} else {
			continue
		}

		// Only include members whose membership hasn't expired
		if dueDate.After(now) {
			activeMember := ActiveMember{
				ID:                 user.ID,
				FirstName:          user.FirstName,
				LastName:           user.LastName,
				Email:              user.Email,
				Phone:              user.Phone,
				Major:              user.Major,
				Classification:     user.Classification,
				ExpectedGraduation: user.ExpectedGraduation,
				Membership:         user.Membership,
				Discord:            user.Discord,
				Team:               user.Team,
				Points:             user.Points,
				DueDate:            dueDate.Format(time.RFC3339),
				LastPaymentDate:    lastPaymentDate.Format(time.RFC3339),
			}
			activeMembers = append(activeMembers, activeMember)
		}
	}

	return activeMembers, nil
}

func parsePaymentDate(dateStr string) (time.Time, error) {
	// Try different date formats that Supabase might use
	formats := []string{
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05.999999Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, strings.TrimSpace(dateStr)); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
