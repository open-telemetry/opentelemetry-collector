// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processortest // import "go.opentelemetry.io/collector/processor/processortest"

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
)

func verifyTracesDoesNotProduceAfterShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	// Create a proc and output its produce to a sink.
	nextSink := new(consumertest.TracesSink)
	proc, err := factory.CreateTraces(context.Background(), NewNopSettings(factory.Type()), cfg, nextSink)
	if errors.Is(err, pipeline.ErrSignalNotSupported) {
		return
	}
	require.NoError(t, err)
	assert.NoError(t, proc.Start(context.Background(), componenttest.NewNopHost()))

	// Send some traces to the proc.
	const generatedCount = 10
	for range generatedCount {
		require.NoError(t, proc.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))
	}

	// Now shutdown the proc.
	assert.NoError(t, proc.Shutdown(context.Background()))

	// The Shutdown() is done. It means the proc must have sent everything we
	// gave it to the next sink.
	assert.Equal(t, generatedCount, nextSink.SpanCount())
}

func verifyLogsDoesNotProduceAfterShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	// Create a proc and output its produce to a sink.
	nextSink := new(consumertest.LogsSink)
	proc, err := factory.CreateLogs(context.Background(), NewNopSettings(factory.Type()), cfg, nextSink)
	if errors.Is(err, pipeline.ErrSignalNotSupported) {
		return
	}
	require.NoError(t, err)
	assert.NoError(t, proc.Start(context.Background(), componenttest.NewNopHost()))

	// Send some logs to the proc.
	const generatedCount = 10
	for range generatedCount {
		require.NoError(t, proc.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
	}

	// Now shutdown the proc.
	assert.NoError(t, proc.Shutdown(context.Background()))

	// The Shutdown() is done. It means the proc must have sent everything we
	// gave it to the next sink.
	assert.Equal(t, generatedCount, nextSink.LogRecordCount())
}

func verifyMetricsDoesNotProduceAfterShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	// Create a proc and output its produce to a sink.
	nextSink := new(consumertest.MetricsSink)
	proc, err := factory.CreateMetrics(context.Background(), NewNopSettings(factory.Type()), cfg, nextSink)
	if errors.Is(err, pipeline.ErrSignalNotSupported) {
		return
	}
	require.NoError(t, err)
	assert.NoError(t, proc.Start(context.Background(), componenttest.NewNopHost()))

	// Send some metrics to the proc. testdata.GenerateMetrics creates metrics with 2 data points each.
	const generatedCount = 10
	for range generatedCount {
		require.NoError(t, proc.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))
	}

	// Now shutdown the proc.
	assert.NoError(t, proc.Shutdown(context.Background()))

	// The Shutdown() is done. It means the proc must have sent everything we
	// gave it to the next sink.
	assert.Equal(t, generatedCount*2, nextSink.DataPointCount())
}

// VerifyShutdown verifies the processor doesn't produce telemetry data after shutdown.
func VerifyShutdown(t *testing.T, factory processor.Factory, cfg component.Config) {
	verifyTracesDoesNotProduceAfterShutdown(t, factory, cfg)
	verifyLogsDoesNotProduceAfterShutdown(t, factory, cfg)
	verifyMetricsDoesNotProduceAfterShutdown(t, factory, cfg)
}
