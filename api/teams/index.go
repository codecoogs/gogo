package teams

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
  id := r.URL.Query().Get("id")

  if id == "" {
    switch r.Method {
    case "POST":
      w.Write([]byte("POST request to teams"))
    default:
      w.Write([]byte("Invalid request type"))
    }
  } else {
    switch r.Method {
    case "GET":
      w.Write([]byte("GET request to teams with id: " + id))
    case "PUT":
      w.Write([]byte("PUT request to teams with id: " + id))
    case "DELETE":
      w.Write([]byte("DELETE request to teams with id: " + id))
    default:
      w.Write([]byte("Invalid request type"))
    }
  }
}