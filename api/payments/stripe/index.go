package stripe

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/codecoogs/gogo/constants"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	codecoogsemail "github.com/codecoogs/gogo/wrappers/email"
	"github.com/google/uuid"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/webhook"
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

func Handler(w http.ResponseWriter, req *http.Request) {
	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		fmt.Println("Failed to create Supabase client: " + err.Error())
		return
	}

	const MaxBodyBytes = int64(65536)
	req.Body = http.MaxBytesReader(w, req.Body, MaxBodyBytes)
	payload, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	fmt.Println("testing!")

	event := stripe.Event{}

	if err := json.Unmarshal(payload, &event); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Webhook error while parsing basic request. %v\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Replace this endpoint secret with your endpoint's unique secret
	// If you are testing with the CLI, find the secret by running 'stripe listen'
	// If you are using an endpoint defined with the API or dashboard, look in your webhook settings
	// at https://dashboard.stripe.com/webhooks
	endpointSecret := os.Getenv("STRIPE_WH")
	signatureHeader := req.Header.Get("Stripe-Signature")
	event, err = webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Webhook signature verification failed. %v\n", err)
		w.WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
		return
	}
	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "checkout.session.completed":
		var checkoutSession stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &checkoutSession)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Printf("Successful payment for %d.", checkoutSession.AmountTotal)
		fmt.Println(checkoutSession.CustomerEmail)

		row := map[string]interface{}{
			"paid": true,
		}

		if _, _, err := client.From(constants.USER_TABLE).Update(row, "", "exact").Eq("email", checkoutSession.CustomerEmail).Execute(); err != nil {
			fmt.Fprintf(os.Stderr, "Error updating user: %v\n", err)
			return
		}

		log.Printf("Successfully updated paid status for %s.", checkoutSession.CustomerEmail)

		codecoogsemail.SendEmail(checkoutSession.CustomerEmail, "Semester")

		// fmt.Println(checkoutSession.Metadata)

		// Then define and call a func to handle the successful payment intent.
		// handlePaymentIntentSucceeded(paymentIntent)
	case "payment_method.attached":
		var paymentMethod stripe.PaymentMethod
		err := json.Unmarshal(event.Data.Raw, &paymentMethod)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing webhook JSON: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Then define and call a func to handle the successful attachment of a PaymentMethod.
		// handlePaymentMethodAttached(paymentMethod)
	default:
		// fmt.Fprintf(os.Stderr, "Unhandled event type: %s\n", event.Type)
	}

	w.WriteHeader(http.StatusOK)
}
