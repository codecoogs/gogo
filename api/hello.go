package handler

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {

  if r.URL.Path == "/hello" {
    w.Write([]byte("Hello from Go in Vercel!"))
    return
  }

  http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func handler() {
  http.HandleFunc("/", Handler)
  http.ListenAndServe(":8080", nil) 
}