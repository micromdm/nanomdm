package mysql

import (
	"context"
	"strconv"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// RetrievePushCert selects the raw PEM cert and key for topic.
func (s *MySQLStorage) RetrievePushCert(ctx context.Context, topic string) ([]byte, []byte, string, error) {
	row, err := s.q.SelectPushCert(ctx, topic)
	if err != nil {
		return nil, nil, "", err
	}
	return row.CertPem, row.KeyPem, strconv.Itoa(int(row.StaleToken)), nil
}

// IsPushCertStale compares the staleToken against the selected stale token for topic.
func (s *MySQLStorage) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	dbStaleToken, err := s.q.SelectPushCertStaleToken(ctx, topic)
	if err != nil {
		return true, err
	}

	staleTokenInt, err := strconv.Atoi(staleToken)
	if err != nil {
		return true, err
	}

	return int(dbStaleToken) != staleTokenInt, err
}

// StorePushCert upserts pemCert and pemKey for pemCert's extracted topic.
func (s *MySQLStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}

	return s.q.UpsertPushCert(ctx, sqlc.UpsertPushCertParams{
		Topic:   topic,
		CertPem: pemCert,
		KeyPem:  pemKey,
	})
}
