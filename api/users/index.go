package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
)

type User struct {
	ID                 *uuid.UUID `json:"id,omitempty"`
	FirstName          string     `json:"first_name"`
	LastName           string     `json:"last_name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	Password           string     `json:"password"`
	Major              string     `json:"major"`
	Classification     string     `json:"classification"`
	ExpectedGraduation string     `json:"expected_graduation"`
	Membership         string     `json:"membership"`
	Paid               bool       `json:"paid"`

	Discord *string    `json:"discord"`
	Team    *uuid.UUID `json:"team"`
	Points  int        `json:"points"`
}

type Response struct {
	Success   bool          `json:"success"`
	StripeURL string        `json:"url"`
	Data      []User        `json:"data,omitempty"`
	Error     *ErrorDetails `json:"error,omitempty"`
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
		case "POST":
			var user User

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
			user.Paid = false

			// if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			// 	crw.SendJSONResponse(http.StatusBadRequest, Response{
			// 		Success: false,
			// 		Error: &ErrorDetails{
			// 			Message: "Invalid request body: " + err.Error(),
			// 		},
			// 	})
			// 	return
			// }

			// remove row "resume" from form data dictionary
			// add row "paid" (boolean value) into the data dictionary

			if _, _, err := client.From(constants.USER_TABLE).Delete("", "").Eq("email", user.Email).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete existing user: " + err.Error(),
					},
				})
			}

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

			var priceID string
			if user.Membership == "Semester" {
				priceID = "price_1Pkp2WRuQxKvYvnuBdqcFUcm"
			} else {
				priceID = "price_1Pkp2WRuQxKvYvnu0GLPeuEE"
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
