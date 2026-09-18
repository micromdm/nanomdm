package multi

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

type mockDelayedService struct {
	service.NopService
	delay     time.Duration
	completed atomic.Bool
	callCount atomic.Int32
}

func (m *mockDelayedService) Authenticate(r *mdm.Request, a *mdm.Authenticate) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.completed.Store(true)
	m.callCount.Add(1)
	return nil
}

func (m *mockDelayedService) TokenUpdate(r *mdm.Request, t *mdm.TokenUpdate) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.completed.Store(true)
	m.callCount.Add(1)
	return nil
}

func (m *mockDelayedService) CheckOut(r *mdm.Request, c *mdm.CheckOut) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.completed.Store(true)
	m.callCount.Add(1)
	return nil
}

func (m *mockDelayedService) CommandAndReportResults(r *mdm.Request, res *mdm.CommandResults) (*mdm.Command, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.completed.Store(true)
	m.callCount.Add(1)
	return nil, nil
}

func TestMultiService_WaitAll(t *testing.T) {
	primary := &service.NopService{}
	secondary := &mockDelayedService{delay: 30 * time.Millisecond}

	ms := New(log.NopLogger, primary, secondary)
	ms.WaitAll = true

	req := mdm.NewRequestWithContext(context.Background(), nil)
	auth := &mdm.Authenticate{}

	err := ms.Authenticate(req, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !secondary.completed.Load() {
		t.Fatal("expected secondary service to be completed when WaitAll is true")
	}
}

func TestMultiService_DefaultNoWait(t *testing.T) {
	primary := &service.NopService{}
	secondary := &mockDelayedService{delay: 50 * time.Millisecond}

	ms := New(log.NopLogger, primary, secondary)
	if ms.WaitAll {
		t.Fatal("expected WaitAll to be false by default")
	}

	req := mdm.NewRequestWithContext(context.Background(), nil)
	auth := &mdm.Authenticate{}

	err := ms.Authenticate(req, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Because WaitAll is false, Authenticate should return before the 50ms sleep finishes
	if secondary.completed.Load() {
		t.Fatal("expected secondary service to NOT be completed yet when WaitAll is false")
	}
}

func TestMultiService_WithWaitAll(t *testing.T) {
	primary := &service.NopService{}
	ms := New(log.NopLogger, primary).WithWaitAll()
	if !ms.WaitAll {
		t.Fatal("expected WithWaitAll() to set WaitAll to true")
	}
}

func TestMultiService_WaitAllSingleService(t *testing.T) {
	primary := &service.NopService{}
	ms := New(log.NopLogger, primary)
	ms.WaitAll = true

	req := mdm.NewRequestWithContext(context.Background(), nil)
	auth := &mdm.Authenticate{}

	// Single service should not deadlock or panic
	err := ms.Authenticate(req, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMultiService_WaitAllMultipleServices(t *testing.T) {
	primary := &service.NopService{}
	s1 := &mockDelayedService{delay: 20 * time.Millisecond}
	s2 := &mockDelayedService{delay: 30 * time.Millisecond}
	s3 := &mockDelayedService{delay: 10 * time.Millisecond}

	ms := New(log.NopLogger, primary, s1, s2, s3)
	ms.WaitAll = true

	req := mdm.NewRequestWithContext(context.Background(), nil)
	auth := &mdm.Authenticate{}

	err := ms.Authenticate(req, auth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s1.completed.Load() || !s2.completed.Load() || !s3.completed.Load() {
		t.Fatal("expected all secondary services to be completed when WaitAll is true")
	}
	if s1.callCount.Load() != 1 || s2.callCount.Load() != 1 || s3.callCount.Load() != 1 {
		t.Fatalf("expected each secondary service to be called exactly once")
	}
}

func TestMultiService_WaitAllTokenUpdateAndCommands(t *testing.T) {
	primary := &service.NopService{}
	secondary := &mockDelayedService{delay: 20 * time.Millisecond}

	ms := New(log.NopLogger, primary, secondary).WithWaitAll()
	req := mdm.NewRequestWithContext(context.Background(), nil)

	// Test TokenUpdate
	secondary.completed.Store(false)
	if err := ms.TokenUpdate(req, &mdm.TokenUpdate{}); err != nil {
		t.Fatalf("TokenUpdate error: %v", err)
	}
	if !secondary.completed.Load() {
		t.Fatal("expected secondary service to be completed for TokenUpdate")
	}

	// Test CheckOut
	secondary.completed.Store(false)
	if err := ms.CheckOut(req, &mdm.CheckOut{}); err != nil {
		t.Fatalf("CheckOut error: %v", err)
	}
	if !secondary.completed.Load() {
		t.Fatal("expected secondary service to be completed for CheckOut")
	}

	// Test CommandAndReportResults
	secondary.completed.Store(false)
	if _, err := ms.CommandAndReportResults(req, &mdm.CommandResults{}); err != nil {
		t.Fatalf("CommandAndReportResults error: %v", err)
	}
	if !secondary.completed.Load() {
		t.Fatal("expected secondary service to be completed for CommandAndReportResults")
	}
}
