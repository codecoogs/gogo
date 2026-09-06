package codecoogssupabase

import (
	"net/http"
	"strings"
)

// Postgres rejections that mean the request was wrong, not the server. The
// database stays the single source of truth for what each column allows, so
// these are recognised by SQLSTATE rather than by re-listing the constraints
// here — a value added to a CHECK in a migration needs no change in Go.
var clientErrors = map[string]struct {
	status  int
	message string
}{
	// check_violation
	"23514": {http.StatusBadRequest, "One or more fields have a value that is not allowed"},
	// unique_violation
	"23505": {http.StatusConflict, "A record with these values already exists"},
	// foreign_key_violation
	"23503": {http.StatusBadRequest, "A referenced record does not exist"},
	// not_null_violation
	"23502": {http.StatusBadRequest, "A required field is missing"},
	// invalid_text_representation
	"22P02": {http.StatusBadRequest, "One or more fields are malformed"},
}

// sqlstate pulls the code out of a postgrest-go error, which formats failures as
// "(23514) new row for relation ...". Returns "" for anything else.
func sqlstate(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	if !strings.HasPrefix(message, "(") {
		return ""
	}

	end := strings.Index(message, ")")
	if end <= 1 {
		return ""
	}

	return message[1:end]
}

// ErrorResponse maps err to an HTTP status and a message safe to return to
// callers. Constraint names and raw Postgres text describe internal schema, so
// caller mistakes get a generic message; genuine server failures keep the detail
// (prefixed with action) because someone has to debug them.
func ErrorResponse(err error, action string) (int, string) {
	if clientError, ok := clientErrors[sqlstate(err)]; ok {
		return clientError.status, clientError.message
	}

	if err == nil {
		return http.StatusInternalServerError, action
	}

	return http.StatusInternalServerError, action + ": " + err.Error()
}
