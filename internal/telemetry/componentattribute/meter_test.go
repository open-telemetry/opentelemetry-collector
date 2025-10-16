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
			ctr, err := test.mp.Meter(
				meterName,
				metric.WithInstrumentationAttributes(test.attrs.ToSlice()...),
			).Int64Counter("testctr")
			require.NoError(t, err)
			ctr.Add(context.Background(), 42)
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
			assert.NotEqual(t, -1, i)
			assert.Equal(t, test.expAttrs, rm.ScopeMetrics[i].Scope.Attributes)
			metrics := rm.ScopeMetrics[i].Metrics
			require.Len(t, metrics, 1)
			assert.Equal(t, "testctr", metrics[0].Name)
			sum, ok := metrics[0].Data.(metricdata.Sum[int64])
			require.True(t, ok)
			require.Len(t, sum.DataPoints, 1)
			assert.Equal(t, int64(42), sum.DataPoints[0].Value)
		})
	}
}
