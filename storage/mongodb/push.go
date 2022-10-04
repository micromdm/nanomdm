package mongodb

import (
	"context"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
)

func (m MongoDBStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	pushInfos := make(map[string]*mdm.Push)

	filter := bson.M{
		"udid": bson.M{
			"$in": ids,
		},
	}
	cursor, err := m.CheckinCollection.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	records := []DeviceCheckinRecord{}
	err = cursor.All(context.TODO(), &records)
	if err != nil {
		return nil, err
	}

	for _, record := range records {
		msg, err := mdm.DecodeCheckin([]byte(record.TokenUpdate))
		if err != nil {
			return nil, err
		}

		message, ok := msg.(*mdm.TokenUpdate)
		if !ok {
			return nil, errors.New("saved TokenUpdate is not a TokenUpdate")
		}

		pushInfos[record.UDID] = &message.Push
	}
	return pushInfos, nil
}
