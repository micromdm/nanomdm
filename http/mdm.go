package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/jessepeterson/nanomdm/log"
	"github.com/jessepeterson/nanomdm/mdm"
	"github.com/jessepeterson/nanomdm/service"
)

// CheckinHandlerFunc decodes an MDM check-in request and adapts it to service.
func CheckinHandlerFunc(service service.Checkin, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		m, err := mdm.DecodeCheckin(bodyBytes)
		if err != nil {
			logger.Info("msg", "decoding check-in", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		mdmReq := &mdm.Request{
			Context:     r.Context(),
			Certificate: GetCert(r.Context()),
		}
		switch message := m.(type) {
		case *mdm.Authenticate:
			err = service.Authenticate(mdmReq, message)
			if err != nil {
				err = fmt.Errorf("authenticate: %w", err)
			}
		case *mdm.TokenUpdate:
			err = service.TokenUpdate(mdmReq, message)
			if err != nil {
				err = fmt.Errorf("tokenupdate: %w", err)
			}
		case *mdm.CheckOut:
			err = service.CheckOut(mdmReq, message)
			if err != nil {
				err = fmt.Errorf("checkout: %w", err)
			}
		default:
			logger.Info("err", mdm.ErrUnrecognizedMessageType)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		if err != nil {
			logger.Info("msg", "service error in check-in", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

// CommandAndReportResultsHandlerFunc decodes an MDM command request and adapts it to service.
func CommandAndReportResultsHandlerFunc(service service.CommandAndReportResults, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		report, err := mdm.DecodeCommandResults(bodyBytes)
		if err != nil {
			logger.Info("msg", "decoding command report", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		mdmReq := &mdm.Request{
			Context:     r.Context(),
			Certificate: GetCert(r.Context()),
		}
		cmd, err := service.CommandAndReportResults(mdmReq, report)
		if err != nil {
			logger.Info("msg", "command report results", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		if cmd != nil {
			w.Write(cmd.Raw)
		}
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
