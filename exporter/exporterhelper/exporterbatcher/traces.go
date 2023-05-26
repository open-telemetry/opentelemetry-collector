// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// BatchFromTraces transforms a Traces object into a Batch.
type BatchFromTraces func(td ptrace.Traces) (Batch, error)

func (f BatchFromTraces) BatchFromTraces(td ptrace.Traces) (Batch, error) {
	return f(td)
}

// BatchFromResourceSpans transforms a ResourceSpans object into a Batch.
type BatchFromResourceSpans func(rs ptrace.ResourceSpans) (Batch, error)

func (f BatchFromResourceSpans) BatchFromResourceSpans(rs ptrace.ResourceSpans) (Batch, error) {
	return f(rs)
}

// BatchFromScopeSpans transforms a ResourceSpans object into a Batch.
type BatchFromScopeSpans func(rs ptrace.ResourceSpans, scopeIdx int) (Batch, error)

func (f BatchFromScopeSpans) BatchFromScopeSpans(rs ptrace.ResourceSpans, scopeIdx int) (Batch, error) {
	return f(rs, scopeIdx)
}

// BatchFromSpan transforms a Span into a Batch.
type BatchFromSpan func(rs ptrace.ResourceSpans, scopeIdx, spanIdx int) (Batch, error)

func (f BatchFromSpan) BatchFromSpan(rs ptrace.ResourceSpans, scopeIdx, spanIdx int) (Batch, error) {
	return f(rs, scopeIdx, spanIdx)
}

type BatchFromBytes func([]byte) (Batch, error)

func (f BatchFromBytes) BatchFromBytes(b []byte) (Batch, error) {
	return f(b)
}

// IdentifyTracesBatch returns an identifier for a Traces object. This function can be used to separate Traces into
// different batches. Batches with different identifiers will not be merged together.
// This function is optional. If not provided, all Traces will be batched together.
type IdentifyTracesBatch func(ctx context.Context, td ptrace.Traces) string

func (f IdentifyTracesBatch) IdentifyTracesBatch(ctx context.Context, td ptrace.Traces) string {
	if f == nil {
		return ""
	}
	return f(ctx, td)
}

// TracesBatchFactory is a factory for creating Batches from Traces.
// Must be used with NewTracesBatcher. Otherwise, it will panic.
type TracesBatchFactory struct {
	BatchFromTraces
	BatchFromResourceSpans
	BatchFromScopeSpans
	BatchFromSpan
	BatchFromBytes
	IdentifyTracesBatch
}

func NewTracesBatchFactory(bft BatchFromTraces, options ...TracesBatchFactoryOption) TracesBatchFactory {
	tbf := TracesBatchFactory{BatchFromTraces: bft}
	for _, option := range options {
		option(&tbf)
	}
	return tbf
}

type TracesBatchFactoryOption func(*TracesBatchFactory)

func WithBatchFromResourceSpans(bf BatchFromResourceSpans) TracesBatchFactoryOption {
	return func(tbf *TracesBatchFactory) {
		tbf.BatchFromResourceSpans = bf
	}
}

func WithBatchFromScopeSpans(bf BatchFromScopeSpans) TracesBatchFactoryOption {
	return func(tbf *TracesBatchFactory) {
		tbf.BatchFromScopeSpans = bf
	}
}

func WithBatchFromSpan(bf BatchFromSpan) TracesBatchFactoryOption {
	return func(tbf *TracesBatchFactory) {
		tbf.BatchFromSpan = bf
	}
}

func WithBatchFromBytes(bf BatchFromBytes) TracesBatchFactoryOption {
	return func(tbf *TracesBatchFactory) {
		tbf.BatchFromBytes = bf
	}
}

func WithIdentifyTracesBatch(id IdentifyTracesBatch) TracesBatchFactoryOption {
	return func(tbf *TracesBatchFactory) {
		tbf.IdentifyTracesBatch = id
	}
}
