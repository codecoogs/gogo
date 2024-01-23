package payments

import (
	"encoding/json"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
	"net/http"
)

type Payment struct {
	ID          uuid.UUID `json:"id"`
	Payer       uuid.UUID `json:"payer"`
	Payee       uuid.UUID `json:"payee"`
	Name        string    `json:"name"`
	Price       int       `json:"price"`
	Quantity    int       `json:"quantity"`
	Description string    `json:"description"`
	Method      string    `json:"method"`
	Expiration  *string   `json:"expiration"`
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Payment     `json:"data,omitempty"`
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

	// TODO:handle error if no id
	id := r.URL.Query().Get("id")

	if id == "" {
		switch r.Method {
		case "POST":
			var payment Payment
			if err := json.NewDecoder(r.Body).Decode(&payment); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From("Payment").Insert(payment, false, "", "", "exact").Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create payment: " + err.Error(),
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
	} else {
		switch r.Method {
		case "GET":
			var payment []Payment
			if _, err := client.From("Payment").Select("*", "exact", false).Eq("id", id).ExecuteTo(&payment); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get payment: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    payment,
			})
		case "PUT":
			var updatedPayment Payment
			if err := json.NewDecoder(r.Body).Decode(&updatedPayment); err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From("Payment").Update(updatedPayment, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update payment: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			if _, _, err := client.From("Payment").Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete payment: " + err.Error(),
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