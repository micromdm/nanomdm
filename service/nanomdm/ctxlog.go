package nanomdm

import (
	"context"

	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/ctxlog"
	"github.com/micromdm/nanomdm/mdm"
)

type (
	ctxKeyID   struct{}
	ctxKeyType struct{}
)

func newContext(ctx context.Context, r *mdm.Request) context.Context {
	newCtx := context.WithValue(ctx, ctxKeyID{}, r.ID)
	return context.WithValue(newCtx, ctxKeyType{}, r.Type)
}

func ctxKVs(ctx context.Context) (out []interface{}) {
	id, ok := ctx.Value(ctxKeyID{}).(string)
	if ok {
		out = append(out, "id", id)
	}
	eType, ok := ctx.Value(ctxKeyType{}).(mdm.EnrollType)
	if ok {
		out = append(out, "type", eType)
	}
	return
}

// ctxLogger sets up and returns a new contextual logger
func (s *Service) ctxLogger(r *mdm.Request) log.Logger {
	r.Context = newContext(r.Context, r)
	r.Context = ctxlog.AddFunc(r.Context, ctxKVs)
	return ctxlog.Logger(r.Context, s.logger)
}
