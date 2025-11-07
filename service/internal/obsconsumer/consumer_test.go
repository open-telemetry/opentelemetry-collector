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

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/service/internal/obsconsumer"
)

type failingConsumer struct {
	err error
}

var (
	_ consumer.Metrics   = (*failingConsumer)(nil)
	_ consumer.Logs      = (*failingConsumer)(nil)
	_ consumer.Traces    = (*failingConsumer)(nil)
	_ xconsumer.Profiles = (*failingConsumer)(nil)
)

func (*failingConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (fc *failingConsumer) ConsumeMetrics(_ context.Context, _ pmetric.Metrics) error {
	return fc.err
}

func (fc *failingConsumer) ConsumeLogs(_ context.Context, _ plog.Logs) error {
	return fc.err
}

func (fc *failingConsumer) ConsumeTraces(_ context.Context, _ ptrace.Traces) error {
	return fc.err
}

func (fc *failingConsumer) ConsumeProfiles(_ context.Context, _ pprofile.Profiles) error {
	return fc.err
}

func TestConsumeRefused(t *testing.T) {
	setGateForTest(t, true)

	ctx := context.Background()
	originalErr := errors.New("test error")
	expectedErr := consumererror.NewDownstream(originalErr)
	mockConsumer := &failingConsumer{err: originalErr}

	// Use delta temporality so sums don't accumulate across tests
	reader := sdkmetric.NewManualReader(sdkmetric.WithTemporalitySelector(func(_ sdkmetric.InstrumentKind) metricdata.Temporality {
		return metricdata.DeltaTemporality
	}))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")

	receivedItemsCounter, err := meter.Int64Counter("received.items")
	require.NoError(t, err)
	receivedSizeCounter, err := meter.Int64Counter("received.size")
	require.NoError(t, err)

	producedItemsCounter, err := meter.Int64Counter("produced.items")
	require.NoError(t, err)
	producedSizeConter, err := meter.Int64Counter("produced.size")
	require.NoError(t, err)

	logger := zap.NewNop()
	receivedSettings := obsconsumer.Settings{ItemCounter: receivedItemsCounter, SizeCounter: receivedSizeCounter, Logger: logger}
	producedSettings := obsconsumer.Settings{ItemCounter: producedItemsCounter, SizeCounter: producedSizeConter, Logger: logger}

	type testCase struct {
		name         string
		testConsumer func() error
	}

	testCases := []testCase{
		{
			name: "metrics",
			testConsumer: func() error {
				consumer1 := obsconsumer.NewMetrics(mockConsumer, receivedSettings)
				consumer2 := obsconsumer.NewMetrics(consumer1, producedSettings)
				md := pmetric.NewMetrics()
				md.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty().Metrics().AppendEmpty().SetEmptyGauge().DataPoints().AppendEmpty()
				return consumer2.ConsumeMetrics(ctx, md)
			},
		},
		{
			name: "logs",
			testConsumer: func() error {
				consumer1 := obsconsumer.NewLogs(mockConsumer, receivedSettings)
				consumer2 := obsconsumer.NewLogs(consumer1, producedSettings)
				ld := plog.NewLogs()
				ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
				return consumer2.ConsumeLogs(ctx, ld)
			},
		},
		{
			name: "traces",
			testConsumer: func() error {
				consumer1 := obsconsumer.NewTraces(mockConsumer, receivedSettings)
				consumer2 := obsconsumer.NewTraces(consumer1, producedSettings)
				td := ptrace.NewTraces()
				td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
				return consumer2.ConsumeTraces(ctx, td)
			},
		},
		{
			name: "profiles",
			testConsumer: func() error {
				consumer1 := obsconsumer.NewProfiles(mockConsumer, receivedSettings)
				consumer2 := obsconsumer.NewProfiles(consumer1, producedSettings)
				pd := pprofile.NewProfiles()
				pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()
				return consumer2.ConsumeProfiles(ctx, pd)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.testConsumer()
			assert.Equal(t, expectedErr, err)

			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			require.NoError(t, err)
			require.Len(t, rm.ScopeMetrics, 1)
			require.Len(t, rm.ScopeMetrics[0].Metrics, 4)

			var receivedItemMetric, receivedSizeMetric metricdata.Metrics
			var producedItemMetric, producedSizeMetric metricdata.Metrics
			for _, m := range rm.ScopeMetrics[0].Metrics {
				switch m.Name {
				case "received.items":
					receivedItemMetric = m
				case "received.size":
					receivedSizeMetric = m
				case "produced.items":
					producedItemMetric = m
				case "produced.size":
					producedSizeMetric = m
				}
			}
			require.NotNil(t, receivedItemMetric)
			require.NotNil(t, receivedSizeMetric)
			require.NotNil(t, producedItemMetric)
			require.NotNil(t, producedSizeMetric)

			data := receivedItemMetric.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 1)
			require.Equal(t, int64(1), data.DataPoints[0].Value)
			attrs := data.DataPoints[0].Attributes
			require.Equal(t, 1, attrs.Len())
			val, ok := attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
			require.True(t, ok)
			require.Equal(t, "failure", val.Emit())

			data = receivedSizeMetric.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 1)
			require.Positive(t, data.DataPoints[0].Value)
			attrs = data.DataPoints[0].Attributes
			require.Equal(t, 1, attrs.Len())
			val, ok = attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
			require.True(t, ok)
			require.Equal(t, "failure", val.Emit())

			data = producedItemMetric.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 1)
			require.Equal(t, int64(1), data.DataPoints[0].Value)
			attrs = data.DataPoints[0].Attributes
			require.Equal(t, 1, attrs.Len())
			val, ok = attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
			require.True(t, ok)
			require.Equal(t, "refused", val.Emit())

			data = producedSizeMetric.Data.(metricdata.Sum[int64])
			require.Len(t, data.DataPoints, 1)
			require.Positive(t, data.DataPoints[0].Value)
			attrs = data.DataPoints[0].Attributes
			require.Equal(t, 1, attrs.Len())
			val, ok = attrs.Value(attribute.Key(obsconsumer.ComponentOutcome))
			require.True(t, ok)
			require.Equal(t, "refused", val.Emit())
		})
	}
}
