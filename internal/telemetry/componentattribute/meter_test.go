// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componentattribute_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"go.opentelemetry.io/collector/internal/telemetry/componentattribute"
)

func TestMPWA(t *testing.T) {
	reader := sdkMetric.NewManualReader()
	mp1 := sdkMetric.NewMeterProvider(sdkMetric.WithReader(reader))

	// Add attribute set
	extraAttrs2 := attribute.NewSet(
		attribute.String("extrakey2", "extraval2"),
	)
	mp2 := componentattribute.MeterProviderWithAttributes(mp1, extraAttrs2)

	// Replace attribute set
	extraAttrs3 := attribute.NewSet(
		attribute.String("extrakey3", "extraval3"),
	)
	mp3 := componentattribute.MeterProviderWithAttributes(mp2, extraAttrs3)

	noAttrs := attribute.NewSet()
	// Add a standard attribute on top of the extra attributes
	attrs4 := attribute.NewSet(
		attribute.String("stdkey", "stdval"),
	)
	expAttrs4 := attribute.NewSet(append(extraAttrs3.ToSlice(), attrs4.ToSlice()...)...)
	// Overwrite the extra attribute
	attrs5 := attribute.NewSet(
		attribute.String("extrakey3", "customval"),
	)

	tests := []struct {
		mp       metric.MeterProvider
		attrs    attribute.Set
		expAttrs attribute.Set
		name     string
	}{
		{mp: mp1, attrs: noAttrs, expAttrs: noAttrs, name: "no extra attributes"},
		{mp: mp2, attrs: noAttrs, expAttrs: extraAttrs2, name: "set extra attributes"},
		{mp: mp3, attrs: noAttrs, expAttrs: extraAttrs3, name: "reset extra attributes"},
		{mp: mp3, attrs: attrs4, expAttrs: expAttrs4, name: "merge attributes"},
		{mp: mp3, attrs: attrs5, expAttrs: attrs5, name: "overwrite extra attribute"},
	}

	for i, test := range tests {
		t.Run(test.name+"/send", func(t *testing.T) {
			meterName := fmt.Sprintf("testmeter%d", i+1)
			ctr, err := test.mp.Meter(meterName).Int64Counter("testctr")
			require.NoError(t, err)
			ctr.Add(context.Background(), 42, metric.WithAttributeSet(test.attrs))
		})
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.LessOrEqual(t, len(rm.ScopeMetrics), len(tests))

	for i, test := range tests {
		t.Run(test.name+"/check", func(t *testing.T) {
			meterName := fmt.Sprintf("testmeter%d", i+1)
			i := slices.IndexFunc(rm.ScopeMetrics, func(sm metricdata.ScopeMetrics) bool {
				return sm.Scope.Name == meterName
			})
			assert.NotEqual(t, i, -1)
			metrics := rm.ScopeMetrics[i].Metrics
			require.Len(t, metrics, 1)
			assert.Equal(t, "testctr", metrics[0].Name)
			sum, ok := metrics[0].Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.Len(t, sum.DataPoints, 1)
			assert.Equal(t, test.expAttrs, sum.DataPoints[0].Attributes)
			assert.Equal(t, int64(42), sum.DataPoints[0].Value)
		})
	}
}

func TestMPWAInstruments(t *testing.T) {
	reader := sdkMetric.NewManualReader()
	mp := sdkMetric.NewMeterProvider(sdkMetric.WithReader(reader))
	attrs := attribute.NewSet(
		attribute.String("extrakey", "extraval"),
	)
	mpwa := componentattribute.MeterProviderWithAttributes(mp, attrs)

	tests := []struct {
		testName  string
		sendData  func(t *testing.T, meter metric.Meter)
		checkData func(t *testing.T, data metricdata.Metrics)
	}{
		{
			testName: "Int64Counter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Int64Counter("intctr")
				require.NoError(t, err)
				inst.Add(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				require.Equal(t, "intctr", metrics.Name)
				sum, ok := metrics.Data.(metricdata.Sum[int64])
				require.True(t, ok)
				require.Len(t, sum.DataPoints, 1)
				point := sum.DataPoints[0]
				assert.Equal(t, int64(42), point.Value)
				assert.Equal(t, attrs, point.Attributes)
			},
		},
		// A lot of instruments to go through, so we won't be as thorough in the check in the later ones
		{
			testName: "Int64UpDownCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Int64UpDownCounter("intctr")
				require.NoError(t, err)
				inst.Add(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Int64Histogram",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Int64Histogram("inthist")
				require.NoError(t, err)
				inst.Record(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Histogram[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Int64Gauge",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Int64Gauge("intgauge")
				require.NoError(t, err)
				inst.Record(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Gauge[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Int64ObservableCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Int64ObservableCounter("intctr", metric.WithInt64Callback(func(_ context.Context, io metric.Int64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Int64ObservableUpDownCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Int64ObservableUpDownCounter("intctr", metric.WithInt64Callback(func(_ context.Context, io metric.Int64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Int64ObservableGauge",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Int64ObservableGauge("intctr", metric.WithInt64Callback(func(_ context.Context, io metric.Int64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Gauge[int64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "RegisterCallback/Int64ObservableCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Int64ObservableCounter("intctr")
				require.NoError(t, err)
				_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveInt64(inst, 42)
					return nil
				}, inst)
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[int64]).DataPoints[0].Attributes)
			},
		},

		// And now the float instruments (mostly copypasted from above)
		{
			testName: "Float64Counter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Float64Counter("floatctr")
				require.NoError(t, err)
				inst.Add(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64UpDownCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Float64UpDownCounter("floatctr")
				require.NoError(t, err)
				inst.Add(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64Histogram",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Float64Histogram("floathist")
				require.NoError(t, err)
				inst.Record(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Histogram[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64Gauge",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Float64Gauge("floatgauge")
				require.NoError(t, err)
				inst.Record(context.Background(), 42)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Gauge[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64ObservableCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Float64ObservableCounter("floatctr", metric.WithFloat64Callback(func(_ context.Context, io metric.Float64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64ObservableUpDownCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Float64ObservableUpDownCounter("floatctr", metric.WithFloat64Callback(func(_ context.Context, io metric.Float64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "Float64ObservableGauge",
			sendData: func(t *testing.T, meter metric.Meter) {
				_, err := meter.Float64ObservableGauge("floatctr", metric.WithFloat64Callback(func(_ context.Context, io metric.Float64Observer) error {
					io.Observe(42)
					return nil
				}))
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Gauge[float64]).DataPoints[0].Attributes)
			},
		},
		{
			testName: "RegisterCallback/Float64ObservableCounter",
			sendData: func(t *testing.T, meter metric.Meter) {
				inst, err := meter.Float64ObservableCounter("floatctr")
				require.NoError(t, err)
				_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
					o.ObserveFloat64(inst, 42)
					return nil
				}, inst)
				require.NoError(t, err)
			},
			checkData: func(t *testing.T, metrics metricdata.Metrics) {
				assert.Equal(t, attrs, metrics.Data.(metricdata.Sum[float64]).DataPoints[0].Attributes)
			},
		},
	}

	for i, test := range tests {
		t.Run(test.testName+"/send", func(t *testing.T) {
			meter := mpwa.Meter(fmt.Sprintf("testmeter%d", i+1))
			test.sendData(t, meter)
		})
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	require.LessOrEqual(t, len(rm.ScopeMetrics), len(tests))

	for i, test := range tests {
		t.Run(test.testName+"/check", func(t *testing.T) {
			meterName := fmt.Sprintf("testmeter%d", i+1)
			i := slices.IndexFunc(rm.ScopeMetrics, func(sm metricdata.ScopeMetrics) bool {
				return sm.Scope.Name == meterName
			})
			assert.NotEqual(t, i, -1)
			metrics := rm.ScopeMetrics[i].Metrics
			require.Len(t, metrics, 1)
			test.checkData(t, metrics[0])
		})
	}
}
