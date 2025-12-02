package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/storage"
)

// readPEMCertAndKey reads a PEM-encoded certificate and non-encrypted
// private key from input bytes and returns the separate PEM certificate
// and private key in cert and key respectively.
func readPEMCertAndKey(input []byte) (cert []byte, key []byte, err error) {
	// if the PEM blocks are mushed together with no newline then add one
	input = bytes.ReplaceAll(input, []byte("----------"), []byte("-----\n-----"))
	var block *pem.Block
	for {
		block, input = pem.Decode(input)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert = pem.EncodeToMemory(block)
		} else if block.Type == "PRIVATE KEY" || strings.HasSuffix(block.Type, " PRIVATE KEY") {
			if x509.IsEncryptedPEMBlock(block) {
				err = errors.New("private key PEM appears to be encrypted")
				break
			}
			key = pem.EncodeToMemory(block)
		} else {
			err = fmt.Errorf("unrecognized PEM type: %q", block.Type)
			break
		}
	}
	return
}

// StorePushCertHandler reads a PEM-encoded certificate and private
// key from the HTTP body and saves it to storage. This effectively
// enables us to do something like:
// "% cat push.pem push.key | curl -T - http://api.example.com/" to
// upload our push certs.
func StorePushCertHandler(storage storage.PushCertStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)

		// read the body of the request
		b, err := io.ReadAll(r.Body)
		if err != nil {
			logAndWriteJSONError(logger, w, "reading body", err, 0)
			return
		}

		// parse for the two separate cert and key PEM blocks
		certPEM, keyPEM, err := readPEMCertAndKey(b)
		if err != nil {
			logAndWriteJSONError(logger, w, "reading PEM cert and key", err, http.StatusBadRequest)
			return
		}

		// sanity check the provided cert and key to make sure they're usable as a pair.
		_, err = tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			logAndWriteJSONError(logger, w, "parse X509 key pair", err, http.StatusBadRequest)
			return
		}

		// parse the certificate (to get the data out of it)
		cert, err := cryptoutil.DecodePEMCertificate(certPEM)
		if err != nil {
			logAndWriteJSONError(logger, w, "decode PEM cert", err, http.StatusBadRequest)
			return
		}

		// get the topic from the certificate
		topic, err := cryptoutil.TopicFromCert(cert)
		if err != nil {
			logAndWriteJSONError(logger, w, "topic from cert", err, http.StatusBadRequest)
			return
		}

		// store the push cert and key
		err = storage.StorePushCert(r.Context(), certPEM, keyPEM)
		if err != nil {
			logAndWriteJSONError(logger, w, "store push cert", err, 0)
			return
		}

		// debug log our success
		logger.Debug("msg", "stored push cert", "topic", topic)

		// JSON API response
		out := &PushCertResponseJson{
			Topic:    topic,
			NotAfter: cert.NotAfter,
		}

		writeJSON(w, out, http.StatusOK, logger)
	}
}
