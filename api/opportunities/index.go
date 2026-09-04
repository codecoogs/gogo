package opportunities

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
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase/postgrest-go"
)

type Opportunity struct {
	ID              *string `json:"id,omitempty"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	LinkURL         string  `json:"link_url,omitempty"`
	LinkedFormID    *string `json:"linked_form_id,omitempty"`
	Category        string  `json:"category,omitempty"`
	IconURL         string  `json:"icon_url,omitempty"`
	Term            string  `json:"term,omitempty"`
	OpensOn         *string `json:"opens_on,omitempty"`
	ClosesOn        *string `json:"closes_on,omitempty"`
	ExpiresAt       *string `json:"expires_at,omitempty"`
	DisplayOrder    *int    `json:"display_order,omitempty"`
	WebsiteViewable *bool   `json:"website_viewable,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	CreatedBy       *string `json:"created_by,omitempty"`
	UpdatedAt       *string `json:"updated_at,omitempty"`
	UpdatedBy       *string `json:"updated_by,omitempty"`
}

type listFilters struct {
	WebsiteViewable *bool
	IsActive        *bool
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Opportunity `json:"data,omitempty"`
	Error   *ErrorDetails `json:"error,omitempty"`
}

type ErrorDetails struct {
	Message string `json:"message"`
}

func parseListFilters(q url.Values) (listFilters, error) {
	var f listFilters

	if q.Has("website_viewable") {
		v, err := strconv.ParseBool(q.Get("website_viewable"))
		if err != nil {
			return f, errors.New("website_viewable must be true or false")
		}
		f.WebsiteViewable = &v
	}

	if q.Has("is_active") {
		v, err := strconv.ParseBool(q.Get("is_active"))
		if err != nil {
			return f, errors.New("is_active must be true or false")
		}
		f.IsActive = &v
	}

	return f, nil
}

func decodeOpportunity(body io.Reader) (Opportunity, error) {
	var opportunity Opportunity
	if err := json.NewDecoder(body).Decode(&opportunity); err != nil {
		return opportunity, err
	}

	if strings.TrimSpace(opportunity.Title) == "" {
		return opportunity, errors.New("title is required")
	}

	hasLink := strings.TrimSpace(opportunity.LinkURL) != ""
	hasForm := opportunity.LinkedFormID != nil && strings.TrimSpace(*opportunity.LinkedFormID) != ""
	if hasLink == hasForm {
		return opportunity, errors.New("exactly one of link_url or linked_form_id is required")
	}

	return opportunity, nil
}

func applyListFilters(query *postgrest.FilterBuilder, f listFilters) *postgrest.FilterBuilder {
	if f.WebsiteViewable != nil {
		query = query.Eq("website_viewable", strconv.FormatBool(*f.WebsiteViewable))
	}
	if f.IsActive != nil {
		query = query.Eq("is_active", strconv.FormatBool(*f.IsActive))
	}

	return query
}

func Handler(w http.ResponseWriter, r *http.Request) {
	crw := &codecoogshttp.ResponseWriter{W: w}
	crw.SetCors(r.Host)

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

			var MyOrderOpts = &postgrest.OrderOpts{
				Ascending: true,
			}

			var opportunities []Opportunity
			query := applyListFilters(client.From(constants.OPPORTUNITY_TABLE).Select("*", "exact", false), filters)
			if _, err := query.Order("display_order", MyOrderOpts).ExecuteTo(&opportunities); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get opportunities: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    opportunities,
			})
		case "POST":
			opportunity, err := decodeOpportunity(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.OPPORTUNITY_TABLE).Insert(opportunity, false, "", "", "exact").Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to create opportunity: " + err.Error(),
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
			var opportunity []Opportunity
			if _, err := client.From(constants.OPPORTUNITY_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&opportunity); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to get opportunity: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
				Data:    opportunity,
			})
		case "PUT":
			updatedOpportunity, err := decodeOpportunity(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			updatedAt := time.Now().UTC().Format(time.RFC3339)
			updatedOpportunity.UpdatedAt = &updatedAt

			if _, _, err := client.From(constants.OPPORTUNITY_TABLE).Update(updatedOpportunity, "", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to update opportunity: " + err.Error(),
					},
				})
				return
			}
			crw.SendJSONResponse(http.StatusOK, Response{
				Success: true,
			})
		case "DELETE":
			if _, _, err := client.From(constants.OPPORTUNITY_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				crw.SendJSONResponse(http.StatusInternalServerError, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Failed to delete opportunity: " + err.Error(),
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
