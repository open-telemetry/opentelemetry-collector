// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package cmetrictest // import "go.opentelemetry.io/collector/consumer/cmetric/cmetrictest"

import (
	"context"
	"sync"

	"go.opentelemetry.io/collector/consumer/cmetric"
	"go.opentelemetry.io/collector/consumer/internal/consumertest"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// MetricsSink is a cmetric.Metrics that acts like a sink that
// stores all metrics and allows querying them for testing.
type MetricsSink struct {
	consumertest.NonMutatingConsumer
	mu             sync.Mutex
	metrics        []pmetric.Metrics
	dataPointCount int
}

var _ cmetric.Metrics = (*MetricsSink)(nil)

// ConsumeMetrics stores metrics to this sink.
func (sme *MetricsSink) ConsumeMetrics(_ context.Context, md pmetric.Metrics) error {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = append(sme.metrics, md)
	sme.dataPointCount += md.DataPointCount()

	return nil
}

// AllMetrics returns the metrics stored by this sink since last Reset.
func (sme *MetricsSink) AllMetrics() []pmetric.Metrics {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	copyMetrics := make([]pmetric.Metrics, len(sme.metrics))
	copy(copyMetrics, sme.metrics)
	return copyMetrics
}

// DataPointCount returns the number of metrics stored by this sink since last Reset.
func (sme *MetricsSink) DataPointCount() int {
	sme.mu.Lock()
	defer sme.mu.Unlock()
	return sme.dataPointCount
}

// Reset deletes any stored data.
func (sme *MetricsSink) Reset() {
	sme.mu.Lock()
	defer sme.mu.Unlock()

	sme.metrics = nil
	sme.dataPointCount = 0
}
