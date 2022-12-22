package mongodb

import (
	"context"
	"crypto/tls"
	"strconv"
	"time"

	"github.com/micromdm/nanomdm/cryptoutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PushCertRecord struct {
	Timestamp   string `bson:"ts,omitempty"`
	Certificate string `bson:"certificate,omitempty"`
	PrivateKey  string `bson:"key,omitempty"`
	Topic       string `bson:"topic,omitempty"`
}

var latestSort = bson.M{
	"$natural": -1,
}

func (m MongoDBStorage) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	filter := bson.M{
		"topic": topic,
	}
	res := PushCertRecord{}
	err := m.PushCertCollection.FindOne(ctx, filter, options.FindOne().SetSort(latestSort)).Decode(&res)
	if err != nil {
		return false, err
	}
	return res.Timestamp == staleToken, nil
}

func (m MongoDBStorage) RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error) {
	filter := bson.M{
		"topic": topic,
	}
	res := PushCertRecord{}
	err = m.PushCertCollection.FindOne(ctx, filter, options.FindOne().SetSort(latestSort)).Decode(&res)
	if err != nil {
		return nil, "", err
	}

	pushCert, err := tls.X509KeyPair([]byte(res.Certificate), []byte(res.PrivateKey))
	if err != nil {
		return nil, "", err
	}

	return &pushCert, res.Timestamp, nil
}

func (m MongoDBStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}

	_, err = m.PushCertCollection.InsertOne(ctx, PushCertRecord{
		Timestamp:   strconv.FormatInt(time.Now().UnixNano(), 10),
		Certificate: string(pemCert),
		PrivateKey:  string(pemKey),
		Topic:       topic,
	})
	if err != nil {
		return err
	}

	return nil
}
