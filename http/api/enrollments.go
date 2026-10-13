package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
	"github.com/micromdm/nanomdm/storage"
)

const defaultEnrollmentsLimit = 100

// EnrollmentsQueryHandler handles POST requests to query enrollments.
func EnrollmentsQueryHandler(store storage.EnrollmentsStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)

		// Parse query parameters for pagination
		pagination := parsePaginationFromQuery(r)

		// Parse request body for filter and options
		var reqBody EnrollmentsQueryJson
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				logAndWriteJSONError(logger, w, "decoding request body", err, http.StatusBadRequest)
				return
			}
		}

		// Build storage query
		query := &storage.EnrollmentsQuery{
			Pagination: pagination,
		}

		// Map filter from JSON to storage type
		if reqBody.Filter != nil {
			query.Filter = &storage.EnrollmentsQueryFilter{
				IDs:            reqBody.Filter.IDs,
				Serials:        reqBody.Filter.Serials,
				UserShortNames: reqBody.Filter.UserShortNames,
				Types:          reqBody.Filter.Types,
				Enabled:        reqBody.Filter.Enabled,
			}
		}

		// Map options from JSON to storage type
		if reqBody.Options != nil {
			query.Options = &storage.EnrollmentQueryOptions{
				IncludeDeviceCert:  reqBody.Options.IncludeDeviceCert,
				IncludeUnlockToken: reqBody.Options.IncludeUnlockToken,
			}
		}

		// Validate pagination
		if pagination != nil {
			if err := pagination.ValidErr(); err != nil {
				logAndWriteJSONError(logger, w, "invalid pagination", err, http.StatusBadRequest)
				return
			}
		}

		// Execute query
		result, err := store.QueryEnrollments(r.Context(), query)
		if err != nil {
			logAndWriteJSONError(logger, w, "querying enrollments", err, http.StatusInternalServerError)
			return
		}

		// Map result to JSON response
		response := mapEnrollmentsResult(result)

		writeJSON(w, response, http.StatusOK, logger)
	}
}

// parsePaginationFromQuery parses pagination parameters from query string.
func parsePaginationFromQuery(r *http.Request) *storage.Pagination {
	q := r.URL.Query()

	var pagination *storage.Pagination

	if limitStr := q.Get("limit"); limitStr != "" {
		if pagination == nil {
			pagination = &storage.Pagination{}
		}
		if limit, err := strconv.Atoi(limitStr); err == nil {
			pagination.Limit = &limit
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if pagination == nil {
			pagination = &storage.Pagination{}
		}
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			pagination.Offset = &offset
		}
	}

	if cursor := q.Get("cursor"); cursor != "" {
		if pagination == nil {
			pagination = &storage.Pagination{}
		}
		pagination.Cursor = &cursor
	}

	return pagination
}

// mapEnrollmentsResult maps storage result to JSON response.
func mapEnrollmentsResult(result *storage.EnrollmentsQueryResult) *EnrollmentsQueryResultJson {
	if result == nil {
		return &EnrollmentsQueryResultJson{
			Enrollments: []EnrollmentJson{},
		}
	}

	enrollments := make([]EnrollmentJson, 0, len(result.Enrollments))
	for _, e := range result.Enrollments {
		enrollment := EnrollmentJson{
			ID:               e.ID,
			Type:             e.Type.String(),
			Enabled:          e.Enabled,
			TokenUpdateTally: e.TokenUpdateTally,
			LastSeen:         e.LastSeen,
		}

		if e.Device != nil {
			enrollment.Device = &DeviceEnrollmentJson{
				SerialNumber: e.Device.SerialNumber,
				DeviceCert:   e.Device.DeviceCert,
				UnlockToken:  e.Device.UnlockToken,
			}
		}

		if e.User != nil {
			enrollment.User = &UserEnrollmentJson{
				UserShortName: e.User.UserShortName,
				UserLongName:  e.User.UserLongName,
			}
		}

		enrollments = append(enrollments, enrollment)
	}

	response := &EnrollmentsQueryResultJson{
		Enrollments: enrollments,
		NextCursor:  result.NextCursor,
	}

	return response
}

// EnrollmentsQueryJson is the JSON request body for enrollment queries.
type EnrollmentsQueryJson struct {
	Filter  *EnrollmentsQueryFilterJson  `json:"filter,omitempty"`
	Options *EnrollmentQueryOptionsJson `json:"options,omitempty"`
}

// EnrollmentsQueryFilterJson is the JSON filter for enrollment queries.
type EnrollmentsQueryFilterJson struct {
	IDs            []string `json:"ids,omitempty"`
	Serials        []string `json:"serials,omitempty"`
	UserShortNames []string `json:"user_short_names,omitempty"`
	Types          []string `json:"types,omitempty"`
	Enabled        *bool    `json:"enabled,omitempty"`
}

// EnrollmentQueryOptionsJson is the JSON options for enrollment queries.
type EnrollmentQueryOptionsJson struct {
	IncludeDeviceCert  bool `json:"include_device_cert,omitempty"`
	IncludeUnlockToken bool `json:"include_unlock_token,omitempty"`
}

// EnrollmentsQueryResultJson is the JSON response for enrollment queries.
type EnrollmentsQueryResultJson struct {
	Enrollments []EnrollmentJson `json:"enrollments"`
	NextCursor  *string          `json:"next_cursor,omitempty"`
}

// EnrollmentJson is the JSON representation of an enrollment.
type EnrollmentJson struct {
	ID               string                `json:"id"`
	Type             string                `json:"type"`
	Device           *DeviceEnrollmentJson `json:"device,omitempty"`
	User             *UserEnrollmentJson   `json:"user,omitempty"`
	Enabled          bool                  `json:"enabled"`
	TokenUpdateTally int                   `json:"token_update_tally"`
	LastSeen         time.Time             `json:"last_seen"`
}

// DeviceEnrollmentJson is the JSON representation of device enrollment data.
type DeviceEnrollmentJson struct {
	SerialNumber string `json:"serial_number,omitempty"`
	DeviceCert   string `json:"device_cert,omitempty"`
	UnlockToken  []byte `json:"unlock_token,omitempty"`
}

// UserEnrollmentJson is the JSON representation of user enrollment data.
type UserEnrollmentJson struct {
	UserShortName string `json:"user_short_name,omitempty"`
	UserLongName  string `json:"user_long_name,omitempty"`
}
