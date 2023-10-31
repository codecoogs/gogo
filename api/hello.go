package hello

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("Welcome to Code[Coogs] GoGo v1!"))
}