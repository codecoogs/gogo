package codecoogssupabase

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// postgrest-go formats failures as fmt.Errorf("(%s) %s", code, message), so these
// mirror what the handlers actually receive.
func pgErr(code, message string) error {
	return fmt.Errorf("(%s) %s", code, message)
}

func TestErrorResponseStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "check violation is the caller's fault",
			err:  pgErr("23514", `new row for relation "opportunities" violates check constraint "opportunities_category_check"`),
			want: http.StatusBadRequest,
		},
		{
			name: "unique violation is a conflict",
			err:  pgErr("23505", `duplicate key value violates unique constraint "opportunities_external_id_key"`),
			want: http.StatusConflict,
		},
		{
			name: "foreign key violation is the caller's fault",
			err:  pgErr("23503", `insert or update on table "opportunities" violates foreign key constraint "opportunities_linked_form_id_fkey"`),
			want: http.StatusBadRequest,
		},
		{
			name: "not null violation is the caller's fault",
			err:  pgErr("23502", `null value in column "title" violates not-null constraint`),
			want: http.StatusBadRequest,
		},
		{
			name: "malformed input is the caller's fault",
			err:  pgErr("22P02", `invalid input syntax for type uuid: "not-a-uuid"`),
			want: http.StatusBadRequest,
		},
		{
			name: "an unrecognised postgres code is a server problem",
			err:  pgErr("42P01", `relation "nope" does not exist`),
			want: http.StatusInternalServerError,
		},
		{
			name: "a non-postgrest error is a server problem",
			err:  errors.New("dial tcp: connection refused"),
			want: http.StatusInternalServerError,
		},
		{
			name: "an empty code is a server problem",
			err:  pgErr("", "something went wrong"),
			want: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ErrorResponse(tt.err, "Failed to create opportunity")
			if got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

// The whole point of the mapping: a caller mistake must not hand back the schema's
// internal vocabulary.
func TestErrorResponseDoesNotLeakSchemaDetail(t *testing.T) {
	leaky := []error{
		pgErr("23514", `new row for relation "opportunities" violates check constraint "opportunities_category_check"`),
		pgErr("23505", `duplicate key value violates unique constraint "opportunities_external_id_key"`),
		pgErr("23503", `violates foreign key constraint "opportunities_linked_form_id_fkey"`),
		pgErr("23502", `null value in column "title" violates not-null constraint`),
		pgErr("22P02", `invalid input syntax for type uuid: "not-a-uuid"`),
	}

	forbidden := []string{"constraint", "relation", "opportunities_", "_fkey", "_key", "null value", "23514", "23505"}

	for _, err := range leaky {
		_, msg := ErrorResponse(err, "Failed to create opportunity")
		lower := strings.ToLower(msg)
		for _, bad := range forbidden {
			if strings.Contains(lower, strings.ToLower(bad)) {
				t.Errorf("message %q leaks %q (from %v)", msg, bad, err)
			}
		}
		if strings.TrimSpace(msg) == "" {
			t.Errorf("message is empty for %v", err)
		}
	}
}

// Server-side failures still need to be debuggable, so those keep the detail.
func TestErrorResponseKeepsDetailForServerErrors(t *testing.T) {
	err := errors.New("dial tcp: connection refused")
	status, msg := ErrorResponse(err, "Failed to create opportunity")

	if status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", status)
	}
	if !strings.Contains(msg, "Failed to create opportunity") {
		t.Errorf("message %q lost the action context", msg)
	}
	if !strings.Contains(msg, "connection refused") {
		t.Errorf("message %q lost the underlying error", msg)
	}
}

func TestErrorResponseNilError(t *testing.T) {
	status, msg := ErrorResponse(nil, "Failed to create opportunity")
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	if strings.TrimSpace(msg) == "" {
		t.Error("message is empty")
	}
}
