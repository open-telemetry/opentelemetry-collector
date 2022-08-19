// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otlptext

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/sdk/metric/aggregator/exponential/mapping/exponent"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/exponential/mapping/logarithm"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestMetricsText(t *testing.T) {
	type args struct {
		md pmetric.Metrics
	}
	tests := []struct {
		name  string
		args  args
		empty bool
	}{
		{"empty metrics", args{pmetric.NewMetrics()}, true},
		{"metrics with all types and datapoints", args{testdata.GenerateMetricsAllTypes()}, false},
		{"metrics with all types without datapoints", args{testdata.GenerateMetricsAllTypesEmpty()}, false},
		{"metrics with invalid metric types", args{testdata.GenerateMetricsMetricTypeInvalid()}, false},
		{"metrics with lots of metrics", args{testdata.GenerateMetrics(10)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := NewTextMetricsMarshaler().MarshalMetrics(tt.args.md)
			assert.NoError(t, err)
			if !tt.empty {
				assert.NotEmpty(t, metrics)
			}
		})
	}
}

func TestExpoHistoOverflowMapping(t *testing.T) {
	// Compute the string value of 0x1p+1024
	b := big.NewFloat(1)
	b.SetMantExp(b, 1024)
	expect := b.String()

	// For positive scales, the +Inf threshold happens at 1024<<scale
	for scale := logarithm.MinScale; scale <= logarithm.MaxScale; scale++ {
		m := newExpoHistoMapping(scale)
		threshold := int32(1024) << scale

		// Check that this can't be mapped to a float64.
		_, err := m.mapping.LowerBoundary(threshold)
		require.Error(t, err)

		// Require the special case.
		require.Equal(t, expect, m.stringLowerBoundary(threshold, false))
		require.Equal(t, "-"+expect, m.stringLowerBoundary(threshold, true))

		require.Equal(t, "OVERFLOW", m.stringLowerBoundary(threshold+1, false))
		require.Equal(t, "-OVERFLOW", m.stringLowerBoundary(threshold+1, true))
	}
	// For scales <= 0, the +Inf threshold happens at 1024>>-scale
	for scale := exponent.MinScale; scale <= exponent.MaxScale; scale++ {
		m := newExpoHistoMapping(scale)
		threshold := int32(1024) >> -scale

		require.Equal(t, expect, m.stringLowerBoundary(threshold, false))
		require.Equal(t, "-"+expect, m.stringLowerBoundary(threshold, true))

		require.Equal(t, "OVERFLOW", m.stringLowerBoundary(threshold+1, false))
		require.Equal(t, "-OVERFLOW", m.stringLowerBoundary(threshold+1, true))
	}

	// For an invalid scale, any positive index overflows
	invalid := newExpoHistoMapping(100)

	require.Equal(t, "OVERFLOW", invalid.stringLowerBoundary(1, false))
	require.Equal(t, "-OVERFLOW", invalid.stringLowerBoundary(1, true))

	// But index 0 always works
	require.Equal(t, "1", invalid.stringLowerBoundary(0, false))
	require.Equal(t, "-1", invalid.stringLowerBoundary(0, true))
}

func TestExpoHistoUnderflowMapping(t *testing.T) {
	// For all valid scales
	for scale := int32(-10); scale <= 20; scale++ {
		m := newExpoHistoMapping(scale)
		idx := m.mapping.MapToIndex(0x1p-1022)
		lb, err := m.mapping.LowerBoundary(idx)
		require.NoError(t, err)

		require.Equal(t, fmt.Sprintf("%g", lb), m.stringLowerBoundary(idx, false))
		require.Equal(t, fmt.Sprintf("-%g", lb), m.stringLowerBoundary(idx, true))

		require.Equal(t, "UNDERFLOW", m.stringLowerBoundary(idx-1, false))
		require.Equal(t, "-UNDERFLOW", m.stringLowerBoundary(idx-1, true))
	}

	// For an invalid scale, any positive index overflows
	invalid := newExpoHistoMapping(100)

	require.Equal(t, "UNDERFLOW", invalid.stringLowerBoundary(-1, false))
	require.Equal(t, "-UNDERFLOW", invalid.stringLowerBoundary(-1, true))
}
