// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// This logger implements zapcore.Core and is based on zaptest/observer.
package otelcol // import "go.opentelemetry.io/collector/otelcol"

import (
	"sync"

	"go.uber.org/zap/zapcore"
)

type loggedEntry struct {
	zapcore.Entry
	Context []zapcore.Field
}

func newBufferedCore(enab zapcore.LevelEnabler) *bufferedCore {
	return &bufferedCore{LevelEnabler: enab}
}

var _ zapcore.Core = (*bufferedCore)(nil)

type bufferedCore struct {
	zapcore.LevelEnabler
	mu      sync.RWMutex
	logs    []loggedEntry
	context []zapcore.Field
}

func (bc *bufferedCore) Level() zapcore.Level {
	return zapcore.LevelOf(bc.LevelEnabler)
}

func (bc *bufferedCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if bc.Enabled(ent.Level) {
		return ce.AddCore(ent, bc)
	}
	return ce
}

func (bc *bufferedCore) With(fields []zapcore.Field) zapcore.Core {
	return &bufferedCore{
		LevelEnabler: bc.LevelEnabler,
		logs:         bc.logs,
		context:      append(bc.context[:len(bc.context):len(bc.context)], fields...),
	}
}

func (bc *bufferedCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	all := make([]zapcore.Field, 0, len(fields)+len(bc.context))
	all = append(all, bc.context...)
	all = append(all, fields...)
	bc.mu.Lock()
	bc.logs = append(bc.logs, loggedEntry{ent, all})
	bc.mu.Unlock()
	return nil
}

func (bc *bufferedCore) Sync() error {
	return nil
}

func (bc *bufferedCore) TakeLogs() []loggedEntry {
	bc.mu.Lock()
	logs := bc.logs
	bc.logs = nil
	bc.mu.Unlock()
	return logs
}
