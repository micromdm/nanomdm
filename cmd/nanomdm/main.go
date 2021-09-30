package main

import (
	"crypto/subtle"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net"
	"net/http"
	"os"

	"github.com/micromdm/nanomdm/certverify"
	"github.com/micromdm/nanomdm/cmd/cli"
	mdmhttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/stdlogfmt"
	"github.com/micromdm/nanomdm/push/buford"
	pushsvc "github.com/micromdm/nanomdm/push/service"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/dump"
	"github.com/micromdm/nanomdm/service/microwebhook"
	"github.com/micromdm/nanomdm/service/multi"
	"github.com/micromdm/nanomdm/service/nanomdm"
)

// overridden by -ldflags -X
var version = "unknown"

const (
	endpointMDM     = "/mdm"
	endpointCheckin = "/checkin"

	endpointAPIPushCert  = "/v1/pushcert"
	endpointAPIPush      = "/v1/push/"
	endpointAPIEnqueue   = "/v1/enqueue/"
	endpointAPIMigration = "/migration"
	endpointAPIVersion   = "/version"
)

func main() {
	cliStorage := cli.NewStorage()
	flag.Var(&cliStorage.Storage, "storage", "name of storage system")
	flag.Var(&cliStorage.DSN, "dsn", "data source name (e.g. connection string or path)")
	var (
		flListen     = flag.String("listen", ":9000", "HTTP listen address")
		flAPIKey     = flag.String("api", "", "API key for API endpoints")
		flVersion    = flag.Bool("version", false, "print version")
		flRootsPath  = flag.String("ca", "", "path to CA cert for verification")
		flWebhook    = flag.String("webhook-url", "", "URL to send requests to")
		flCertHeader = flag.String("cert-header", "", "HTTP header containing URL-escaped TLS client certificate")
		flDebug      = flag.Bool("debug", false, "log debug messages")
		flDump       = flag.Bool("dump", false, "dump MDM requests and responses to stdout")
		flDisableMDM = flag.Bool("disable-mdm", false, "disable MDM HTTP endpoint")
		flCheckin    = flag.Bool("checkin", false, "enable separate HTTP endpoint for MDM check-ins")
		flMigration  = flag.Bool("migration", false, "HTTP endpoint for enrollment migrations")
		flRetro      = flag.Bool("retro", false, "Allow retroactive certificate-authorization association")
		flDMURLPfx   = flag.String("dm", "", "URL to send Declarative Management requests to")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if *flDisableMDM && *flAPIKey == "" {
		stdlog.Fatal("nothing for server to do")
	}

	logger := stdlogfmt.New(stdlog.Default(), *flDebug)

	if *flRootsPath == "" {
		stdlog.Fatal("must supply CA cert path flag")
	}
	caPEM, err := ioutil.ReadFile(*flRootsPath)
	if err != nil {
		stdlog.Fatal(err)
	}
	verifier, err := certverify.NewPoolVerifier(caPEM, x509.ExtKeyUsageClientAuth)
	if err != nil {
		stdlog.Fatal(err)
	}

	mdmStorage, err := cliStorage.Parse(logger)
	if err != nil {
		stdlog.Fatal(err)
	}

	// create 'core' MDM service
	nanoOpts := []nanomdm.Option{nanomdm.WithLogger(logger.With("service", "nanomdm"))}
	if *flDMURLPfx != "" {
		logger.Debug("msg", "declarative management setup", "url", *flDMURLPfx)
		dm := nanomdm.NewDeclarativeManagementHTTPCaller(*flDMURLPfx)
		nanoOpts = append(nanoOpts, nanomdm.WithDeclarativeManagement(dm))
	}
	nano := nanomdm.New(mdmStorage, nanoOpts...)

	mux := http.NewServeMux()

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

		// register 'core' MDM HTTP handler
		var mdmHandler http.Handler
		if *flCheckin {
			// if we use the check-in handler then only handle commands
			mdmHandler = mdmhttp.CommandAndReportResultsHandlerFunc(mdmService, logger.With("handler", "command"))
		} else {
			// if we don't use a check-in handler then do both
			mdmHandler = mdmhttp.CheckinAndCommandHandlerFunc(mdmService, logger.With("handler", "checkin-command"))
		}
		mdmHandler = mdmhttp.CertVerifyMiddleware(mdmHandler, verifier, logger.With("handler", "cert-verify"))
		if *flCertHeader != "" {
			mdmHandler = mdmhttp.CertExtractPEMHeaderMiddleware(mdmHandler, *flCertHeader, logger.With("handler", "cert-extract"))
		} else {
			mdmHandler = mdmhttp.CertExtractMdmSignatureMiddleware(mdmHandler, logger.With("handler", "cert-extract"))
		}
		mux.Handle(endpointMDM, mdmHandler)

		if *flCheckin {
			// if we specified a separate check-in handler, set it up
			var checkinHandler http.Handler
			checkinHandler = mdmhttp.CheckinHandlerFunc(mdmService, logger.With("handler", "checkin"))
			checkinHandler = mdmhttp.CertVerifyMiddleware(checkinHandler, verifier, logger.With("handler", "cert-verify"))
			if *flCertHeader != "" {
				checkinHandler = mdmhttp.CertExtractPEMHeaderMiddleware(checkinHandler, *flCertHeader, logger.With("handler", "cert-extract"))
			} else {
				checkinHandler = mdmhttp.CertExtractMdmSignatureMiddleware(checkinHandler, logger.With("handler", "cert-extract"))
			}
			mux.Handle(endpointCheckin, checkinHandler)
		}
	}

	if *flAPIKey != "" {
		const apiUsername = "nanomdm"

		// create our push provider and push service
		pushProviderFactory := buford.NewPushProviderFactory()
		pushService := pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, logger.With("service", "push"))

		// register API handler for push cert storage/upload.
		var pushCertHandler http.Handler
		pushCertHandler = mdmhttp.StorePushCertHandlerFunc(mdmStorage, logger.With("handler", "store-cert"))
		pushCertHandler = basicAuth(pushCertHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIPushCert, pushCertHandler)

		// register API handler for push notifications.
		// we strip the prefix to use the path as an id.
		var pushHandler http.Handler
		pushHandler = mdmhttp.PushHandlerFunc(pushService, logger.With("handler", "push"))
		pushHandler = http.StripPrefix(endpointAPIPush, pushHandler)
		pushHandler = basicAuth(pushHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIPush, pushHandler)

		// register API handler for new command queueing.
		// we strip the prefix to use the path as an id.
		var enqueueHandler http.Handler
		enqueueHandler = mdmhttp.RawCommandEnqueueHandler(mdmStorage, pushService, logger.With("handler", "enqueue"))
		enqueueHandler = http.StripPrefix(endpointAPIEnqueue, enqueueHandler)
		enqueueHandler = basicAuth(enqueueHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(endpointAPIEnqueue, enqueueHandler)

		if *flMigration {
			// setup a "migration" handler that takes Check-In messages
			// without bothering with certificate auth or other
			// middleware.
			//
			// if the source MDM can put together enough of an
			// authenticate and tokenupdate message to effectively
			// generate "enrollments" then this effively allows us to
			// migrate MDM enrollments between servers.
			var migHandler http.Handler
			migHandler = mdmhttp.CheckinHandlerFunc(nano, logger.With("handler", "migration"))
			migHandler = basicAuth(migHandler, apiUsername, *flAPIKey, "nanomdm")
			mux.Handle(endpointAPIMigration, migHandler)
		}
	}

	mux.HandleFunc(endpointAPIVersion, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"` + version + `"}`))
	})

	logger.Info("msg", "starting server", "listen", *flListen)
	err = http.ListenAndServe(*flListen, simpleLog(mux, logger.With("handler", "log")))
	logs := []interface{}{"msg", "server shutdown"}
	if err != nil {
		logs = append(logs, "err", err)
	}
	logger.Info(logs...)
}

func basicAuth(next http.Handler, username, password, realm string) http.HandlerFunc {
	uBytes := []byte(username)
	pBytes := []byte(password)
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), uBytes) != 1 || subtle.ConstantTimeCompare([]byte(p), pBytes) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func simpleLog(next http.Handler, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		logs := []interface{}{
			"addr", host,
			"method", r.Method,
			"path", r.URL.Path,
			"agent", r.UserAgent(),
		}
		if fwdedFor := r.Header.Get("X-Forwarded-For"); fwdedFor != "" {
			logs = append(logs, "real_ip", fwdedFor)
		}
		logger.Info(logs...)
		next.ServeHTTP(w, r)
	}
}
