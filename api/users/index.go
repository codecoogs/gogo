package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
	"github.com/supabase-community/supabase-go"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
)

type SupabaseTime time.Time

func (st *SupabaseTime) UnmarshalJSON(b []byte) error {
	// Trim the quotes from the JSON string
	s := strings.Trim(string(b), "\"")

	// Parse the time string provided by Supabase
	t, err := time.Parse("2006-01-02T15:04:05.999999", s)
	if err != nil {
		return err
	}

	*st = SupabaseTime(t)
	return nil
}

func (st SupabaseTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", time.Time(st).Format("2006-01-02T15:04:05.999999"))), nil
}

type User struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	AuthID             string     `json:"auth_id"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership         string     `json:"membership"`

	Paid        bool `json:"paid"`
	ShirtBought bool `json:"shirt-bought"`

	Created SupabaseTime `json:"created"`
	Updated SupabaseTime `json:"updated"`

	Discord string `json:"discord"`
}

type ActiveMember struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership         string `json:"membership"`
	Discord            string `json:"discord"`
	DueDate            string `json:"due_date"`
	LastPaymentDate    string `json:"last_payment_date,omitempty"`
}

// UserWithPaymentInfo returns all users with last payment date, next due date, and paid status
type UserWithPaymentInfo struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership        string `json:"membership"`
	Discord           string `json:"discord"`
	Paid              bool   `json:"paid"`
	LastPaymentDate   string `json:"last_payment_date,omitempty"`
	NextDueDate       string `json:"next_due_date,omitempty"`
}

type PaymentWithDate struct {
	ID        *uuid.UUID `json:"id,omitempty"`
	UserID    uuid.UUID  `json:"user_id"`
	CreatedAt string     `json:"created_at"`
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
	Membership string `json:"membership"`
	Paid       bool   `json:"paid"`
	Discord    string `json:"discord"`
	Updated    string `json:"updated"`
}

type Response struct {
	Success           bool                  `json:"success"`
	StripeURL         string                `json:"url"`
	Data              []User                `json:"data,omitempty"`
	ActiveMembers     []ActiveMember        `json:"active_members,omitempty"`
	UsersPaymentInfo  []UserWithPaymentInfo `json:"users_payment_info,omitempty"`
	Error             *ErrorDetails         `json:"error,omitempty"`
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
	activeMemberships := r.URL.Query().Get("active_memberships")
	paymentInfo := r.URL.Query().Get("payment_info")

	if id == "" {
		switch r.Method {
		case "GET":
			// Handle all users with payment info (last payment, next due, paid status)
			if paymentInfo == "true" {
				usersWithPaymentInfo, err := getAllUsersWithPaymentInfo(client)
				if err != nil {
					crw.SendJSONResponse(http.StatusInternalServerError, Response{
						Success: false,
						Error: &ErrorDetails{
							Message: "Failed to get users with payment info: " + err.Error(),
						},
					})
					return
				}

				crw.SendJSONResponse(http.StatusOK, Response{
					Success:          true,
					UsersPaymentInfo: usersWithPaymentInfo,
				})
				return
			}
			// Handle active memberships query
			if activeMemberships == "true" {
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
					Success:       true,
					ActiveMembers: activeMembers,
				})
				return
			}
			// Default GET behavior (could be implemented later)
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed. Use ?active_memberships=true to get active members, or ?payment_info=true to get all users with payment/due dates.",
				},
			})
		case "POST":
			var user User
			var existingUsers []User

			if err := r.ParseMultipartForm(32 << 20); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request form data: " + err.Error(),
					},
				})
				return
			}

			user.FirstName = r.FormValue("first_name")
			user.LastName = r.FormValue("last_name")
			user.Email = r.FormValue("email")
			user.Phone = r.FormValue("phone")
			user.Major = r.FormValue("major")
			user.Classification = r.FormValue("classification")
			user.ExpectedGraduation = r.FormValue("expected_graduation")
			user.Discord = r.FormValue("discord")
			user.Membership = r.FormValue("membership")

			count, err := client.From(constants.USER_TABLE).Select("*", "exact", false).Eq("email", user.Email).ExecuteTo(&existingUsers)
			fmt.Println(user)

			if err == nil {
				fmt.Println(count)

				if count == 0 {
					fmt.Println("Creating new member...")

					user.Created = SupabaseTime(time.Now().UTC())
					user.Updated = SupabaseTime(time.Now().UTC())

					if _, _, err := client.From(constants.USER_TABLE).Insert(user, false, "", "", "exact").Execute(); err != nil {
						crw.SendJSONResponse(http.StatusInternalServerError, Response{
							Success: false,
							Error: &ErrorDetails{
								Message: "Failed to create user: " + err.Error(),
							},
						})
						fmt.Println(err)
						return
					}
				} else {
					fmt.Println("Member already exists in database")

					var existingUser = existingUsers[0]

					existingUser.FirstName = user.FirstName
					existingUser.LastName = user.LastName
					existingUser.Phone = user.Phone
					existingUser.Major = user.Major
					existingUser.Classification = user.Classification
					existingUser.ExpectedGraduation = user.ExpectedGraduation
					existingUser.Discord = user.Discord

					existingUser.Updated = SupabaseTime(time.Now().UTC())

					if _, _, err := client.From(constants.USER_TABLE).Update(existingUser, "", "exact").Eq("email", existingUser.Email).Execute(); err != nil {
						crw.SendJSONResponse(http.StatusInternalServerError, Response{
							Success: false,
							Error: &ErrorDetails{
								Message: "Failed to update user: " + err.Error(),
							},
						})
						return
					}
				}
			} else {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to select users: " + err.Error(),
					},
				})
				return
			}

			var priceID string
			if user.Membership == "Semester" {
				priceID = "price_1S0vgqRuQxKvYvnuVBHgkIip" // stripe id for semester
			} else {
				priceID = "price_1S0vgqRuQxKvYvnusQgsCBSb" // stripe id for yearly
			}

			stripe.Key = os.Getenv("STRIPE_SK")
			params := &stripe.CheckoutSessionParams{
				SuccessURL:       stripe.String("https://www.codecoogs.com/success"),
				CustomerCreation: stripe.String(string(stripe.CheckoutSessionCustomerCreationAlways)),
				CustomerEmail:    &user.Email,
				LineItems: []*stripe.CheckoutSessionLineItemParams{
					&stripe.CheckoutSessionLineItemParams{
						Price:    stripe.String(priceID),
						Quantity: stripe.Int64(1),
					},
				},
				Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
				Metadata: map[string]string{
					"CustomerPhone": user.Phone,
				},
			}

			result, err := session.New(params)
			if err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create Stripe Session: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success:   true,
				StripeURL: result.URL,
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
		switch r.Method {
		case "GET":
			var user []User
			if _, err := client.From(constants.USER_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&user); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get user: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    user,
			})
		case "PUT":
			var updatedUser User
			if err := json.NewDecoder(r.Body).Decode(&updatedUser); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}
			// TODO: Perform validation checks on updatedUser data

			if _, _, err := client.From(constants.USER_TABLE).Update(updatedUser, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update user: " + err.Error(),
					},
				})
				return
			}

			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			if _, _, err := client.From(constants.USER_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete users: " + err.Error(),
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

func getActiveMembers(client *supabase.Client) ([]ActiveMember, error) {
	// Get all users with Yearly or Semester membership
	var allUsers []UserQuery
	_, err := client.From(constants.USER_TABLE).
		Select("id, first_name, last_name, email, phone, major, classification, expected_graduation, membership, paid, discord, updated", "exact", false).
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
		Select("id, user_id, created_at", "exact", false).
		ExecuteTo(&allPayments)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments: %w", err)
	}

	// Create a map of user ID to most recent payment date
	userPaymentMap := make(map[uuid.UUID]time.Time)
	for _, payment := range allPayments {
		// Parse the created_at timestamp
		if payment.CreatedAt == "" {
			continue
		}

		paymentDate, err := parsePaymentDate(payment.CreatedAt)
		if err != nil {
			// Skip payments where we can't parse the date
			continue
		}

		// Keep the most recent payment date for each user
		if existingDate, exists := userPaymentMap[payment.UserID]; !exists || paymentDate.After(existingDate) {
			userPaymentMap[payment.UserID] = paymentDate
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
				DueDate:            dueDate.Format(time.RFC3339),
				LastPaymentDate:    lastPaymentDate.Format(time.RFC3339),
			}
			activeMembers = append(activeMembers, activeMember)
		}
	}

	return activeMembers, nil
}

// getAllUsersWithPaymentInfo returns all users with last payment date, next due date (for Yearly/Semester), and paid status
func getAllUsersWithPaymentInfo(client *supabase.Client) ([]UserWithPaymentInfo, error) {
	var allUsers []UserQuery
	_, err := client.From(constants.USER_TABLE).
		Select("id, first_name, last_name, email, phone, major, classification, expected_graduation, membership, paid, discord, updated", "exact", false).
		ExecuteTo(&allUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	if len(allUsers) == 0 {
		return []UserWithPaymentInfo{}, nil
	}

	var allPayments []PaymentWithDate
	_, err = client.From(constants.PAYMENT_TABLE).
		Select("id, user_id, created_at", "exact", false).
		ExecuteTo(&allPayments)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payments: %w", err)
	}

	userPaymentMap := make(map[uuid.UUID]time.Time)
	for _, payment := range allPayments {
		if payment.CreatedAt == "" {
			continue
		}
		paymentDate, err := parsePaymentDate(payment.CreatedAt)
		if err != nil {
			continue
		}
		if existingDate, exists := userPaymentMap[payment.UserID]; !exists || paymentDate.After(existingDate) {
			userPaymentMap[payment.UserID] = paymentDate
		}
	}

	var result []UserWithPaymentInfo
	for _, user := range allUsers {
		if user.ID == nil {
			continue
		}

		lastPaymentDate, hasPayment := userPaymentMap[*user.ID]
		if !hasPayment && user.Paid {
			updatedTime, err := parsePaymentDate(user.Updated)
			if err == nil {
				lastPaymentDate = updatedTime
				hasPayment = true
			}
		}

		lastPaymentStr := ""
		if hasPayment {
			lastPaymentStr = lastPaymentDate.Format(time.RFC3339)
		}

		nextDueStr := ""
		if user.Membership == "Yearly" && hasPayment {
			nextDueStr = lastPaymentDate.AddDate(1, 0, 0).Format(time.RFC3339)
		} else if user.Membership == "Semester" && hasPayment {
			nextDueStr = lastPaymentDate.AddDate(0, 6, 0).Format(time.RFC3339)
		}

		result = append(result, UserWithPaymentInfo{
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
			Paid:               user.Paid,
			LastPaymentDate:    lastPaymentStr,
			NextDueDate:        nextDueStr,
		})
	}

	return result, nil
}

func parsePaymentDate(dateStr string) (time.Time, error) {
	// Try different date formats that Supabase/PostgreSQL might use
	formats := []string{
		time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
		time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05.999999999Z",
		"2006-01-02T15:04:05.999999Z",
		"2006-01-02T15:04:05.999999+00:00",
		"2006-01-02T15:04:05.999999+00",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+00:00",
		"2006-01-02T15:04:05+00",
		"2006-01-02T15:04:05",
	}

	dateStr = strings.TrimSpace(dateStr)
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
