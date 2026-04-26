// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package obsconsumer_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type mockLogsConsumer struct {
	err          error
	capabilities consumer.Capabilities
}

func (m *mockLogsConsumer) ConsumeLogs(_ context.Context, _ plog.Logs) error {
	return m.err
}

func (m *mockLogsConsumer) Capabilities() consumer.Capabilities {
	return m.capabilities
}

func TestLogsNopWhenGateDisabled(t *testing.T) {
	setGateForTest(t, false)

	mp := sdkmetric.NewMeterProvider()
	meter := mp.Meter("test")
	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)

	cons := consumertest.NewNop()
	require.Equal(t, cons, obsconsumer.NewLogs(cons, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, BodyBytesProcessedCounter: bodyBytesProcessedCounter, Logger: zap.NewNop()}))
}

func TestLogsItemsOnly(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	sizeCounterDisabled := newDisabledCounter(sizeCounter)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounterDisabled := newDisabledCounter(bodyBytesProcessedCounter)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounterDisabled, BodyBytesProcessedCounter: bodyBytesProcessedCounterDisabled, Logger: logger})

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty()

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 1)

	metric := rm.ScopeMetrics[0].Metrics[0]
	require.Equal(t, "item_counter", metric.Name)

	data := metric.Data.(metricdata.Sum[int64])
	require.Len(t, data.DataPoints, 1)
	require.Equal(t, int64(1), data.DataPoints[0].Value)

	attrs := data.DataPoints[0].Attributes
	require.Equal(t, 1, attrs.Len())
	val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestLogsConsumeSuccess(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, BodyBytesProcessedCounter: bodyBytesProcessedCounter, Logger: logger})

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("hello")

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	var itemMetric, sizeMetric, bodyBytesProcessedMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		case "body_bytes_processed_counter":
			bodyBytesProcessedMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)
	require.NotNil(t, bodyBytesProcessedMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 1, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 1, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	bodyBytesProcessedData := bodyBytesProcessedMetric.Data.(metricdata.Sum[int64])
	require.Len(t, bodyBytesProcessedData.DataPoints, 1)
	require.Equal(t, int64(5), bodyBytesProcessedData.DataPoints[0].Value) // len("hello") == 5

	bodyBytesProcessedAttrs := bodyBytesProcessedData.DataPoints[0].Attributes
	require.Equal(t, 1, bodyBytesProcessedAttrs.Len())
	val, ok = bodyBytesProcessedAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestLogsConsumeFailure(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	downstreamErr := consumererror.NewDownstream(expectedErr)
	mockConsumer := &mockLogsConsumer{err: expectedErr}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, BodyBytesProcessedCounter: bodyBytesProcessedCounter, Logger: logger})

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("hello")

	err = consumer.ConsumeLogs(ctx, ld)
	assert.Equal(t, downstreamErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	var itemMetric, sizeMetric, bodyBytesProcessedMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		case "body_bytes_processed_counter":
			bodyBytesProcessedMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)
	require.NotNil(t, bodyBytesProcessedMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 1, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())

	require.NotNil(t, sizeMetric)
	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 1, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())

	bodyBytesProcessedData := bodyBytesProcessedMetric.Data.(metricdata.Sum[int64])
	require.Len(t, bodyBytesProcessedData.DataPoints, 1)
	require.Equal(t, int64(5), bodyBytesProcessedData.DataPoints[0].Value) // len("hello") == 5

	bodyBytesProcessedAttrs := bodyBytesProcessedData.DataPoints[0].Attributes
	require.Equal(t, 1, bodyBytesProcessedAttrs.Len())
	val, ok = bodyBytesProcessedAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "failure", val.Emit())

	// Check that the logger was called with an error
	require.Len(t, logs.All(), 1)
	assert.Contains(t, logs.All()[0].Message, "Logs pipeline component had an error")
}

func TestLogsWithStaticAttributes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)

	staticAttr := attribute.String("test", "value")
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, BodyBytesProcessedCounter: bodyBytesProcessedCounter, Logger: logger},
		obsconsumer.WithStaticDataPointAttribute(staticAttr))

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("hello")

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	var itemMetric, sizeMetric, bodyBytesProcessedMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		case "body_bytes_processed_counter":
			bodyBytesProcessedMetric = m
		}
	}
	require.NotNil(t, itemMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 1)
	require.Equal(t, int64(1), itemData.DataPoints[0].Value)

	itemAttrs := itemData.DataPoints[0].Attributes
	require.Equal(t, 2, itemAttrs.Len())
	val, ok := itemAttrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = itemAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 1)
	require.Positive(t, sizeData.DataPoints[0].Value)

	sizeAttrs := sizeData.DataPoints[0].Attributes
	require.Equal(t, 2, sizeAttrs.Len())
	val, ok = sizeAttrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = sizeAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	bodyBytesProcessedData := bodyBytesProcessedMetric.Data.(metricdata.Sum[int64])
	require.Len(t, bodyBytesProcessedData.DataPoints, 1)
	require.Equal(t, int64(5), bodyBytesProcessedData.DataPoints[0].Value) // len("hello") == 5

	bodyBytesProcessedAttrs := bodyBytesProcessedData.DataPoints[0].Attributes
	require.Equal(t, 2, bodyBytesProcessedAttrs.Len())
	val, ok = bodyBytesProcessedAttrs.Value(attribute.Key("test"))
	require.True(t, ok)
	require.Equal(t, "value", val.Emit())
	val, ok = bodyBytesProcessedAttrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
	require.True(t, ok)
	require.Equal(t, "success", val.Emit())

	// Check that the logger was not called
	assert.Empty(t, logs.All())
}

func TestLogsMultipleItemsMixedOutcomes(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	expectedErr := errors.New("test error")
	downstreamErr := consumererror.NewDownstream(expectedErr)
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)

	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, BodyBytesProcessedCounter: bodyBytesProcessedCounter, Logger: logger})

	// First batch: 2 successful items with body "ab" (2 bytes each)
	ld1 := plog.NewLogs()
	for range 2 {
		r := ld1.ResourceLogs().AppendEmpty()
		sl := r.ScopeLogs().AppendEmpty()
		sl.LogRecords().AppendEmpty().Body().SetStr("ab")
	}
	err = consumer.ConsumeLogs(ctx, ld1)
	require.NoError(t, err)

	// Second batch: 1 failed item with body "abc" (3 bytes)
	mockConsumer.err = expectedErr
	ld2 := plog.NewLogs()
	r := ld2.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("abc")
	err = consumer.ConsumeLogs(ctx, ld2)
	assert.Equal(t, downstreamErr, err)

	// Third batch: 2 successful items with body "ab" (2 bytes each)
	mockConsumer.err = nil
	ld3 := plog.NewLogs()
	for range 2 {
		r = ld3.ResourceLogs().AppendEmpty()
		sl = r.ScopeLogs().AppendEmpty()
		sl.LogRecords().AppendEmpty().Body().SetStr("ab")
	}
	err = consumer.ConsumeLogs(ctx, ld3)
	require.NoError(t, err)

	// Fourth batch: 1 failed item with body "abc" (3 bytes)
	mockConsumer.err = expectedErr
	ld4 := plog.NewLogs()
	r = ld4.ResourceLogs().AppendEmpty()
	sl = r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("abc")
	err = consumer.ConsumeLogs(ctx, ld4)
	assert.Equal(t, downstreamErr, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 3)

	var itemMetric, sizeMetric, bodyBytesProcessedMetric metricdata.Metrics
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case "item_counter":
			itemMetric = m
		case "size_counter":
			sizeMetric = m
		case "body_bytes_processed_counter":
			bodyBytesProcessedMetric = m
		}
	}
	require.NotNil(t, itemMetric)
	require.NotNil(t, sizeMetric)
	require.NotNil(t, bodyBytesProcessedMetric)

	itemData := itemMetric.Data.(metricdata.Sum[int64])
	require.Len(t, itemData.DataPoints, 2)
	sizeData := sizeMetric.Data.(metricdata.Sum[int64])
	require.Len(t, sizeData.DataPoints, 2)
	bodyBytesProcessedData := bodyBytesProcessedMetric.Data.(metricdata.Sum[int64])
	require.Len(t, bodyBytesProcessedData.DataPoints, 2)

	var successDP, failureDP metricdata.DataPoint[int64]
	for _, dp := range itemData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successDP = dp
		} else {
			failureDP = dp
		}
	}
	require.Equal(t, int64(4), successDP.Value)
	require.Equal(t, int64(2), failureDP.Value)

	var successSizeDP, failureSizeDP metricdata.DataPoint[int64]
	for _, dp := range sizeData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successSizeDP = dp
		} else {
			failureSizeDP = dp
		}
	}
	// Size values change because we now set body strings on log records
	require.Positive(t, successSizeDP.Value)
	require.Positive(t, failureSizeDP.Value)

	var successBodyBytesProcessedDP, failureBodyBytesProcessedDP metricdata.DataPoint[int64]
	for _, dp := range bodyBytesProcessedData.DataPoints {
		val, ok := dp.Attributes.Value(attribute.Key(obsconsumer.ComponentOutcome))
		if ok && val.Emit() == "success" {
			successBodyBytesProcessedDP = dp
		} else {
			failureBodyBytesProcessedDP = dp
		}
	}
	require.Equal(t, int64(8), successBodyBytesProcessedDP.Value) // 2*2 + 2*2 = 8 bytes ("ab" * 4)
	require.Equal(t, int64(6), failureBodyBytesProcessedDP.Value) // 3 + 3 = 6 bytes ("abc" * 2)

	// Check that the logger was called for errors
	require.Len(t, logs.All(), 2)
	for _, log := range logs.All() {
		assert.Contains(t, log.Message, "Logs pipeline component had an error")
	}
}

func TestLogsBodyBytesProcessedDisabled(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	mockConsumer := &mockLogsConsumer{}

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
	require.NoError(t, err)
	bodyBytesProcessedCounterDisabled := newDisabledCounter(bodyBytesProcessedCounter)

	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{
		ItemCounter:               itemCounter,
		SizeCounter:               sizeCounter,
		BodyBytesProcessedCounter: bodyBytesProcessedCounterDisabled,
		Logger:                    zap.NewNop(),
	})

	ld := plog.NewLogs()
	r := ld.ResourceLogs().AppendEmpty()
	sl := r.ScopeLogs().AppendEmpty()
	sl.LogRecords().AppendEmpty().Body().SetStr("hello")

	err = consumer.ConsumeLogs(ctx, ld)
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	require.Len(t, rm.ScopeMetrics[0].Metrics, 2) // only item_counter and size_counter

	for _, m := range rm.ScopeMetrics[0].Metrics {
		require.NotEqual(t, "body_bytes_processed_counter", m.Name)
	}
}

func TestLogsBodyBytesProcessedStructuredBodies(t *testing.T) {
	setGateForTest(t, true)

	type bodySetup func(body pcommon.Value)

	tests := []struct {
		name     string
		setup    bodySetup
		expected int64
	}{
		// Primitive types
		{
			name:     "empty",
			setup:    func(_ pcommon.Value) {},
			expected: 0,
		},
		{
			name:     "string",
			setup:    func(v pcommon.Value) { v.SetStr("hello") },
			expected: 5,
		},
		{
			name:     "empty_string",
			setup:    func(v pcommon.Value) { v.SetStr("") },
			expected: 0,
		},
		{
			name:     "int",
			setup:    func(v pcommon.Value) { v.SetInt(42) },
			expected: 8,
		},
		{
			name:     "double",
			setup:    func(v pcommon.Value) { v.SetDouble(3.14) },
			expected: 8,
		},
		{
			name:     "bool",
			setup:    func(v pcommon.Value) { v.SetBool(true) },
			expected: 1,
		},
		{
			name: "bytes",
			setup: func(v pcommon.Value) {
				v.SetEmptyBytes().FromRaw([]byte{1, 2, 3})
			},
			expected: 3,
		},
		{
			name:     "empty_bytes",
			setup:    func(v pcommon.Value) { v.SetEmptyBytes() },
			expected: 0,
		},

		// Flat map and slice
		{
			name:     "empty_map",
			setup:    func(v pcommon.Value) { v.SetEmptyMap() },
			expected: 0,
		},
		{
			name:     "empty_slice",
			setup:    func(v pcommon.Value) { v.SetEmptySlice() },
			expected: 0,
		},
		{
			name: "flat_map_string_values",
			setup: func(v pcommon.Value) {
				// key "msg" (3) + value "hello world" (11) + key "level" (5) + value "info" (4) = 23
				m := v.SetEmptyMap()
				m.PutStr("msg", "hello world")
				m.PutStr("level", "info")
			},
			expected: 23,
		},
		{
			name: "flat_map_mixed_value_types",
			setup: func(v pcommon.Value) {
				// key "name" (4) + "alice" (5)
				// + key "age" (3) + int (8)
				// + key "active" (6) + bool (1)
				// + key "score" (5) + double (8) = 40
				m := v.SetEmptyMap()
				m.PutStr("name", "alice")
				m.PutInt("age", 30)
				m.PutBool("active", true)
				m.PutDouble("score", 99.5)
			},
			expected: 40,
		},
		{
			name: "flat_slice",
			setup: func(v pcommon.Value) {
				// "a" (1) + "bb" (2) + "ccc" (3) = 6
				s := v.SetEmptySlice()
				s.AppendEmpty().SetStr("a")
				s.AppendEmpty().SetStr("bb")
				s.AppendEmpty().SetStr("ccc")
			},
			expected: 6,
		},
		{
			name: "slice_mixed_types",
			setup: func(v pcommon.Value) {
				// "hello" (5) + int (8) + bool (1) + double (8) = 22
				s := v.SetEmptySlice()
				s.AppendEmpty().SetStr("hello")
				s.AppendEmpty().SetInt(1)
				s.AppendEmpty().SetBool(false)
				s.AppendEmpty().SetDouble(2.0)
			},
			expected: 22,
		},

		// Nested: map containing map
		{
			name: "nested_map_in_map",
			setup: func(v pcommon.Value) {
				// key "outer" (5) + nested map:
				//   key "inner" (5) + "value" (5) = 10
				// total = 5 + 10 = 15
				outer := v.SetEmptyMap()
				inner := outer.PutEmptyMap("outer")
				inner.PutStr("inner", "value")
			},
			expected: 15,
		},
		{
			name: "nested_map_three_levels",
			setup: func(v pcommon.Value) {
				// "a" (1) + map:
				//   "b" (1) + map:
				//     "c" (1) + "deep" (4)
				// total = 1 + 1 + 1 + 4 = 7
				l1 := v.SetEmptyMap()
				l2 := l1.PutEmptyMap("a")
				l3 := l2.PutEmptyMap("b")
				l3.PutStr("c", "deep")
			},
			expected: 7,
		},

		// Nested: map containing slice
		{
			name: "map_containing_slice",
			setup: func(v pcommon.Value) {
				// key "tags" (4) + slice:
				//   "x" (1) + "yy" (2) = 3
				// total = 4 + 3 = 7
				m := v.SetEmptyMap()
				s := m.PutEmptySlice("tags")
				s.AppendEmpty().SetStr("x")
				s.AppendEmpty().SetStr("yy")
			},
			expected: 7,
		},

		// Nested: slice containing map
		{
			name: "slice_containing_map",
			setup: func(v pcommon.Value) {
				// element 0 map: key "k" (1) + "v" (1) = 2
				// element 1 map: key "ab" (2) + int (8) = 10
				// total = 12
				s := v.SetEmptySlice()
				m0 := s.AppendEmpty().SetEmptyMap()
				m0.PutStr("k", "v")
				m1 := s.AppendEmpty().SetEmptyMap()
				m1.PutInt("ab", 7)
			},
			expected: 12,
		},

		// Nested: slice containing slice
		{
			name: "slice_containing_slice",
			setup: func(v pcommon.Value) {
				// inner slice 0: "aa" (2) + "bb" (2) = 4
				// inner slice 1: "c" (1) = 1
				// total = 5
				outer := v.SetEmptySlice()
				inner0 := outer.AppendEmpty().SetEmptySlice()
				inner0.AppendEmpty().SetStr("aa")
				inner0.AppendEmpty().SetStr("bb")
				inner1 := outer.AppendEmpty().SetEmptySlice()
				inner1.AppendEmpty().SetStr("c")
			},
			expected: 5,
		},

		// Deeply nested: 5 levels of maps
		{
			name: "five_level_nested_map",
			setup: func(v pcommon.Value) {
				// "1" (1) -> "2" (1) -> "3" (1) -> "4" (1) -> "5" (1) + "leaf" (4)
				// total = 1 + 1 + 1 + 1 + 1 + 4 = 9
				l1 := v.SetEmptyMap()
				l2 := l1.PutEmptyMap("1")
				l3 := l2.PutEmptyMap("2")
				l4 := l3.PutEmptyMap("3")
				l5 := l4.PutEmptyMap("4")
				l5.PutStr("5", "leaf")
			},
			expected: 9,
		},

		// Deeply nested: alternating maps and slices
		{
			name: "alternating_map_slice_nesting",
			setup: func(v pcommon.Value) {
				// map: key "items" (5) -> slice -> [map: key "id" (2) + int (8)] = 15
				m := v.SetEmptyMap()
				s := m.PutEmptySlice("items")
				inner := s.AppendEmpty().SetEmptyMap()
				inner.PutInt("id", 1)
			},
			expected: 15,
		},

		// Map with empty nested containers
		{
			name: "map_with_empty_nested_containers",
			setup: func(v pcommon.Value) {
				// key "a" (1) + empty map (0) + key "b" (1) + empty slice (0) + key "c" (1) + "val" (3) = 6
				m := v.SetEmptyMap()
				m.PutEmptyMap("a")
				m.PutEmptySlice("b")
				m.PutStr("c", "val")
			},
			expected: 6,
		},

		// Slice with empty nested containers
		{
			name: "slice_with_empty_nested_containers",
			setup: func(v pcommon.Value) {
				// empty map (0) + empty slice (0) + "data" (4) = 4
				s := v.SetEmptySlice()
				s.AppendEmpty().SetEmptyMap()
				s.AppendEmpty().SetEmptySlice()
				s.AppendEmpty().SetStr("data")
			},
			expected: 4,
		},

		// Complex realistic structure: JSON-parsed log body
		{
			name: "realistic_structured_log",
			setup: func(v pcommon.Value) {
				// Simulates a parsed JSON log:
				// {
				//   "message": "request completed",    key(7) + val(17) = 24
				//   "status": 200,                     key(6) + int(8)  = 14
				//   "duration_ms": 42.5,               key(11) + dbl(8) = 19
				//   "success": true,                   key(7) + bool(1) = 8
				//   "headers": {                       key(7)
				//     "content-type": "application/json",  key(12) + val(16) = 28
				//     "x-request-id": "abc123"             key(12) + val(6) = 18
				//   },                                                    = 53
				//   "tags": ["web", "api"]             key(4) + "web"(3) + "api"(3) = 10
				// }
				// total = 24 + 14 + 19 + 8 + 53 + 10 = 128
				m := v.SetEmptyMap()
				m.PutStr("message", "request completed")
				m.PutInt("status", 200)
				m.PutDouble("duration_ms", 42.5)
				m.PutBool("success", true)
				headers := m.PutEmptyMap("headers")
				headers.PutStr("content-type", "application/json")
				headers.PutStr("x-request-id", "abc123")
				tags := m.PutEmptySlice("tags")
				tags.AppendEmpty().SetStr("web")
				tags.AppendEmpty().SetStr("api")
			},
			expected: 128,
		},

		// Map with bytes value
		{
			name: "map_with_bytes_value",
			setup: func(v pcommon.Value) {
				// key "data" (4) + 5 bytes = 9
				m := v.SetEmptyMap()
				m.PutEmptyBytes("data").FromRaw([]byte{0, 1, 2, 3, 4})
			},
			expected: 9,
		},

		// Deeply nested slices (4 levels)
		{
			name: "four_level_nested_slices",
			setup: func(v pcommon.Value) {
				// [[["abc"]]] = "abc" (3)
				l1 := v.SetEmptySlice()
				l2 := l1.AppendEmpty().SetEmptySlice()
				l3 := l2.AppendEmpty().SetEmptySlice()
				l3.AppendEmpty().SetStr("abc")
			},
			expected: 3,
		},

		// Wide and deep: map with multiple nested branches
		{
			name: "wide_and_deep_tree",
			setup: func(v pcommon.Value) {
				// "left" (4) -> "ll" (2) -> "x" (1)        = 4 + 2 + 1 = 7
				// "right" (5) -> "rr" (2) -> int(8)         = 5 + 2 + 8 = 15
				// "mid" (3) -> "mm" (2) -> bool(1)           = 3 + 2 + 1 = 6
				// total = 28
				root := v.SetEmptyMap()
				left := root.PutEmptyMap("left")
				left.PutStr("ll", "x")
				right := root.PutEmptyMap("right")
				right.PutInt("rr", 99)
				mid := root.PutEmptyMap("mid")
				mid.PutBool("mm", true)
			},
			expected: 28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockConsumer := &mockLogsConsumer{}

			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			meter := mp.Meter("test")

			itemCounter, err := meter.Int64Counter("item_counter")
			require.NoError(t, err)
			sizeCounter, err := meter.Int64Counter("size_counter")
			require.NoError(t, err)
			sizeCounterDisabled := newDisabledCounter(sizeCounter)
			bodyBytesProcessedCounter, err := meter.Int64Counter("body_bytes_processed_counter")
			require.NoError(t, err)

			consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{
				ItemCounter:               itemCounter,
				SizeCounter:               sizeCounterDisabled,
				BodyBytesProcessedCounter: bodyBytesProcessedCounter,
				Logger:                    zap.NewNop(),
			})

			ld := plog.NewLogs()
			lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			tt.setup(lr.Body())

			err = consumer.ConsumeLogs(ctx, ld)
			require.NoError(t, err)

			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			require.NoError(t, err)
			require.Len(t, rm.ScopeMetrics, 1)

			for _, m := range rm.ScopeMetrics[0].Metrics {
				if m.Name == "body_bytes_processed_counter" {
					data := m.Data.(metricdata.Sum[int64])
					require.Len(t, data.DataPoints, 1)
					assert.Equal(t, tt.expected, data.DataPoints[0].Value)
					return
				}
			}
			t.Fatal("body_bytes_processed_counter metric not found")
		})
	}
}

func TestLogsCapabilities(t *testing.T) {
	setGateForTest(t, true)

	mockConsumer := &mockLogsConsumer{
		capabilities: consumer.Capabilities{MutatesData: true},
	}
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	itemCounter, err := meter.Int64Counter("item_counter")
	require.NoError(t, err)
	sizeCounter, err := meter.Int64Counter("size_counter")
	require.NoError(t, err)
	sizeCounterDisabled := newDisabledCounter(sizeCounter)

	// Test with item counter only
	consumer := obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounterDisabled, Logger: zap.NewNop()})
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)

	// Test with both counters
	consumer = obsconsumer.NewLogs(mockConsumer, obsconsumer.Settings{ItemCounter: itemCounter, SizeCounter: sizeCounter, Logger: zap.NewNop()})
	require.Equal(t, consumer.Capabilities(), mockConsumer.capabilities)
}
