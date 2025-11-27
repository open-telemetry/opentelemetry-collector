// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package processorhelper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testLogsCfg = struct{}{}

func TestNewLogs(t *testing.T) {
	lp, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil))
	require.NoError(t, err)

	assert.True(t, lp.Capabilities().MutatesData)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
	assert.NoError(t, lp.Shutdown(context.Background()))
}

func TestNewLogs_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	require.NoError(t, err)

	assert.Equal(t, want, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, lp.Shutdown(context.Background()))
	assert.False(t, lp.Capabilities().MutatesData)
}

func TestNewLogs_NilRequiredFields(t *testing.T) {
	_, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewLogs_ProcessLogError(t *testing.T) {
	want := errors.New("my_error")
	lp, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
}

func TestNewLogs_ProcessLogsErrSkipProcessingData(t *testing.T) {
	lp, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), newTestLProcessor(ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, lp.ConsumeLogs(context.Background(), plog.NewLogs()))
}

func newTestLProcessor(retError error) ProcessLogsFunc {
	return func(_ context.Context, ld plog.Logs) (plog.Logs, error) {
		return ld, retError
	}
}

func TestLogsConcurrency(t *testing.T) {
	logsFunc := func(_ context.Context, ld plog.Logs) (plog.Logs, error) {
		return ld, nil
	}

	incomingLogs := plog.NewLogs()
	incomingLogRecords := incomingLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()

	// Add 3 records to the incoming
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()

	lp, err := NewLogs(context.Background(), processortest.NewNopSettings(processortest.NopType), &testLogsCfg, consumertest.NewNop(), logsFunc)
	require.NoError(t, err)
	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10000 {
				assert.NoError(t, lp.ConsumeLogs(context.Background(), incomingLogs))
			}
		}()
	}
	wg.Wait()
	assert.NoError(t, lp.Shutdown(context.Background()))
}

func TestLogs_RecordInOut(t *testing.T) {
	// Regardless of how many logs are ingested, emit just one
	mockAggregate := func(_ context.Context, _ plog.Logs) (plog.Logs, error) {
		ld := plog.NewLogs()
		ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		return ld, nil
	}

	incomingLogs := plog.NewLogs()
	incomingLogRecords := incomingLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()

	// Add 3 records to the incoming
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()

	tel := componenttest.NewTelemetry()
	lp, err := NewLogs(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), incomingLogs))
	assert.NoError(t, lp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      3,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "logs")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      1,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "logs")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestLogs_RecordIn_ErrorOut(t *testing.T) {
	// Regardless of input, return error
	mockErr := func(_ context.Context, _ plog.Logs) (plog.Logs, error) {
		return plog.NewLogs(), errors.New("fake")
	}

	incomingLogs := plog.NewLogs()
	incomingLogRecords := incomingLogs.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords()

	// Add 3 records to the incoming
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()
	incomingLogRecords.AppendEmpty()

	tel := componenttest.NewTelemetry()
	lp, err := NewLogs(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockErr)
	require.NoError(t, err)

	require.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	require.Error(t, lp.ConsumeLogs(context.Background(), incomingLogs))
	require.NoError(t, lp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      3,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "logs")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      0,
				Attributes: attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "logs")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestLogs_ProcessInternalDuration(t *testing.T) {
	mockAggregate := func(_ context.Context, _ plog.Logs) (plog.Logs, error) {
		ld := plog.NewLogs()
		ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		return ld, nil
	}

	incomingLogs := plog.NewLogs()

	tel := componenttest.NewTelemetry()
	lp, err := NewLogs(context.Background(), newSettings(tel), &testLogsCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, lp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, lp.ConsumeLogs(context.Background(), incomingLogs))
	assert.NoError(t, lp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorInternalDuration(t, tel,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Count:        1,
				BucketCounts: []uint64{1},
				Attributes:   attribute.NewSet(attribute.String("processor", "processorhelper"), attribute.String("otel.signal", "logs")),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func newSettings(tel *componenttest.Telemetry) processor.Settings {
	set := processortest.NewNopSettings(component.MustNewType("processorhelper"))
	set.TelemetrySettings = tel.NewTelemetrySettings()
	return set
}
