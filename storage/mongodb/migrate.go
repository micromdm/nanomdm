package mongodb

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
)

func (m MongoDBStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {

	// Devices
	deviceFilter := bson.M{
		"parent": bson.TypeNull,
	}

	deviceCursor, err := m.CheckinCollection.Find(ctx, deviceFilter)
	if err != nil {
		return err
	}

	deviceEnrollments := []DeviceCheckinRecord{}
	if err := deviceCursor.All(ctx, &deviceEnrollments); err != nil {
		return err
	}

	for _, deviceEnrollment := range deviceEnrollments {
		if deviceEnrollment.AuthenticateRequest != "" {
			msg, err := mdm.DecodeCheckin([]byte(deviceEnrollment.AuthenticateRequest))
			if err != nil {
				c <- err
				continue
			}
			c <- msg
		} else {
			continue
		}

		if deviceEnrollment.TokenUpdate != "" {
			msg, err := mdm.DecodeCheckin([]byte(deviceEnrollment.TokenUpdate))
			if err != nil {
				c <- err
				continue
			}
			c <- msg
		} else {
			continue
		}
	}

	// Users
	userFilter := bson.M{
		"parent": bson.M{
			"$ne": bson.TypeNull,
		},
	}

	userCursor, err := m.CheckinCollection.Find(ctx, userFilter)
	if err != nil {
		return err
	}

	userEnrollments := []DeviceCheckinRecord{}
	if err := userCursor.All(ctx, &userEnrollments); err != nil {
		return err
	}

	for _, userEnrollments := range userEnrollments {
		if userEnrollments.UserAuthenticateRequest != "" {
			msg, err := mdm.DecodeCheckin([]byte(userEnrollments.UserAuthenticateRequest))
			if err != nil {
				c <- err
				continue
			}
			c <- msg
		} else {
			continue
		}

		if userEnrollments.TokenUpdate != "" {
			msg, err := mdm.DecodeCheckin([]byte(userEnrollments.TokenUpdate))
			if err != nil {
				c <- err
				continue
			}
			c <- msg
		} else {
			continue
		}
	}

	return nil
}
