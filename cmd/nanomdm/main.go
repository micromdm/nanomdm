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

	"github.com/jessepeterson/nanomdm/certverify"
	mdmhttp "github.com/jessepeterson/nanomdm/http"
	"github.com/jessepeterson/nanomdm/log"
	"github.com/jessepeterson/nanomdm/log/stdlogfmt"
	"github.com/jessepeterson/nanomdm/push/buford"
	pushsvc "github.com/jessepeterson/nanomdm/push/service"
	"github.com/jessepeterson/nanomdm/service"
	"github.com/jessepeterson/nanomdm/service/certauth"
	"github.com/jessepeterson/nanomdm/service/dump"
	"github.com/jessepeterson/nanomdm/service/microwebhook"
	"github.com/jessepeterson/nanomdm/service/multi"
	"github.com/jessepeterson/nanomdm/service/nanomdm"
	"github.com/jessepeterson/nanomdm/storage"
	"github.com/jessepeterson/nanomdm/storage/file"
	"github.com/jessepeterson/nanomdm/storage/mysql"
)

// overridden by -ldflags -X
var version = "unknown"

func main() {
	var (
		flDSN        = flag.String("dsn", "", "SQL data source name (connection string)")
		flFileDBPath = flag.String("db", "db", "Path of file storage directory (if used)")
		flListen     = flag.String("listen", ":9000", "HTTP listen address")
		flAPIKey     = flag.String("api", "", "API key for API endpoints")
		flVersion    = flag.Bool("version", false, "print version")
		flRootsPath  = flag.String("ca", "", "path to CA cert for verification")
		flWebhook    = flag.String("webhook-url", "", "URL to send requests to")
		flCertHeader = flag.String("cert-header", "", "HTTP header containing URL-escaped TLS client certificate")
		flDebug      = flag.Bool("debug", false, "log debug messages")
		flDump       = flag.Bool("dump", false, "dump MDM requests and responses to stdout")

		flMDMEndpoint = flag.String("endpoint-mdm", "/mdm", "HTTP endpoint for MDM commands")
		flCIEndpoint  = flag.String("endpoint-checkin", "", "HTTP endpoint for MDM check-ins")
		flMigEndpoint = flag.String("endpoint-checkin-migration", "", "HTTP endpoint for migration MDM check-ins")
		flRetro       = flag.Bool("retro", false, "Allow retroactive certificate-authorization association")
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	if *flMDMEndpoint == "" && *flCIEndpoint == "" && *flAPIKey == "" {
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

	var mdmStorage storage.AllStorage
	// select between our storage repositories
	if *flDSN != "" {
		mdmStorage, err = mysql.New(*flDSN, logger)
	} else {
		mdmStorage, err = file.New(*flFileDBPath)
	}
	if err != nil {
		stdlog.Fatal(err)
	}

	// create 'core' MDM service
	nano := nanomdm.New(mdmStorage, logger.With("service", "nanomdm"))

	mux := http.NewServeMux()

	if *flMDMEndpoint != "" || *flCIEndpoint != "" {
		var mdmService service.CheckinAndCommandService
		mdmService = nano
		if *flWebhook != "" {
			webhookService := microwebhook.New(*flWebhook)
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

		if *flMDMEndpoint != "" {
			// register 'core' MDM HTTP handler
			var mdmHandler http.Handler
			if *flCIEndpoint == "" {
				// if we don't specify a separate check-in handler, do it all
				// in the MDM endpoint
				mdmHandler = mdmhttp.CheckinAndCommandHandlerFunc(mdmService, logger.With("handler", "checkin-command"))
			} else {
				mdmHandler = mdmhttp.CommandAndReportResultsHandlerFunc(mdmService, logger.With("handler", "command"))
			}
			mdmHandler = mdmhttp.CertVerifyMiddleware(mdmHandler, verifier, logger.With("handler", "cert-verify"))
			if *flCertHeader != "" {
				mdmHandler = mdmhttp.CertExtractPEMHeaderMiddleware(mdmHandler, *flCertHeader, logger.With("handler", "cert-extract"))
			} else {
				mdmHandler = mdmhttp.CertExtractMdmSignatureMiddleware(mdmHandler, logger.With("handler", "cert-extract"))
			}
			mux.Handle(*flMDMEndpoint, mdmHandler)
		}

		if *flCIEndpoint != "" {
			// if we specified a separate check-in handler, set it up
			var checkinHandler http.Handler
			checkinHandler = mdmhttp.CheckinHandlerFunc(mdmService, logger.With("handler", "checkin"))
			checkinHandler = mdmhttp.CertVerifyMiddleware(checkinHandler, verifier, logger.With("handler", "cert-verify"))
			if *flCertHeader != "" {
				checkinHandler = mdmhttp.CertExtractPEMHeaderMiddleware(checkinHandler, *flCertHeader, logger.With("handler", "cert-extract"))
			} else {
				checkinHandler = mdmhttp.CertExtractMdmSignatureMiddleware(checkinHandler, logger.With("handler", "cert-extract"))
			}
			mux.Handle(*flCIEndpoint, checkinHandler)
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
		mux.Handle("/v1/pushcert", pushCertHandler)

		// register API handler for push notifications.
		// we strip the prefix to use the path as an id.
		const pushPrefix = "/v1/push/"
		var pushHandler http.Handler
		pushHandler = mdmhttp.PushHandlerFunc(pushService, logger.With("handler", "push"))
		pushHandler = http.StripPrefix(pushPrefix, pushHandler)
		pushHandler = basicAuth(pushHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(pushPrefix, pushHandler)

		// register API handler for new command queueing.
		// we strip the prefix to use the path as an id.
		const enqueuePrefix = "/v1/enqueue/"
		var enqueueHandler http.Handler
		enqueueHandler = mdmhttp.RawCommandEnqueueHandler(mdmStorage, pushService, logger.With("handler", "enqueue"))
		enqueueHandler = http.StripPrefix(enqueuePrefix, enqueueHandler)
		enqueueHandler = basicAuth(enqueueHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(enqueuePrefix, enqueueHandler)

		if *flMigEndpoint != "" {
			// setup a "migration" handler that takes Check-In messages
			// without bothering with certificate auth or other
			// middleware.
			//
			// if the source MDM can put together enough of an
			// authenticate and tokenupdate message to effectively
			// generate "enrollments" then
			var migHandler http.Handler
			migHandler = mdmhttp.CheckinHandlerFunc(nano, logger.With("handler", "migration"))
			migHandler = basicAuth(migHandler, apiUsername, *flAPIKey, "nanomdm")
			mux.Handle(*flMigEndpoint, migHandler)
		}
	}

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"` + version + `"}`))
	})

	logger.Info("msg", "starting server", "listen", *flListen)
	http.ListenAndServe(*flListen, simpleLog(mux, logger.With("handler", "log")))
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
