package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/micromdm/nanomdm/storage"
	"github.com/micromdm/nanolib/log"
)

// EnrollmentListHandler exposes persisted enrollment summaries when the
// configured storage backend supports the optional EnrollmentReader API.
func EnrollmentListHandler(store interface{}, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, ok := store.(storage.EnrollmentReader)
		if !ok {
			logAndWriteJSONError(logger, w, "list enrollments", errors.New("storage backend does not support enrollment listing"), http.StatusNotImplemented)
			return
		}
		limit := 100
		if raw := r.URL.Query().Get("limit"); raw != "" {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 1 || value > 500 {
				logAndWriteJSONError(logger, w, "parse limit", errors.New("limit must be between 1 and 500"), http.StatusBadRequest)
				return
			}
			limit = value
		}
		items, err := reader.ListEnrollments(r.Context(), strings.TrimSpace(r.URL.Query().Get("q")), limit)
		if err != nil {
			logAndWriteJSONError(logger, w, "list enrollments", err, http.StatusInternalServerError)
			return
		}
		writeJSON(w, items, http.StatusOK, logger)
	}
}

// EnrollmentDetailHandler returns one persisted enrollment summary.
func EnrollmentDetailHandler(store interface{}, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reader, ok := store.(storage.EnrollmentReader)
		if !ok {
			logAndWriteJSONError(logger, w, "get enrollment", errors.New("storage backend does not support enrollment details"), http.StatusNotImplemented)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/")
		if id == "" {
			logAndWriteJSONError(logger, w, "get enrollment", errors.New("missing enrollment id"), http.StatusBadRequest)
			return
		}
		item, err := reader.GetEnrollment(r.Context(), id)
		if err != nil {
			logAndWriteJSONError(logger, w, "get enrollment", err, http.StatusNotFound)
			return
		}
		writeJSON(w, item, http.StatusOK, logger)
	}
}
