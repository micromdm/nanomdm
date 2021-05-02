package mysql

import (
	"context"
	"crypto/tls"
	"strconv"

	"github.com/jessepeterson/nanomdm/cryptoutil"
)

func (s *MySQLStorage) RetrievePushCert(ctx context.Context, topic string) (*tls.Certificate, string, error) {
	var certPEM, keyPEM []byte
	var staleToken int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT cert_pem, key_pem, stale_token FROM push_certs WHERE topic = ?`,
		topic,
	).Scan(&certPEM, &keyPEM, &staleToken)
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, "", err
	}
	return &cert, strconv.Itoa(staleToken), err
}

func (s *MySQLStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	var staleTokenInt, dbStaleToken int
	staleTokenInt, err := strconv.Atoi(staleToken)
	if err != nil {
		return true, err
	}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT stale_token FROM push_certs WHERE topic = ?`,
		topic,
	).Scan(&dbStaleToken)
	return dbStaleToken != staleTokenInt, err
}

func (s *MySQLStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}
	exists, err := s.queryRowContextRowExists(
		ctx,
		`SELECT COUNT(*) FROM push_certs WHERE topic = ?`,
		topic,
	)
	if err != nil {
		return err
	}
	if exists {
		_, err = s.db.ExecContext(
			ctx,
			`UPDATE push_certs SET cert_pem = ?, key_pem = ?, stale_token = stale_token + 1 WHERE topic = ?`,
			pemCert, pemKey, topic,
		)
	} else {
		_, err = s.db.ExecContext(
			ctx,
			`INSERT INTO push_certs (topic, cert_pem, key_pem, stale_token) VALUES (?, ?, ?, 0)`,
			topic, pemCert, pemKey,
		)
	}
	return err
}
