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
	mu   sync.Mutex
}

func (c *collectorCore) Enabled(l zapcore.Level) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.core.Enabled(l)
}

func (c *collectorCore) With(f []zapcore.Field) zapcore.Core {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.core.With(f)
}

func (c *collectorCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.core.Check(e, ce)
}

func (c *collectorCore) Write(e zapcore.Entry, f []zapcore.Field) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.core.Write(e, f)
}

func (c *collectorCore) Sync() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.core.Sync()
}

func (c *collectorCore) SetCore(core zapcore.Core) {
	c.mu.Lock()
	c.core = core
	c.mu.Unlock()
}
