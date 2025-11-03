// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"sync/atomic"

	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*collectorCore)(nil)

type collectorCore struct {
	delegate atomic.Pointer[zapcore.Core]
}

func newCollectorCore(core zapcore.Core) *collectorCore {
	cc := &collectorCore{}
	cc.SetCore(core)
	return cc
}

func (c *collectorCore) Enabled(l zapcore.Level) bool {
	return c.loadDelegate().Enabled(l)
}

func (c *collectorCore) With(f []zapcore.Field) zapcore.Core {
	return newCollectorCore(c.loadDelegate().With(f))
}

func (c *collectorCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	core := c.loadDelegate()
	if core.Enabled(e.Level) {
		return ce.AddCore(e, core)
	}
	return ce
}

func (c *collectorCore) Write(e zapcore.Entry, f []zapcore.Field) error {
	return c.loadDelegate().Write(e, f)
}

func (c *collectorCore) Sync() error {
	return c.loadDelegate().Sync()
}

func (c *collectorCore) SetCore(core zapcore.Core) {
	c.delegate.Store(&core)
}

// loadDelegate returns the delegate.
func (c *collectorCore) loadDelegate() zapcore.Core {
	return *c.delegate.Load()
}
