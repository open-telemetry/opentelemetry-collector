// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"go.opentelemetry.io/collector/pdata/plog"
)

// BatchFromLogs transforms a Logs object into a Batch.
type BatchFromLogs func(td plog.Logs) (Batch, error)

func (f BatchFromLogs) BatchFromLogs(td plog.Logs) (Batch, error) {
	return f(td)
}

// BatchFromResourceLogs transforms a ResourceLogs object into a Batch.
type BatchFromResourceLogs func(rl plog.ResourceLogs) (Batch, error)

func (f BatchFromResourceLogs) BatchFromResourceLogs(rl plog.ResourceLogs) (Batch, error) {
	return f(rl)
}

// BatchFromScopeLogs transforms a ResourceLogs object into a Batch.
type BatchFromScopeLogs func(rl plog.ResourceLogs, scopeIdx int) (Batch, error)

func (f BatchFromScopeLogs) BatchFromScopeLogs(rl plog.ResourceLogs, scopeIdx int) (Batch, error) {
	return f(rl, scopeIdx)
}

// BatchFromLogRecord transforms a Span into a Batch.
type BatchFromLogRecord func(rl plog.ResourceLogs, scopeIdx, spanIdx int) (Batch, error)

func (f BatchFromLogRecord) BatchFromLogRecord(rl plog.ResourceLogs, scopeIdx, spanIdx int) (Batch, error) {
	return f(rl, scopeIdx, spanIdx)
}

// IdentifyLogsBatch returns an identifier for a Logs object. This function can be used to separate Logs into
// different batches. Batches with different identifiers will not be merged together.
// This function is optional. If not provided, all Logs will be batched together.
type IdentifyLogsBatch func(ctx context.Context, td plog.Logs) string

func (f IdentifyLogsBatch) IdentifyLogsBatch(ctx context.Context, td plog.Logs) string {
	if f == nil {
		return ""
	}
	return f(ctx, td)
}

type LogsBatchFactory struct {
	BatchFromLogs
	BatchFromResourceLogs
	BatchFromScopeLogs
	BatchFromLogRecord
	IdentifyLogsBatch
}

type LogsBatchFactoryOption func(*LogsBatchFactory)

func WithBatchFromResourceLogs(bf BatchFromResourceLogs) LogsBatchFactoryOption {
	return func(tbf *LogsBatchFactory) {
		tbf.BatchFromResourceLogs = bf
	}
}

func WithBatchFromScopeLogs(bf BatchFromScopeLogs) LogsBatchFactoryOption {
	return func(tbf *LogsBatchFactory) {
		tbf.BatchFromScopeLogs = bf
	}
}

func WithBatchFromLogRecord(bf BatchFromLogRecord) LogsBatchFactoryOption {
	return func(tbf *LogsBatchFactory) {
		tbf.BatchFromLogRecord = bf
	}
}

func WithIdentifyLogsBatch(id IdentifyLogsBatch) LogsBatchFactoryOption {
	return func(tbf *LogsBatchFactory) {
		tbf.IdentifyLogsBatch = id
	}
}

func NewLogsBatchFactory(bft BatchFromLogs, options ...LogsBatchFactoryOption) LogsBatchFactory {
	tbf := LogsBatchFactory{BatchFromLogs: bft}
	for _, option := range options {
		option(&tbf)
	}
	return tbf
}
