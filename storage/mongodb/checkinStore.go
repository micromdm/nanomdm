package mongodb

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DeviceCheckinRecord struct {
	UDID                    string            `bson:"udid,omitempty"`
	SerialNumber            string            `bson:"serial_number,omitempty"`
	AuthenticateRequest     string            `bson:"authenticate_request,omitempty"`
	Identity                *x509.Certificate `bson:"identity,omitempty"`
	UnlockToken             []byte            `bson:"unlock_token,omitempty"`
	Children                []string          `bson:"children,omitempty"`
	Parent                  string            `bson:"parent,omitempty"`
	TokenUpdate             string            `bson:"token_update,omitempty"`
	TokenUpdateTally        int               `bson:"token_update_tally,omitempty"`
	Disabled                bool              `bson:"disabled,omitempty"`
	UserAuthenticateRequest string            `bson:"user_authenticate_request,omitempty"`
	DigestResponse          string            `bson:"digest_response,omitempty"`
}

func (m MongoDBStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	upsert := true
	filter := bson.M{
		"udid": r.ID,
	}
	update := bson.M{
		"$set": DeviceCheckinRecord{
			UDID:                r.ID,
			Identity:            r.Certificate,
			SerialNumber:        msg.SerialNumber,
			AuthenticateRequest: string(msg.Raw),
		},
	}

	_, err := m.CheckinCollection.UpdateOne(context.TODO(), filter, update, &options.UpdateOptions{Upsert: &upsert})

	return err
}

func (m MongoDBStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	record := DeviceCheckinRecord{
		UDID: r.ID,
	}

	if len(msg.UnlockToken) > 0 {
		record.UnlockToken = msg.UnlockToken
	}

	if r.ParentID != "" {
		filter := bson.M{
			"udid": r.ParentID,
		}
		update := bson.M{
			"$push": bson.M{
				"children": r.ID,
			},
		}

		res, err := m.CheckinCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return err
		}
		if res.MatchedCount != 1 {
			return errors.New("parent device enrollment missing on user token update")
		}
	}

	upsert := true
	filter := bson.M{
		"udid": r.ID,
	}
	update := bson.M{
		"$set": DeviceCheckinRecord{
			UDID:        r.ID,
			Parent:      r.ParentID,
			TokenUpdate: string(msg.Raw),
		},
		"$inc": DeviceCheckinRecord{
			TokenUpdateTally: 1,
		},
		"$unset": bson.M{
			"disabled": "",
		},
	}
	_, err := m.CheckinCollection.UpdateOne(context.TODO(), filter, update, &options.UpdateOptions{Upsert: &upsert})

	return err
}

func (m MongoDBStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	upsert := true
	filter := bson.M{
		"udid": r.ID,
	}
	update := bson.M{
		"$set": DeviceCheckinRecord{
			UDID:                    r.ID,
			UserAuthenticateRequest: string(msg.Raw),
			DigestResponse:          msg.DigestResponse,
		},
	}
	_, err := m.CheckinCollection.UpdateOne(context.TODO(), filter, update, &options.UpdateOptions{Upsert: &upsert})

	return err
}

func (m MongoDBStorage) Disable(r *mdm.Request) error {
	childEnrollments, err := m.listChildEnrollments(r.ParentID)
	if err != nil {
		return err
	}

	// Disable Child Enrollments
	disableUpdate := bson.M{
		"$set": DeviceCheckinRecord{
			Disabled: true,
		},
	}

	if len(childEnrollments) > 0 {
		childFilter := bson.M{
			"udid": bson.M{
				"$in": childEnrollments,
			},
		}

		_, err := m.CheckinCollection.UpdateOne(context.TODO(), childFilter, disableUpdate)
		if err != nil {
			return err
		}
	}

	// Disable Parent Enrollment
	parentFilter := bson.M{
		"udid": r.ID,
	}
	_, err = m.CheckinCollection.UpdateOne(context.TODO(), parentFilter, disableUpdate)

	return err
}

func (m MongoDBStorage) listChildEnrollments(udid string) ([]string, error) {

	filter := bson.M{
		"udid": udid,
	}

	parentRecord := &DeviceCheckinRecord{}
	res := m.CheckinCollection.FindOne(context.TODO(), filter)
	err := res.Decode(parentRecord)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return []string{}, nil
		}
		return []string{}, err
	}

	return parentRecord.Children, nil
}

func (m MongoDBStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	filter := bson.M{
		"udid": id,
	}

	res := DeviceCheckinRecord{}
	err := m.CheckinCollection.FindOne(ctx, filter).Decode(&res)

	return res.TokenUpdateTally, err
}
