package main

import (
	"crypto/x509"
	"flag"
	"fmt"
	stdlog "log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/micromdm/nanomdm/certverify"
	"github.com/micromdm/nanomdm/cli"
	"github.com/micromdm/nanomdm/cryptoutil"
	mdmhttp "github.com/micromdm/nanomdm/http"
	httpapi "github.com/micromdm/nanomdm/http/api"
	"github.com/micromdm/nanomdm/http/authproxy"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	"github.com/micromdm/nanomdm/push/nanopush"
	pushsvc "github.com/micromdm/nanomdm/push/service"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/dump"
	"github.com/micromdm/nanomdm/service/microwebhook"
	"github.com/micromdm/nanomdm/service/multi"
	"github.com/micromdm/nanomdm/service/nanomdm"

	nlhttp "github.com/micromdm/nanolib/http"
	"github.com/micromdm/nanolib/http/trace"
	"github.com/micromdm/nanolib/log/stdlogfmt"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	endpointMDM     = "/mdm"
	endpointCheckin = "/checkin"

	endpointAuthProxy = "/authproxy/"

	endpointAPIMigration = "/migration"
	endpointAPIVersion   = "/version"
)

const (
	EnrollmentIDHeader = "X-Enrollment-ID"
	TraceIDHeader      = "X-Trace-ID"
)

func main() {
	cliStorage := cli.NewStorage()
	flag.Var(&cliStorage.Storage, "storage", "name of storage backend")
	flag.Var(&cliStorage.DSN, "storage-dsn", "data source name (e.g. connection string or path)")
	flag.Var(&cliStorage.DSN, "dsn", "data source name; deprecated: use -storage-dsn")
	flag.Var(&cliStorage.Options, "storage-options", "storage backend options")
	var (
		flListen     = flag.String("listen", ":9000", "HTTP listen address")
		flAPIKey     = flag.String("api", "", "API key for API endpoints")
		flVersion    = flag.Bool("version", false, "print version")
		flRootsPath  = flag.String("ca", "", "path to PEM CA cert(s)")
		flIntsPath   = flag.String("intermediate", "", "path to PEM intermediate cert(s)")
		flWebhook    = flag.String("webhook-url", "", "URL to send requests to")
		flCertHeader = flag.String("cert-header", "", "HTTP header containing TLS client certificate")
		flDebug      = flag.Bool("debug", false, "log debug messages")
		flDump       = flag.Bool("dump", false, "dump MDM requests and responses to stdout")
		flDisableMDM = flag.Bool("disable-mdm", false, "disable MDM HTTP endpoint")
		flCheckin    = flag.Bool("checkin", false, "enable separate HTTP endpoint for MDM check-ins")
		flMigration  = flag.Bool("migration", false, "enable HTTP endpoint for enrollment migrations")
		flRetro      = flag.Bool("retro", false, "Allow retroactive certificate-authorization association")
		flDMURLPfx   = flag.String("dm", "", "URL to send Declarative Management requests to")
		flAuthProxy  = flag.String("auth-proxy-url", "", "Reverse proxy URL target for MDM-authenticated HTTP requests")
		flUAZLChal   = flag.Bool("ua-zl-dc", false, "reply with zero-length DigestChallenge for UserAuthenticate")
		flVerify     = flag.String("verify", "pool", "device identity verification type")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if *flDisableMDM && *flAPIKey == "" {
		stdlog.Fatal("nothing for server to do")
	}

	logger := stdlogfmt.New(stdlogfmt.WithDebugFlag(*flDebug))

	if *flRootsPath == "" {
		stdlog.Fatal("must supply CA cert path flag")
	}
	caPEM, err := os.ReadFile(*flRootsPath)
	if err != nil {
		stdlog.Fatal(fmt.Errorf("reading root CA: %w", err))
	}

	var verifier certverify.CertVerifier
	switch *flVerify {
	case "pool":
		var intsPEM []byte
		if *flIntsPath != "" {
			intsPEM, err = os.ReadFile(*flIntsPath)
			if err != nil {
				stdlog.Fatal(fmt.Errorf("reading intermediate CA: %w", err))
			}
		}
		verifier, err = certverify.NewPoolVerifier(caPEM, intsPEM, x509.ExtKeyUsageClientAuth)
		if err != nil {
			stdlog.Fatal(err)
		}
	case "signature-only":
		if *flIntsPath != "" {
			stdlog.Fatal("intermediate cannot be used with signature-only verification")
		}
		verifier, err = certverify.NewSignatureVerifier(caPEM)
		if err != nil {
			stdlog.Fatal(err)
		}
		logger.Info(
			"msg", "reduced security: signature-only verifier",
			// double up and use a err in case that key is used for reporting
			"err", "reduced security: signature-only verifier",
		)
	default:
		stdlog.Fatal(fmt.Errorf("invalid verify flag: %s", *flVerify))
	}

	mdmStorage, err := cliStorage.Parse(logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	tokenMux := nanomdm.NewTokenMux()

	// create 'core' MDM service
	nanoOpts := []nanomdm.Option{
		nanomdm.WithUserAuthenticate(nanomdm.NewUAService(mdmStorage, *flUAZLChal)),
		nanomdm.WithGetToken(tokenMux),
		nanomdm.WithLogger(logger.With("service", "nanomdm")),
	}
	if *flDMURLPfx != "" {
		var warningText string
		if !strings.HasSuffix(*flDMURLPfx, "/") {
			warningText = ": warning: URL has no trailing slash"
		}
		logger.Debug("msg", "declarative management setup"+warningText, "url", *flDMURLPfx)
		dm, err := nanomdm.NewDeclarativeManagementHTTPCaller(*flDMURLPfx, http.DefaultClient)
		if err != nil {
			stdlog.Fatal(err)
		}
		nanoOpts = append(nanoOpts, nanomdm.WithDeclarativeManagement(dm))
	}
	nano := nanomdm.New(mdmStorage, nanoOpts...)

	mux := http.NewServeMux()
	mdmAuthMux := mdmhttp.NewMWMux(mux)

	if *flCertHeader != "" {
		// extract certificate from HTTP header (mTLS)
		mdmAuthMux.Use(func(h http.Handler) http.Handler {
			return httpmdm.CertExtractPEMHeaderMiddleware(h, *flCertHeader, logger.With("handler", "cert-extract"))
		})
	} else {
		opts := []httpmdm.SigLogOption{httpmdm.SigLogWithLogger(logger.With("handler", "cert-extract"))}

		if *flDebug {
			opts = append(opts, httpmdm.SigLogWithLogErrors(true))
		}

		// extract certificate from Mdm-Signature header
		mdmAuthMux.Use(func(h http.Handler) http.Handler {
			return httpmdm.CertExtractMdmSignatureMiddleware(h, httpmdm.MdmSignatureVerifierFunc(cryptoutil.VerifyMdmSignature), opts...)
		})
	}

	// finally, verify the identity certificate
	mdmAuthMux.Use(func(h http.Handler) http.Handler {
		return httpmdm.CertVerifyMiddleware(h, verifier, logger.With("handler", "cert-verify"))
	})

	if !*flDisableMDM {
		var mdmService service.CheckinAndCommandService = nano
		if *flWebhook != "" {
			webhookService := microwebhook.New(*flWebhook, mdmStorage)
			mdmService = multi.New(logger.With("service", "multi"), mdmService, webhookService)
		}
		certAuthOpts := []certauth.Option{certauth.WithLogger(logger.With("service", "certauth"))}
		if *flRetro {
			certAuthOpts = append(certAuthOpts, certauth.WithAllowRetroactive())
		}
		mdmService = certauth.New(mdmService, mdmStorage, certAuthOpts...)
		if *flDump {
			mdmService = dump.New(mdmService, os.Stdout)
		}

		// register 'core' MDM HTTP handlers
		if *flCheckin {
			// if we specified a separate check-in handler, set it up
			mdmAuthMux.Handle(endpointCheckin, httpmdm.CheckinHandler(mdmService, logger.With("handler", "checkin")))

			// if we use the check-in handler then only handle commands
			mdmAuthMux.Handle(endpointMDM, httpmdm.CommandAndReportResultsHandler(mdmService, logger.With("handler", "command")))
		} else {
			// if we don't use a check-in handler then do both
			mdmAuthMux.Handle(endpointMDM, httpmdm.CheckinAndCommandHandler(mdmService, logger.With("handler", "checkin-command")))
		}

		if *flAuthProxy != "" {
			authProxy, err := authproxy.New(*flAuthProxy,
				authproxy.WithLogger(logger.With("handler", "authproxy")),
				authproxy.WithHeaderFunc(EnrollmentIDHeader, httpmdm.GetEnrollmentID),
				authproxy.WithHeaderFunc(TraceIDHeader, trace.GetTraceID),
			)
			if err != nil {
				stdlog.Fatal(err)
			}

			apMux := mdmhttp.NewMWMux(mdmAuthMux)

			// wrap with enrollment ID lookup middleware
			apMux.Use(func(h http.Handler) http.Handler {
				return httpmdm.CertWithEnrollmentIDMiddleware(
					h,
					certauth.HashCert,
					mdmStorage,
					true,
					logger.With("handler", "with-enrollment-id"))
			})

			apMux.Handle(endpointAuthProxy, http.StripPrefix(endpointAuthProxy, authProxy))

			logger.Debug("msg", "authproxy setup", "url", *flAuthProxy)
		}
	}

	if *flAPIKey != "" {
		const apiUsername = "nanomdm"

		apiAuthMux := mdmhttp.NewMWMux(mux)

		apiAuthMux.Use(func(h http.Handler) http.Handler {
			return nlhttp.NewSimpleBasicAuthHandler(h, apiUsername, *flAPIKey, "nanomdm")
		})

		// create our push provider and push service
		pushProviderFactory := nanopush.NewFactory()
		pushService := pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, logger.With("service", "push"))

		// register API handlers
		httpapi.HandleAPIv1("/v1", apiAuthMux, logger, mdmStorage, pushService)

		if *flMigration {
			// setup a "migration" handler that takes Check-In messages
			// without bothering with certificate auth or other
			// middleware.
			//
			// if the source MDM can put together enough of an
			// authenticate and tokenupdate message to effectively
			// generate "enrollments" then this effively allows us to
			// migrate MDM enrollments between servers.
			apiAuthMux.Handle(
				endpointAPIMigration,
				httpmdm.CheckinHandler(nano, logger.With("handler", "migration")),
			)
		}
	}

	mux.HandleFunc(endpointAPIVersion, nlhttp.NewJSONVersionHandler(version))

	rand.Seed(time.Now().UnixNano())

	logger.Info("msg", "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, trace.NewTraceLoggingHandler(mux, logger.With("handler", "log"), newTraceID))
	logs := []interface{}{"msg", "server shutdown"}
	if err != nil {
		logs = append(logs, "err", err)
	}
	logger.Info(logs...)
}

// newTraceID generates a new HTTP trace ID for context logging.
// Currently this just makes a random string. This would be better
// served by e.g. https://github.com/oklog/ulid or something like
// https://opentelemetry.io/ someday.
func newTraceID(_ *http.Request) string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
