package points

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
  
	switch r.Method {
	case "GET":
		w.Write([]byte("GET request to points of user with id: " + id))
	case "PATCH":
		w.Write([]byte("PATCH request to points of user with id: " + id))
	default:
		w.Write([]byte("Invalid request type"))
	}
  }