package users

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
  id := r.URL.Query().Get("id")

  switch r.Method {
  case "GET":
	  w.Write([]byte("GET request to user with id: " + id))
  case "PUT":
	  w.Write([]byte("PUT request to user with id: " + id))
  case "DELETE":
	  w.Write([]byte("DELETE request to user with id: " + id))
  default:
	  w.Write([]byte("Invalid request type"))
  }
}