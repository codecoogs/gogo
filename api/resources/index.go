package resources

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/codecoogs/gogo/constants"
	codecoogshttp "github.com/codecoogs/gogo/wrappers/http"
	codecoogssupabase "github.com/codecoogs/gogo/wrappers/supabase"
	"github.com/supabase/postgrest-go"
)

type Resource struct {
	ID              *string `json:"id,omitempty"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	Category        string  `json:"category"`
	LinkURL         string  `json:"link_url"`
	Extension       string  `json:"extension,omitempty"`
	ThumbnailURL    string  `json:"thumbnail_url,omitempty"`
	ResourceDate    *string `json:"resource_date,omitempty"`
	DisplayOrder    *int    `json:"display_order,omitempty"`
	WebsiteViewable *bool   `json:"website_viewable,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	CreatedAt       *string `json:"created_at,omitempty"`
	CreatedBy       *string `json:"created_by,omitempty"`
	UpdatedAt       *string `json:"updated_at,omitempty"`
	UpdatedBy       *string `json:"updated_by,omitempty"`
}

// The website refetches the list on every page load, so a short shared cache
// keeps refreshes off the API without making edits take long to appear. Reads
// by id are not cached, so admin tools always see their own writes.
const listCacheSeconds = 300

type listFilters struct {
	WebsiteViewable *bool
	IsActive        *bool
	Category        string
}

type Response struct {
	Success bool          `json:"success"`
	Data    []Resource    `json:"data,omitempty"`
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

	f.Category = q.Get("category")

	return f, nil
}

func decodeResource(body io.Reader) (Resource, error) {
	var resource Resource
	if err := json.NewDecoder(body).Decode(&resource); err != nil {
		return resource, err
	}

	if strings.TrimSpace(resource.Title) == "" {
		return resource, errors.New("title is required")
	}
	if strings.TrimSpace(resource.Category) == "" {
		return resource, errors.New("category is required")
	}
	if strings.TrimSpace(resource.LinkURL) == "" {
		return resource, errors.New("link_url is required")
	}

	return resource, nil
}

func applyListFilters(query *postgrest.FilterBuilder, f listFilters) *postgrest.FilterBuilder {
	if f.WebsiteViewable != nil {
		query = query.Eq("website_viewable", strconv.FormatBool(*f.WebsiteViewable))
	}
	if f.IsActive != nil {
		query = query.Eq("is_active", strconv.FormatBool(*f.IsActive))
	}
	if f.Category != "" {
		query = query.Eq("category", f.Category)
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

			var resources []Resource
			query := applyListFilters(client.From(constants.RESOURCE_TABLE).Select("*", "exact", false), filters)
			if _, err := query.Order("display_order", MyOrderOpts).ExecuteTo(&resources); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to get resources")
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
				Data:    resources,
			})
		case "POST":
			resource, err := decodeResource(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.RESOURCE_TABLE).Insert(resource, false, "", "", "exact").Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to create resource")
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
			var resource []Resource
			if _, err := client.From(constants.RESOURCE_TABLE).Select("*", "exact", false).Eq("id", id).ExecuteTo(&resource); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to get resource")
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
				Data:    resource,
			})
		case "PUT":
			updatedResource, err := decodeResource(r.Body)
			if err != nil {
				crw.SendJSONResponse(http.StatusBadRequest, Response{
					Success: false,
					Error: &ErrorDetails{
						Message: "Invalid request body: " + err.Error(),
					},
				})
				return
			}

			if _, _, err := client.From(constants.RESOURCE_TABLE).Update(updatedResource, "", "exact").Eq("id", id).Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to update resource")
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
			if _, _, err := client.From(constants.RESOURCE_TABLE).Delete("", "exact").Eq("id", id).Execute(); err != nil {
				status, message := codecoogssupabase.ErrorResponse(err, "Failed to delete resource")
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
