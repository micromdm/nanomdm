// Package authproxy is a simple reverse proxy for Apple MDM clients.
package authproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	mdmhttp "github.com/micromdm/nanomdm/http"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/ctxlog"
)

const (
	EnrollmentIDHeader = "X-Enrollment-ID"
	TraceIDHeader      = "X-Trace-ID"
)

// New creates a new NanoMDM enrollment authenticating reverse proxy.
// This reverse proxy is mostly the standard httputil proxy. It depends
// on middleware HTTP handlers to enforce authentication and set the
// context value for the enrollment ID.
func New(dest string, logger log.Logger) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(dest)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		ctxlog.Logger(r.Context(), logger).Info("err", err)
		// use the same error as the standrad reverse proxy
		w.WriteHeader(http.StatusBadGateway)
	}
	dir := proxy.Director
	proxy.Director = func(req *http.Request) {
		dir(req)
		req.Host = target.Host
		// save the effort of forwarding this huge header
		req.Header.Del("Mdm-Signature")
		if id := httpmdm.GetEnrollmentID(req.Context()); id != "" {
			req.Header.Set(EnrollmentIDHeader, id)
		}
		// TODO: this couples us to our specific idea of trace logging
		// Perhaps have an optional config for header specificaiton?
		if id := mdmhttp.GetTraceID(req.Context()); id != "" {
			req.Header.Set(TraceIDHeader, id)
		}
	}
	return proxy, nil
}
