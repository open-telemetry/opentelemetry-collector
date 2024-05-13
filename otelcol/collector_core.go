// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

var _ zapcore.Core = (*collectorCore)(nil)

type collectorCore struct {
	core zapcore.Core
	rw   sync.RWMutex
}

func (c *collectorCore) Enabled(l zapcore.Level) bool {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.core.Enabled(l)
}

func (c *collectorCore) With(f []zapcore.Field) zapcore.Core {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return &collectorCore{
		core: c.core.With(f),
	}
}

func (c *collectorCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	c.rw.RLock()
	defer c.rw.RUnlock()
	if c.core.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}

func (c *collectorCore) Write(e zapcore.Entry, f []zapcore.Field) error {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.core.Write(e, f)
}

func (c *collectorCore) Sync() error {
	c.rw.RLock()
	defer c.rw.RUnlock()
	return c.core.Sync()
}

func (c *collectorCore) SetCore(core zapcore.Core) {
	c.rw.Lock()
	defer c.rw.Unlock()
	c.core = core
}
