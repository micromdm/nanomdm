package mongodb

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BootstrapTokenRecord struct {
	UDID           string `bson:"udid,omitempty"`
	BootstrapToken []byte `bson:"bootstrap_token,omitempty"`
}

func (m MongoDBStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	upsert := true
	filter := bson.M{
		"udid": r.ID,
	}
	update := bson.M{
		"$set": BootstrapTokenRecord{
			UDID:           r.ID,
			BootstrapToken: msg.BootstrapToken.BootstrapToken,
		},
	}

	_, err := m.BootstrapTokenCollection.UpdateOne(context.TODO(), filter, update, &options.UpdateOptions{Upsert: &upsert})

	return err
}

func (m MongoDBStorage) RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	filter := bson.M{
		"udid": r.ID,
	}

	tokenRecord := &BootstrapTokenRecord{}
	err := m.BootstrapTokenCollection.FindOne(context.TODO(), filter).Decode(tokenRecord)
	if err != nil {
		return nil, err
	}
	return &mdm.BootstrapToken{
		BootstrapToken: tokenRecord.BootstrapToken,
	}, nil
}
