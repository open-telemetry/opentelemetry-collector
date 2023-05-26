// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package exporterbatcher

import (
	"context"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// BatchFromMetrics transforms a Metrics object into a Batch.
type BatchFromMetrics func(td pmetric.Metrics) (Batch, error)

func (f BatchFromMetrics) BatchFromMetrics(td pmetric.Metrics) (Batch, error) {
	return f(td)
}

// BatchFromResourceMetrics transforms a ResourceMetrics object into a Batch.
type BatchFromResourceMetrics func(rm pmetric.ResourceMetrics) (Batch, error)

func (f BatchFromResourceMetrics) BatchFromResourceMetrics(rm pmetric.ResourceMetrics) (Batch, error) {
	return f(rm)
}

// BatchFromScopeMetrics transforms a ResourceMetrics object into a Batch.
type BatchFromScopeMetrics func(rm pmetric.ResourceMetrics, scopeIdx int) (Batch, error)

func (f BatchFromScopeMetrics) BatchFromScopeMetrics(rm pmetric.ResourceMetrics, scopeIdx int) (Batch, error) {
	return f(rm, scopeIdx)
}

// BatchFromMetric transforms a Span into a Batch.
type BatchFromMetric func(rm pmetric.ResourceMetrics, scopeIdx, spanIdx int) (Batch, error)

func (f BatchFromMetric) BatchFromMetric(rm pmetric.ResourceMetrics, scopeIdx, spanIdx int) (Batch, error) {
	return f(rm, scopeIdx, spanIdx)
}

// IdentifyMetricsBatch returns an identifier for a Metrics object. This function can be used to separate Metrics into
// different batches. Batches with different identifiers will not be merged together.
// This function is optional. If not provided, all Metrics will be batched together.
type IdentifyMetricsBatch func(ctx context.Context, td pmetric.Metrics) string

func (f IdentifyMetricsBatch) IdentifyMetricsBatch(ctx context.Context, td pmetric.Metrics) string {
	if f == nil {
		return ""
	}
	return f(ctx, td)
}

type MetricsBatchFactory struct {
	BatchFromMetrics
	BatchFromResourceMetrics
	BatchFromScopeMetrics
	BatchFromMetric
	IdentifyMetricsBatch
}

type MetricsBatchFactoryOption func(*MetricsBatchFactory)

func WithBatchFromResourceMetrics(bf BatchFromResourceMetrics) MetricsBatchFactoryOption {
	return func(tbf *MetricsBatchFactory) {
		tbf.BatchFromResourceMetrics = bf
	}
}

func WithBatchFromScopeMetrics(bf BatchFromScopeMetrics) MetricsBatchFactoryOption {
	return func(tbf *MetricsBatchFactory) {
		tbf.BatchFromScopeMetrics = bf
	}
}

func WithBatchFromMetric(bf BatchFromMetric) MetricsBatchFactoryOption {
	return func(tbf *MetricsBatchFactory) {
		tbf.BatchFromMetric = bf
	}
}

func WithIdentifyMetricsBatch(id IdentifyMetricsBatch) MetricsBatchFactoryOption {
	return func(tbf *MetricsBatchFactory) {
		tbf.IdentifyMetricsBatch = id
	}
}

func NewMetricsBatchFactory(bft BatchFromMetrics, options ...MetricsBatchFactoryOption) MetricsBatchFactory {
	tbf := MetricsBatchFactory{BatchFromMetrics: bft}
	for _, option := range options {
		option(&tbf)
	}
	return tbf
}
