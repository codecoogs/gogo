package events

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/codecoogs/gogo/constants"
	codecoogsauth "github.com/codecoogs/gogo/wrappers/auth"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase/postgrest-go"
)

// Google owns title, times, location, description and status; officers own
// is_public, point_category and flyer_url. Both sets live on this struct because
// the id routes read and write whole rows, but the sync route deliberately uses
// a narrower struct so a calendar poll can never touch the officer columns.
type Event struct {
	ID            *int64  `json:"id,omitempty"`
	GoogleEventID *string `json:"google_event_id,omitempty"`
	Title         string  `json:"title"`
	Description   *string `json:"description,omitempty"`
	Location      *string `json:"location,omitempty"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	PointCategory *string `json:"point_category,omitempty"`
	FlyerURL      *string `json:"flyer_url,omitempty"`
	IsPublic      *bool   `json:"is_public,omitempty"`
	Status        string  `json:"status,omitempty"`
	CreatedAt     *string `json:"created_at,omitempty"`
	CreatedBy     *string `json:"created_by,omitempty"`
	UpdatedAt     *string `json:"updated_at,omitempty"`
	UpdatedBy     *string `json:"updated_by,omitempty"`
}

// The website refetches the list on every page load, so a short shared cache
// keeps refreshes off the API without making edits take long to appear. Reads
// by id are not cached, so admin tools always see their own writes.
const listCacheSeconds = 300

type listFilters struct {
	IsPublic    *bool
	Status      string
	StartAfter  string
	StartBefore string
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Event       `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`
}

type ErrorDetails struct {
	Message string `json:"message"`
}

// Supabase returns timestamptz as RFC3339, but all-day events are stored as a
// bare date. Both have to round-trip, so accept either.
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

func parseListFilters(q url.Values) (listFilters, error) {
	var f listFilters

	if q.Has("is_public") {
		v, err := strconv.ParseBool(q.Get("is_public"))
		if err != nil {
			return f, errors.New("is_public must be true or false")
		}
		f.IsPublic = &v
	}

	f.Status = q.Get("status")

	if v := q.Get("start_after"); v != "" {
		if _, err := parseEventTime(v); err != nil {
			return f, errors.New("start_after " + err.Error())
		}
		f.StartAfter = v
	}

	if v := q.Get("start_before"); v != "" {
		if _, err := parseEventTime(v); err != nil {
			return f, errors.New("start_before " + err.Error())
		}
		f.StartBefore = v
	}

	return f, nil
}

func decodeEvent(body io.Reader) (Event, error) {
	var event Event
	if err := json.NewDecoder(body).Decode(&event); err != nil {
		return event, err
	}

	if strings.TrimSpace(event.Title) == "" {
		return event, errors.New("title is required")
	}

	start, err := parseEventTime(event.StartTime)
	if err != nil {
		return event, errors.New("start_time " + err.Error())
	}

	end, err := parseEventTime(event.EndTime)
	if err != nil {
		return event, errors.New("end_time " + err.Error())
	}

	if end.Before(start) {
		return event, errors.New("end_time must not be before start_time")
	}

	return event, nil
}

func applyListFilters(query *postgrest.FilterBuilder, f listFilters) *postgrest.FilterBuilder {
	if f.IsPublic != nil {
		query = query.Eq("is_public", strconv.FormatBool(*f.IsPublic))
	}
	if f.Status != "" {
		query = query.Eq("status", f.Status)
	}
	if f.StartAfter != "" {
		query = query.Gte("start_time", f.StartAfter)
	}
	if f.StartBefore != "" {
		query = query.Lte("start_time", f.StartBefore)
	}

	return query
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
	crw.SetCors(r.Host)

	if r.Method == "OPTIONS" {
		crw.SendJSONResponse(http.StatusOK, Response{Success: true})
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

	// Reads stay open for the website; every write needs the shared secret.
	if r.Method != "GET" {
		if token := r.Header.Get("Authorization"); !codecoogsauth.Authorize(token) {
			crw.SendJSONResponse(http.StatusUnauthorized, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Unauthorized access",
				},
			})
			return
		}
	}

	id := r.URL.Query().Get("id")

	if id == "" {
		switch r.Method {
		case "GET":
			filters, err := parseListFilters(r.URL.Query())
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid query parameters: " + err.Error(),
					},
				})
				return
			}

			var startTimeOrder = &postgrest.OrderOpts{
				Ascending: true,
			}

			var events []Event
			query := applyListFilters(client.From(constants.EVENT_TABLE).Select("*", "exact", false), filters)
			if _, err := query.Order("start_time", startTimeOrder).ExecuteTo(&events); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to get events")
				crw.SendJSONResponse(status, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: message,
					},
				})
				return
			}
			crw.SetCache(listCacheSeconds)
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    events,
			})
		case "POST":
			event, err := decodeEvent(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.EVENT_TABLE).Insert(event, false, "", "", "exact").Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to create event")
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
			})
		default:
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed for this resource",
				},
			})
		}
	} else {
		switch r.Method {
		case "GET":
			var event []Event
			if _, err := client.From(constants.EVENT_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&event); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to get event")
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
				Data:    event,
			})
		case "PUT":
			updatedEvent, err := decodeEvent(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.EVENT_TABLE).Update(updatedEvent, "", "exact").Eq("id", id).Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to update event")
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
			})
		case "DELETE":
			if _, _, err := client.From(constants.EVENT_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to delete event")
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
			})
		default:
			crw.SendJSONResponse(http.StatusMethodNotAllowed, Response{
				Success: false,
				Error: &ErrorDetails{
					Message: "Method not allowed for this resource",
				},
			})
		}
	}
}
