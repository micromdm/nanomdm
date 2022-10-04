package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBStorage struct {
	MongoClient              *mongo.Client
	CheckinCollection        *mongo.Collection
	CommandResultCollection  *mongo.Collection
	CommandPendingCollection *mongo.Collection
	BootstrapTokenCollection *mongo.Collection
	PushCertCollection       *mongo.Collection
	CertAuthCollection       *mongo.Collection
}

// TODO (feature) - Enable configuration of these names
const (
	databaseName = "nanomdm"

	checkinStoreName        = "checkin_store"
	commandResultStoreName  = "command_result_store"
	commandPendingStoreName = "command_pending_store"
	bootstrapTokenStoreName = "bootstrap_token_store"
	pushCertStoreName       = "push_cert_store"
	certAuthStoreName       = "cert_auth_store"
)

func New(ctx context.Context, uri string, username string, password string) (*MongoDBStorage, error) {
	var err error
	storage := &MongoDBStorage{}

	mongoOpts := options.Client().ApplyURI(uri)
	mongoOpts.SetAuth(options.Credential{Username: username, Password: password})

	storage.MongoClient, err = mongo.NewClient(mongoOpts)
	if err != nil {
		return nil, err
	}

	err = storage.MongoClient.Connect(ctx)
	if err != nil {
		return nil, err
	}

	storage.CheckinCollection = storage.MongoClient.Database(databaseName).Collection(checkinStoreName)
	_, err = storage.CheckinCollection.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys: bson.M{
			"udid": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	storage.CommandResultCollection = storage.MongoClient.Database(databaseName).Collection(commandResultStoreName)
	storage.CommandPendingCollection = storage.MongoClient.Database(databaseName).Collection(commandPendingStoreName)
	_, err = storage.CheckinCollection.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
		{
			Keys: bson.M{
				"uuid": 1,
			},
		},
		{
			Keys: bson.M{
				"enrollment_udid": 2,
			},
		},
		{
			Keys: bson.M{
				"status": 3,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	storage.BootstrapTokenCollection = storage.MongoClient.Database(databaseName).Collection(checkinStoreName)
	_, err = storage.BootstrapTokenCollection.Indexes().CreateOne(context.TODO(), mongo.IndexModel{
		Keys: bson.M{
			"udid": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, err
	}

	storage.PushCertCollection = storage.MongoClient.Database(databaseName).Collection(pushCertStoreName)
	_, err = storage.PushCertCollection.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
		{
			Keys: bson.M{
				"ts": 1,
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{
				"topic": 2,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	storage.CertAuthCollection = storage.MongoClient.Database(databaseName).Collection(certAuthStoreName)
	_, err = storage.CertAuthCollection.Indexes().CreateMany(context.TODO(), []mongo.IndexModel{
		{
			Keys: bson.M{
				"enrollment_id": 1,
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.M{
				"cert_hash": 2,
			},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return nil, err
	}
	return storage, nil
}
