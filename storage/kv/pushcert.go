package kv

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/cryptoutil"
)

const (
	keyPushCertPEM        = "pem"
	keyPushCertKey        = "key"
	keyPushCertStaleToken = "stale_token"
)

// RetrievePushCert validates the freshness of the APNs push cert with topic from the KV store.
func (s *KV) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	tokenBytes, err := s.pushCert.Get(ctx, join(topic, keyPushCertStaleToken))
	return staleToken != string(tokenBytes), err
}

// RetrievePushCert retrieves the TLS certificate and private key from the KV store.
func (s *KV) RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error) {
	getMap, err := kv.GetMap(
		ctx,
		s.pushCert,
		[]string{
			join(topic, keyPushCertPEM),
			join(topic, keyPushCertKey),
			join(topic, keyPushCertStaleToken),
		},
	)
	if err != nil {
		return nil, "", err
	}

	tlsCert, err := tls.X509KeyPair(getMap[join(topic, keyPushCertPEM)], getMap[join(topic, keyPushCertKey)])
	if err != nil {
		return nil, "", err
	}

	return &tlsCert, string(getMap[join(topic, keyPushCertStaleToken)]), nil
}

// StorePushCert stores pemCert and pemKey by APNs topic in the KV store.
func (s *KV) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}
	return kv.PerformCRUDBucketTxn(ctx, s.pushCert, func(ctx context.Context, b kv.CRUDBucket) error {
		var token int

		if tokenBytes, err := b.Get(ctx, join(topic, keyPushCertStaleToken)); err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
			return fmt.Errorf("getting token for topic: %s: %w", topic, err)
		} else if err == nil {
			token, err = strconv.Atoi(string(tokenBytes))
			if err != nil {
				// token strconv error: eat the error and reset the token
				token = 0
			} else {
				token++
			}
		}

		return kv.SetMap(ctx, b, map[string][]byte{
			join(topic, keyPushCertPEM):        pemCert,
			join(topic, keyPushCertKey):        pemKey,
			join(topic, keyPushCertStaleToken): []byte(strconv.Itoa(token)),
		})
	})
}
