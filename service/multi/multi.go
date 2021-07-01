// Package multi contains a multi-service dispatcher.
package multi

import (
	"context"

	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
)

// MultiService executes multiple services for the same service calls.
// The first service returns values or errors to the caller. We give the
// first service a chance to alter any 'core' request data (say, the
// Enrollment ID) by waiting for it to finish then we run the remaining
// services' calls in parallel.
type MultiService struct {
	logger log.Logger
	svcs   []service.CheckinAndCommandService
}

func New(logger log.Logger, svcs ...service.CheckinAndCommandService) *MultiService {
	if len(svcs) < 1 {
		panic("must supply at least one service")
	}
	return &MultiService{logger: logger, svcs: svcs}
}

// RequestWithContext returns a clone of r and sets its context to ctx.
func RequestWithContext(r *mdm.Request, ctx context.Context) *mdm.Request {
	r2 := r.Clone()
	r2.Context = ctx
	return r2
}

func (ms *MultiService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	err := ms.svcs[0].Authenticate(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			err := svc.Authenticate(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return err
}

func (ms *MultiService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	err := ms.svcs[0].TokenUpdate(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			err := svc.TokenUpdate(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return err
}

func (ms *MultiService) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	err := ms.svcs[0].CheckOut(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			err := svc.CheckOut(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return err
}

func (ms *MultiService) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	respBytes, err := ms.svcs[0].UserAuthenticate(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			_, err := svc.UserAuthenticate(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return respBytes, err
}

func (ms *MultiService) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	err := ms.svcs[0].SetBootstrapToken(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			err := svc.SetBootstrapToken(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return err
}

func (ms *MultiService) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	bsToken, err := ms.svcs[0].GetBootstrapToken(r, m)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			_, err := svc.GetBootstrapToken(rc, m)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return bsToken, err
}

func (ms *MultiService) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	cmd, err := ms.svcs[0].CommandAndReportResults(r, results)
	rc := RequestWithContext(r, context.Background())
	for i, svc := range ms.svcs[1:] {
		go func(n int, svc service.CheckinAndCommandService) {
			_, err := svc.CommandAndReportResults(rc, results)
			if err != nil {
				ms.logger.Info("msg", "multi service", "service", n, "err", err)
			}
		}(i+1, svc)
	}
	return cmd, err
}
