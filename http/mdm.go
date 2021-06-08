package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

// CheckinHandlerFunc decodes an MDM check-in request and adapts it to service.
func CheckinHandlerFunc(svc service.Checkin, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		mdmReq := &mdm.Request{
			Context:     r.Context(),
			Certificate: GetCert(r.Context()),
		}
		respBytes, err := service.CheckinRequest(svc, mdmReq, bodyBytes)
		if err != nil {
			logger.Info("msg", "check-in request", "err", err)
			var decodeError *service.DecodeError
			if errors.Is(err, mdm.ErrUnrecognizedMessageType) || errors.As(err, &decodeError) {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Write(respBytes)
	}
}

// CommandAndReportResultsHandlerFunc decodes an MDM command request and adapts it to service.
func CommandAndReportResultsHandlerFunc(svc service.CommandAndReportResults, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		mdmReq := &mdm.Request{
			Context:     r.Context(),
			Certificate: GetCert(r.Context()),
		}
		respBytes, err := service.CommandAndReportResultsRequest(svc, mdmReq, bodyBytes)
		if err != nil {
			logger.Info("msg", "command report results", "err", err)
			var decodeError *service.DecodeError
			if errors.As(err, &decodeError) {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.Write(respBytes)
	}
}

// CheckinAndCommandHandlerFunc handles both check-in and command requests.
func CheckinAndCommandHandlerFunc(service service.CheckinAndCommandService, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "application/x-apple-aspen-mdm-checkin") {
			CheckinHandlerFunc(service, logger).ServeHTTP(w, r)
			return
		}
		// assume a non-check-in is a command request
		CommandAndReportResultsHandlerFunc(service, logger).ServeHTTP(w, r)
	}
}
