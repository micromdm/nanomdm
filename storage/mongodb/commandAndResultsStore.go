package mongodb

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PendingCommand struct {
	UUID            string       `bson:"uuid,omitempty"`
	EnrollmentUDID  string       `bson:"enrollment_udid,omitempty"`
	EntryTimestamp  string       `bson:"entry_timestamp,omitempty"`
	Status          string       `bson:"status,omitempty"`
	StatusTimestamp string       `bson:"status_timestamp,omitempty"`
	Command         *mdm.Command `bson:"command,omitempty"`
}

type CompletedCommand struct {
	Command   *PendingCommand     `bson:"command,omitempty"`
	Result    *mdm.CommandResults `bson:"result,omitempty"`
	Timestamp string              `bson:"timestamp,omitempty"`
}

const (
	CommandStatusNotNow = "NotNow"
	CommandStatusIdle   = "Idle"
)

func (m MongoDBStorage) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	filter := bson.M{
		"uuid":            report.CommandUUID,
		"enrollment_udid": r.ID,
	}

	if report.Status == CommandStatusNotNow || report.Status == CommandStatusIdle {
		update := bson.M{
			"$set": PendingCommand{
				Status:          report.Status,
				StatusTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
			},
		}

		_, err := m.CommandPendingCollection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return err
		}

		return nil
	}

	command := &PendingCommand{}
	err := m.CommandPendingCollection.FindOneAndDelete(context.TODO(), filter).Decode(command)
	if err != nil {
		return err
	}

	_, err = m.CommandResultCollection.InsertOne(context.TODO(), CompletedCommand{
		Command:   command,
		Result:    report,
		Timestamp: strconv.FormatInt(time.Now().Unix(), 10),
	})
	if err != nil {
		return err
	}

	return nil
}

func (m MongoDBStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	filter := bson.M{
		"enrollment_udid": r.ID,
	}
	if skipNotNow {
		filter = bson.M{
			"enrollment_udid": r.ID,
			"status": bson.M{
				"$ne": CommandStatusNotNow,
			},
		}
	}

	earliestSort := bson.M{
		"$natural": 0,
	}

	// Returning in LIFO ordering to ensure a single blocking command does not fail future commands
	command := &PendingCommand{}
	err := m.CommandPendingCollection.FindOne(context.TODO(), filter, options.FindOne().SetSort(earliestSort)).Decode(command)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	return command.Command, nil
}

func (m MongoDBStorage) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	commands := []interface{}{}

	for _, id := range ids {
		commands = append(commands, PendingCommand{
			EnrollmentUDID: id,
			UUID:           cmd.CommandUUID,
			Command:        cmd,
			EntryTimestamp: strconv.FormatInt(time.Now().Unix(), 10),
		})
	}

	_, err := m.CommandPendingCollection.InsertMany(context.TODO(), commands)
	if err != nil {
		return nil, err
	}

	// TODO (feature) - Perform validation on the inserted commands if not all commands were created successfully

	return nil, nil
}

func (m MongoDBStorage) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}

	childEnrollments, err := m.listChildEnrollments(r.ID)
	if err != nil {
		return err
	}
	allEnrollments := []string{r.ID}
	allEnrollments = append(allEnrollments, childEnrollments...)

	filter := bson.M{
		"enrollment_udid": bson.M{
			"$in": allEnrollments,
		},
	}

	// TODO (feature) - Archive deleted commands
	_, err = m.CommandPendingCollection.DeleteMany(context.Background(), filter)
	if err != nil {
		return err
	}

	return nil
}
