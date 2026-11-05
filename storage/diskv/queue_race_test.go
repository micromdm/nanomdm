package diskv

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
)

// A device acks the only queued command while another command is enqueued.
// Afterwards the second command must still be retrievable.
func TestQueueEnqueueVsAckRace(t *testing.T) {
	s := New(t.TempDir())
	ctx := context.Background()
	r := mdm.NewRequestWithContext(ctx, nil)
	r.EnrollID = &mdm.EnrollID{Type: mdm.Device, ID: "dev1"}

	cmd := func(u string) *mdm.Command {
		c := &mdm.Command{CommandUUID: u, Raw: []byte("<plist/>")}
		c.Command.RequestType = "DeviceInformation"
		return c
	}
	ack := func(u string) error {
		return s.StoreCommandReport(r, &mdm.CommandResults{CommandUUID: u, Status: "Acknowledged"})
	}

	for i := 0; i < 500; i++ {
		a, b := fmt.Sprintf("a-%d", i), fmt.Sprintf("b-%d", i)
		if _, err := s.EnqueueCommand(ctx, []string{"dev1"}, cmd(a)); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); s.EnqueueCommand(ctx, []string{"dev1"}, cmd(b)) }()
		go func() { defer wg.Done(); ack(a) }()
		wg.Wait()

		next, err := s.RetrieveNextCommand(r, false)
		if err != nil {
			t.Fatal(err)
		}
		if next == nil || next.CommandUUID != b {
			t.Fatalf("iteration %d: queued %s is unreachable (next=%v): the queue lost its head", i, b, next)
		}
		if err := ack(b); err != nil {
			t.Fatal(err)
		}
	}
}
