package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func migrate(t *testing.T, ctx context.Context, store storage.StoreMigrator, d *device) {
	checkins := make(chan interface{})
	var retrieveErr error
	go func() {
		// dispatch to our storage backend to start sending the checkins
		// channel our MDM check-in messages.
		retrieveErr = store.RetrieveMigrationCheckins(ctx, checkins)
		close(checkins)
	}()

	// checkin accumulator
	checkinAcc := make(map[string]map[string][]interface{})

	for checkin := range checkins {
		switch v := checkin.(type) {
		case *mdm.Authenticate:
			checkinAcc = appendCheckin(checkinAcc, v.UDID, v.MessageType.MessageType, checkin)
		case *mdm.TokenUpdate:
			checkinAcc = appendCheckin(checkinAcc, v.UDID, v.MessageType.MessageType, checkin)
		case *mdm.SetBootstrapToken:
			checkinAcc = appendCheckin(checkinAcc, v.UDID, v.MessageType.MessageType, checkin)
		case error:
			t.Errorf("error in migration checkins: %v", v)
		default:
			t.Error("unknown checkin type")
		}
	}

	if retrieveErr != nil {
		t.Fatal(retrieveErr)
	}

	id := d.EnrollID().ID

	checkin, err := get1CheckinForIDType(checkinAcc, id, "Authenticate")
	if err != nil {
		t.Fatal(err)
	}

	if auth, ok := checkin.(*mdm.Authenticate); !ok {
		t.Error("invalid type")
	} else {
		if have, want := auth.SerialNumber, d.SerialNumber(); have != want {
			t.Errorf("have: %s, want: %s", have, want)
		}
	}

	checkin, err = get1CheckinForIDType(checkinAcc, id, "TokenUpdate")
	if err != nil {
		t.Fatal(err)
	}

	if tokUpd, ok := checkin.(*mdm.TokenUpdate); !ok {
		t.Error("invalid type")
	} else {
		if have, want := tokUpd.PushMagic, d.GetPush().PushMagic; have != want {
			t.Errorf("have: %s, want: %s", have, want)
		}
	}

	/*
		checkin, err = get1CheckinForIDType(checkinAcc, id, "SetBootstrapToken")
		if err != nil {
			t.Fatal(err)
		}

		if bsTok, ok := checkin.(*mdm.SetBootstrapToken); !ok {
			t.Error("invalid type")
		} else {
			// "hello world" comes from bstoken.go test: the token passed to the device
			hw := base64.StdEncoding.EncodeToString([]byte("hello world"))
			if have, want := string(bsTok.BootstrapToken.BootstrapToken), hw; have != want {
				t.Errorf("have: %s, want: %s", have, want)
			}
		}
	*/

}

func get1CheckinForIDType(in map[string]map[string][]interface{}, id, messageType string) (interface{}, error) {
	if _, ok := in[id]; !ok {
		return nil, fmt.Errorf("id not found: %s", id)
	}

	checkins, ok := in[id][messageType]
	if !ok {
		return nil, fmt.Errorf("message type for id %s not found: %s", id, messageType)
	}

	if have, want := len(checkins), 1; have != want {
		return nil, fmt.Errorf("checkin length: have: %d, want: %d", have, want)
	}

	return checkins[0], nil
}

func appendCheckin(in map[string]map[string][]interface{}, id, messageType string, checkin interface{}) map[string]map[string][]interface{} {
	if _, ok := in[id]; !ok {
		in[id] = make(map[string][]interface{})
	}
	in[id][messageType] = append(in[id][messageType], checkin)
	return in
}
