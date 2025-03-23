package multi

import (
	"context"
	"sync"
	"testing"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/test"
)

type ctxTest1 struct{}

type testSvc struct {
	wg      *sync.WaitGroup
	capture string
	service.CheckinAndCommandService
}

func (ts *testSvc) Authenticate(r *mdm.Request, _ *mdm.Authenticate) error {
	ts.capture, _ = r.Context.Value(&ctxTest1{}).(string)
	ts.wg.Done()
	return nil
}

func TestContextPassthru(t *testing.T) {
	nopSvc1 := &test.NopService{}

	var ctx context.Context = context.Background()

	ctx = context.WithValue(ctx, &ctxTest1{}, "test-ctx-val")

	r := &mdm.Request{Context: ctx}

	var wg sync.WaitGroup

	wg.Add(1)
	ts := &testSvc{
		wg:                       &wg,
		CheckinAndCommandService: &test.NopService{},
	}

	multi := New(log.NopLogger, nopSvc1, ts)

	err := multi.Authenticate(r, &mdm.Authenticate{})
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if have, want := ts.capture, "test-ctx-val"; have != want {
		t.Errorf("have: %v, want: %v", have, want)
	}
}
