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

	mdmhttp "github.com/jessepeterson/nanomdm/http"
	"github.com/jessepeterson/nanomdm/log"
	stdlogadapter "github.com/jessepeterson/nanomdm/log/stdlog"
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

// AllStorage represents all the required storage by NanoMDM
type AllStorage interface {
	storage.ServiceStore
	storage.PushStore
	storage.PushCertStore
	storage.CommandEnqueuer
	storage.CertAuthStore
}

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
	)
	flag.Parse()

	if *flVersion {
		fmt.Println(version)
		return
	}

	logger := stdlogadapter.New(stdlog.Default(), *flDebug)

	if *flRootsPath == "" {
		stdlog.Fatal("must supply CA cert path flag")
	}
	caPEM, err := ioutil.ReadFile(*flRootsPath)
	if err != nil {
		stdlog.Fatal(err)
	}
	verifier, err := NewVerifier(caPEM, x509.ExtKeyUsageClientAuth)
	if err != nil {
		stdlog.Fatal(err)
	}

	var mdmStorage AllStorage
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
	var mdmService service.CheckinAndCommandService
	mdmService = nanomdm.New(mdmStorage, logger)
	if *flWebhook != "" {
		webhookService := microwebhook.New(*flWebhook)
		mdmService = multi.New(logger, mdmService, webhookService)
	}
	mdmService = certauth.NewCertAuthMiddleware(mdmService, mdmStorage, logger)
	if *flDump {
		mdmService = dump.New(mdmService, os.Stdout)
	}

	mux := http.NewServeMux()

	// register 'core' MDM HTTP handler
	var mdmHandler http.Handler
	if *flCIEndpoint == "" {
		// if we don't specify a separate check-in handler, do it all
		// in the MDM endpoint
		mdmHandler = mdmhttp.CheckinAndCommandHandlerFunc(mdmService, logger)
	} else {
		mdmHandler = mdmhttp.CommandAndReportResultsHandlerFunc(mdmService, logger)
	}
	mdmHandler = mdmhttp.CertVerifyMiddleware(mdmHandler, verifier, logger)
	if *flCertHeader != "" {
		mdmHandler = mdmhttp.CertExtractPEMHeaderMiddleware(mdmHandler, *flCertHeader, logger)
	} else {
		mdmHandler = mdmhttp.CertExtractMdmSignatureMiddleware(mdmHandler, logger)
	}
	mux.Handle(*flMDMEndpoint, mdmHandler)

	if *flCIEndpoint != "" {
		// if we specified a separate check-in handler, set it up
		var checkinHandler http.Handler
		checkinHandler = mdmhttp.CheckinHandlerFunc(mdmService, logger)
		checkinHandler = mdmhttp.CertVerifyMiddleware(checkinHandler, verifier, logger)
		if *flCertHeader != "" {
			checkinHandler = mdmhttp.CertExtractPEMHeaderMiddleware(checkinHandler, *flCertHeader, logger)
		} else {
			checkinHandler = mdmhttp.CertExtractMdmSignatureMiddleware(checkinHandler, logger)
		}
		mux.Handle(*flCIEndpoint, checkinHandler)
	}

	if *flAPIKey != "" {
		const apiUsername = "nanomdm"

		// create our push provider and push service
		pushProviderFactory := buford.NewPushProviderFactory()
		pushService := pushsvc.New(mdmStorage, mdmStorage, pushProviderFactory, logger)

		// register API handler for push cert storage/upload.
		var pushCertHandler http.Handler
		pushCertHandler = mdmhttp.StorePushCertHandlerFunc(mdmStorage, logger)
		pushCertHandler = basicAuth(pushCertHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle("/v1/pushcert", pushCertHandler)

		// register API handler for push notifications.
		// we strip the prefix to use the path as an id.
		const pushPrefix = "/v1/push/"
		var pushHandler http.Handler
		pushHandler = mdmhttp.PushHandlerFunc(pushService, logger)
		pushHandler = http.StripPrefix(pushPrefix, pushHandler)
		pushHandler = basicAuth(pushHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(pushPrefix, pushHandler)

		// register API handler for new command queueing.
		// we strip the prefix to use the path as an id.
		const enqueuePrefix = "/v1/enqueue/"
		var enqueueHandler http.Handler
		enqueueHandler = mdmhttp.RawCommandEnqueueHandler(mdmStorage, pushService, logger)
		enqueueHandler = http.StripPrefix(enqueuePrefix, enqueueHandler)
		enqueueHandler = basicAuth(enqueueHandler, apiUsername, *flAPIKey, "nanomdm")
		mux.Handle(enqueuePrefix, enqueueHandler)
	}

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"` + version + `"}`))
	})

	logger.Info("msg", "starting server", "listen", *flListen)
	http.ListenAndServe(*flListen, simpleLog(mux, logger))
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
