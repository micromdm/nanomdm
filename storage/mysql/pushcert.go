package mysql

import (
	"context"
	"crypto/tls"
	"strconv"

	"github.com/micromdm/nanomdm/cryptoutil"
)

func (s *MySQLStorage) RetrievePushCert(ctx context.Context, topic string) (*tls.Certificate, string, error) {
	row, err := s.q.RetrievePushCert(ctx, topic)
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair([]byte(row.CertPem), []byte(row.KeyPem))
	if err != nil {
		return nil, "", err
	}
	return &cert, strconv.Itoa(int(row.StaleToken)), err
}

func (s *MySQLStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	var staleTokenInt, dbStaleToken int
	staleTokenInt, err := strconv.Atoi(staleToken)
	if err != nil {
		return true, err
	}
	err = s.db.QueryRowContext(
		ctx,
		`SELECT stale_token FROM push_certs WHERE topic = ?;`,
		topic,
	).Scan(&dbStaleToken)
	return dbStaleToken != staleTokenInt, err
}

func (s *MySQLStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(
		ctx, `
INSERT INTO push_certs
    (topic, cert_pem, key_pem, stale_token)
VALUES
    (?, ?, ?, 0) AS new
ON DUPLICATE KEY
UPDATE
    cert_pem = new.cert_pem,
    key_pem = new.key_pem,
    push_certs.stale_token = push_certs.stale_token + 1;`,
		topic, pemCert, pemKey,
	)
	return err
}
