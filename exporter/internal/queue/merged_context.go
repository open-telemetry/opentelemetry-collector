// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queue // import "go.opentelemetry.io/collector/exporter/internal/queue"
import (
	"context"
	"time"

	"go.uber.org/multierr"
)

type mergedContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	ctxArr []context.Context
}

func NewMergedContext(ctx context.Context) mergedContext {
	cancelCtx, cancel := context.WithCancel(ctx)
	return mergedContext{
		ctx:    cancelCtx,
		cancel: cancel,
		ctxArr: []context.Context{ctx},
	}
}

func (mc *mergedContext) Merge(other context.Context) {
	mc.ctxArr = append(mc.ctxArr, other)
	context.AfterFunc(other, func() {
		mc.cancel()
	})
}

func (mc mergedContext) Deadline() (time.Time, bool) {
	var mergedDeadline time.Time
	var mergedOk bool
	for _, c := range mc.ctxArr {
		deadline, ok := c.Deadline()
		if ok {
			if !mergedOk {
				mergedDeadline = deadline
				mergedOk = true
			} else if mergedDeadline.After(deadline) {
				mergedDeadline = deadline
			}
		}
	}
	return mergedDeadline, mergedOk
}

func (mc mergedContext) Done() <-chan struct{} {
	return mc.ctx.Done()
}

func (mc mergedContext) Err() error {
	var mergedErr error
	for _, c := range mc.ctxArr {
		if err := c.Err(); err != nil {
			mergedErr = multierr.Append(mergedErr, c.Err())
		}
	}
	return mergedErr
}

func (mc mergedContext) Value(key any) any {
	for _, c := range mc.ctxArr {
		if value := c.Value(key); value != nil {
			return value
		}
	}
	return nil
}
