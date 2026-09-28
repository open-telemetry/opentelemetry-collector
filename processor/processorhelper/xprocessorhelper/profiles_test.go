// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xprocessorhelper

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processorhelper/internal/metadatatest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func newProfilesSettings(tel *componenttest.Telemetry) processor.Settings {
	set := processortest.NewNopSettings(processortest.NopType)
	set.TelemetrySettings = tel.NewTelemetrySettings()
	return set
}

var testProfilesCfg = struct{}{}

func TestNewProfiles(t *testing.T) {
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil))
	require.NoError(t, err)

	assert.True(t, pp.Capabilities().MutatesData)
	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
	assert.NoError(t, pp.Shutdown(context.Background()))
}

func TestNewProfiles_WithOptions(t *testing.T) {
	want := errors.New("my_error")
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil),
		WithStart(func(context.Context, component.Host) error { return want }),
		WithShutdown(func(context.Context) error { return want }),
		WithCapabilities(consumer.Capabilities{MutatesData: false}))
	require.NoError(t, err)

	assert.Equal(t, want, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.Equal(t, want, pp.Shutdown(context.Background()))
	assert.False(t, pp.Capabilities().MutatesData)
}

func TestNewProfiles_NilRequiredFields(t *testing.T) {
	_, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), nil)
	assert.Error(t, err)
}

func TestNewProfiles_ProcessProfileError(t *testing.T) {
	want := errors.New("my_error")
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(want))
	require.NoError(t, err)
	assert.Equal(t, want, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func TestNewProfiles_ProcessProfilesErrSkipProcessingData(t *testing.T) {
	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(processorhelper.ErrSkipProcessingData))
	require.NoError(t, err)
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), pprofile.NewProfiles()))
}

func newTestPProcessor(retError error) ProcessProfilesFunc {
	return func(_ context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
		return pd, retError
	}
}

func TestProfilesConcurrency(t *testing.T) {
	profilesFunc := func(_ context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
		return pd, nil
	}

	incomingProfiles := pprofile.NewProfiles()
	ps := incomingProfiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles()

	// Add 3 profiles to the incoming
	ps.AppendEmpty()
	ps.AppendEmpty()
	ps.AppendEmpty()

	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), profilesFunc)
	require.NoError(t, err)
	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			for range 10000 {
				assert.NoError(t, pp.ConsumeProfiles(context.Background(), incomingProfiles))
			}
		})
	}
	wg.Wait()
	assert.NoError(t, pp.Shutdown(context.Background()))
}

func TestProfiles_RecordInOut(t *testing.T) {
	// Regardless of how many samples are ingested, emit just one
	mockAggregate := func(_ context.Context, _ pprofile.Profiles) (pprofile.Profiles, error) {
		pd := pprofile.NewProfiles()
		pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()
		return pd, nil
	}

	incomingProfiles := pprofile.NewProfiles()
	samples := incomingProfiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples()

	// Add 4 samples to the incoming
	samples.AppendEmpty()
	samples.AppendEmpty()
	samples.AppendEmpty()
	samples.AppendEmpty()

	tel := componenttest.NewTelemetry()
	pp, err := NewProfiles(context.Background(), newProfilesSettings(tel), &testProfilesCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), incomingProfiles))
	assert.NoError(t, pp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      4,
				Attributes: attribute.NewSet(attribute.String("processor", "nop"), attribute.String("otel.signal", "profiles")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      1,
				Attributes: attribute.NewSet(attribute.String("processor", "nop"), attribute.String("otel.signal", "profiles")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestProfiles_RecordIn_ErrorOut(t *testing.T) {
	// Regardless of input, return error
	mockErr := func(_ context.Context, _ pprofile.Profiles) (pprofile.Profiles, error) {
		return pprofile.NewProfiles(), errors.New("fake")
	}

	incomingProfiles := pprofile.NewProfiles()
	samples := incomingProfiles.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples()

	// Add 4 samples to the incoming
	samples.AppendEmpty()
	samples.AppendEmpty()
	samples.AppendEmpty()
	samples.AppendEmpty()

	tel := componenttest.NewTelemetry()
	pp, err := NewProfiles(context.Background(), newProfilesSettings(tel), &testProfilesCfg, consumertest.NewNop(), mockErr)
	require.NoError(t, err)

	require.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))
	require.Error(t, pp.ConsumeProfiles(context.Background(), incomingProfiles))
	require.NoError(t, pp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorIncomingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      4,
				Attributes: attribute.NewSet(attribute.String("processor", "nop"), attribute.String("otel.signal", "profiles")),
			},
		}, metricdatatest.IgnoreTimestamp())
	metadatatest.AssertEqualProcessorOutgoingItems(t, tel,
		[]metricdata.DataPoint[int64]{
			{
				Value:      0,
				Attributes: attribute.NewSet(attribute.String("processor", "nop"), attribute.String("otel.signal", "profiles")),
			},
		}, metricdatatest.IgnoreTimestamp())
}

func TestProfiles_ProcessInternalDuration(t *testing.T) {
	mockAggregate := func(_ context.Context, _ pprofile.Profiles) (pprofile.Profiles, error) {
		pd := pprofile.NewProfiles()
		pd.ResourceProfiles().AppendEmpty().ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()
		return pd, nil
	}

	incomingProfiles := pprofile.NewProfiles()

	tel := componenttest.NewTelemetry()
	pp, err := NewProfiles(context.Background(), newProfilesSettings(tel), &testProfilesCfg, consumertest.NewNop(), mockAggregate)
	require.NoError(t, err)

	assert.NoError(t, pp.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, pp.ConsumeProfiles(context.Background(), incomingProfiles))
	assert.NoError(t, pp.Shutdown(context.Background()))

	metadatatest.AssertEqualProcessorInternalDuration(t, tel,
		[]metricdata.HistogramDataPoint[float64]{
			{
				Count:        1,
				BucketCounts: []uint64{1},
				Attributes:   attribute.NewSet(attribute.String("processor", "nop"), attribute.String("otel.signal", "profiles")),
			},
		}, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func TestProfiles_SpanEvents(t *testing.T) {
	sr := new(tracetest.SpanRecorder)
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	ctx, span := tp.Tracer("test").Start(context.Background(), "test")

	pp, err := NewProfiles(context.Background(), processortest.NewNopSettings(processortest.NopType), &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil))
	require.NoError(t, err)
	require.NoError(t, pp.ConsumeProfiles(ctx, pprofile.NewProfiles()))
	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 1)
	events := spans[0].Events()
	require.Len(t, events, 2)
	assert.Equal(t, "Start processing.", events[0].Name)
	assert.Equal(t, "End processing.", events[1].Name)
	for _, ev := range events {
		assert.Equal(t, []attribute.KeyValue{attribute.String("processor", "nop")}, ev.Attributes)
	}
}

// errorMeter is a meter that returns errors when creating counters.
type errorMeter struct {
	noop.Meter
}

func (errorMeter) Int64Counter(string, ...metric.Int64CounterOption) (metric.Int64Counter, error) {
	return nil, errors.New("counter creation error")
}

// errorMeterProvider provides errorMeter instances.
type errorMeterProvider struct {
	noop.MeterProvider
}

func (errorMeterProvider) Meter(string, ...metric.MeterOption) metric.Meter {
	return errorMeter{}
}

func TestNewProfiles_TelemetryError(t *testing.T) {
	set := processortest.NewNopSettings(processortest.NopType)
	set.MeterProvider = errorMeterProvider{}

	_, err := NewProfiles(context.Background(), set, &testProfilesCfg, consumertest.NewNop(), newTestPProcessor(nil))
	require.ErrorContains(t, err, "counter creation error")
}
