package payments

import (
	"encoding/json"
	"net/http"
	"github.com/codecoogs/gogo/wrappers/http"
	"github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/google/uuid"
)

type Payment struct {
  ID uuid.UUID `json:"id"`
  Payer uuid.UUID `json:"payer"`
  Payee uuid.UUID `json:"payee"`
  Name string `json:"name"`
  Price int `json:"price"`
  Quantity int `json:"quantity"`
  Description string `json:"description"`
  Method string `json:"method"`
  Expiration *string `json:"expiration"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
  crw := &codecoogshttp.ResponseWriter{W: w}
  crw.SetCors(r.Host)
  
  client, err := codecoogssupabase.CreateClient()
  if err != nil {
    crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to create Supabase client: " + err.Error()})
    return
  }

  // TODO:handle error if no id 
  id := r.URL.Query().Get("id")

  switch r.Method {
  case "GET":
    var payment []Payment
    if _, err := client.From("Payment").Select("*", "exact", false).Eq("id", id).ExecuteTo(&payment); err != nil {
      crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to get payment: " + err.Error()})
      return
    }
    crw.SendJSONResponse(http.StatusOK, map[string]interface{}{"data": payment})
  case "PUT":
    var updatedPayment Payment
      if err := json.NewDecoder(r.Body).Decode(&updatedPayment); err != nil {
        crw.SendJSONResponse(http.StatusBadRequest, map[string]string{"message": "Invalid request body: " + err.Error()})
        return
      }

      if _, _, err := client.From("Payment").Update(updatedPayment, "", "exact").Eq("id", id).Execute(); err != nil {
          crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to update payment: " + err.Error()})
          return
      }
      crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully updated payment with id: " + id})
  case "DELETE":
    if _, _, err := client.From("Payment").Delete("", "exact").Eq("id", id).Execute(); err != nil {
      crw.SendJSONResponse(http.StatusInternalServerError, map[string]string{"message": "Failed to delete payment: " + err.Error()})
      return
    }
    crw.SendJSONResponse(http.StatusOK, map[string]string{"message": "Successfully deleted payment with id: " + id})
  default:
    crw.SendJSONResponse(http.StatusMethodNotAllowed, map[string]string{"message": "Method not allowed for this resource"})
  }
}