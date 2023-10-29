package hello

import (
  "net/http"
)

func Handler(w http.ResponseWriter, r *http.Request) {

  if r.URL.Path == "/v1/hello" {
    w.Write([]byte("Hello from Go in Vercel!"))
  }
}