package multi

import (
	"context"
	"time"
)

// ContextWithoutCancel returns a derived context that points to the parent context
// and is not canceled when parent is canceled.
// The returned context returns no Deadline or Err, and its Done channel is nil.
// Calling Cause on the returned context returns nil.
func ContextWithoutCancel(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return withoutCancelCtx{parent}
}

type withoutCancelCtx struct {
	c context.Context
}

func (withoutCancelCtx) Deadline() (deadline time.Time, ok bool) {
	return
}

func (withoutCancelCtx) Done() <-chan struct{} {
	return nil
}

func (withoutCancelCtx) Err() error {
	return nil
}

func (c withoutCancelCtx) Value(key interface{}) interface{} {
	return c.c.Value(key)
}

// func (c withoutCancelCtx) String() string {
// 	return c.c.String() + ".WithoutCancel"
// }
