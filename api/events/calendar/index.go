package calendar

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/codecoogs/gogo/constants"
	codecoogsauth "github.com/codecoogs/gogo/wrappers/auth"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
)

// SyncEvent is deliberately narrower than events.Event: it carries only the
// columns Google Calendar owns. is_public, point_category and flyer_url are set
// by officers and must survive every poll, so they have no field here at all.
//
// None of these tags carry omitempty, and that is load-bearing. PostgREST builds
// an upsert's SET clause from the union of keys across the batch and fills any
// key a row omits with the column default, so a row that dropped "description"
// would blank out a description another row supplied. Every row must serialise
// the same seven keys, nulls included.
type SyncEvent struct {
	GoogleEventID string  `json:"google_event_id"`
	Title         string  `json:"title"`
	Description   *string `json:"description"`
	Location      *string `json:"location"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	Status        string  `json:"status"`
}

type Response struct {
	Success bool          `json:"success"`
	Synced  int           `json:"synced"`
	Error   *ErrorDetails `json:"error,omitempty"`
}

type ErrorDetails struct {
	Message string `json:"message"`
}

// The status column only accepts what the calendar can actually produce once
// Google's confirmed/tentative have been collapsed by the bot.
var validStatuses = map[string]struct{}{
	"scheduled": {},
	"cancelled": {},
}

// Timed events arrive as RFC3339; all-day events as a bare date.
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

func parseEventTime(value string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("must be an RFC3339 timestamp or a YYYY-MM-DD date")
}

// decodeSyncEvents rejects the whole batch if any row is bad. A partial sync
// would leave the website showing a calendar that matches neither Google nor
// the last good state, which is harder to notice than a failed poll.
func decodeSyncEvents(body io.Reader) ([]SyncEvent, error) {
	var events []SyncEvent
	if err := json.NewDecoder(body).Decode(&events); err != nil {
		return nil, err
	}

	for i, event := range events {
		if err := validateSyncEvent(event); err != nil {
			return nil, fmt.Errorf("event %d (%s): %w", i, event.GoogleEventID, err)
		}
	}

	return events, nil
}

func validateSyncEvent(event SyncEvent) error {
	if strings.TrimSpace(event.GoogleEventID) == "" {
		return errors.New("google_event_id is required")
	}
	if strings.TrimSpace(event.Title) == "" {
		return errors.New("title is required")
	}

	start, err := parseEventTime(event.StartTime)
	if err != nil {
		return errors.New("start_time " + err.Error())
	}

	end, err := parseEventTime(event.EndTime)
	if err != nil {
		return errors.New("end_time " + err.Error())
	}

	if end.Before(start) {
		return errors.New("end_time must not be before start_time")
	}

	if _, ok := validStatuses[event.Status]; !ok {
		return errors.New("status must be scheduled or cancelled")
	}

	return nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
	crw.SetCors(r.Host)
	crw.SetNoCache()

	if r.Method == "OPTIONS" {
		crw.SendJSONResponse(http.StatusOK, Response{Success: true})
		return
	}

	if r.Method != "POST" {
		crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Method not allowed for this resource",
			},
		})
		return
	}

	if token := r.Header.Get("Authorization"); !codecoogsauth.Authorize(token) {
		crw.SendJSONResponse(http.StatusUnauthorized, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Unauthorized access",
			},
		})
		return
	}

	client, err := codecoogssupabase.CreateClient()
	if err != nil {
		crw.SendJSONResponse(http.StatusInternalServerError, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Failed to create Supabase client: " + err.Error(),
			},
		})
		return
	}

	events, err := decodeSyncEvents(r.Body)
	if err != nil {
		crw.SendJSONResponse(http.StatusBadRequest, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// An empty calendar window is a valid poll result, not a reason to write.
	if len(events) == 0 {
		crw.SendJSONResponse(http.StatusOK, Response{
			Success: true,
			Synced:  0,
		})
		return
	}

	if _, _, err := client.From(constants.EVENT_TABLE).Upsert(events, "google_event_id", "minimal", "exact").Execute(); err != nil {
		status, message := codecoogssupabase.ErrorResponse(err, "Failed to sync events")
		crw.SendJSONResponse(status, Response{
			Success: false,
			Error: &ErrorDetails{
				Message: message,
			},
		})
		return
	}

	crw.SendJSONResponse(http.StatusOK, Response{
		Success: true,
		Synced:  len(events),
	})
}
