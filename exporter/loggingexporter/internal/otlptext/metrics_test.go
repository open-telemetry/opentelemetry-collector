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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lightstep/go-expohisto/mapping/exponent"
	"github.com/lightstep/go-expohisto/mapping/logarithm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestMetricsText(t *testing.T) {
	tests := []struct {
		name string
		in   pmetric.Metrics
		out  string
	}{
		{
			name: "empty_metrics",
			in:   pmetric.NewMetrics(),
			out:  "empty.out",
		},
		{
			name: "metrics_with_all_types",
			in:   testdata.GenerateMetricsAllTypes(),
			out:  "metrics_with_all_types.out",
		},
		{
			name: "two_metrics",
			in:   testdata.GenerateMetrics(2),
			out:  "two_metrics.out",
		},
		{
			name: "invalid_metric_type",
			in:   testdata.GenerateMetricsMetricTypeInvalid(),
			out:  "invalid_metric_type.out",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTextMetricsMarshaler().MarshalMetrics(tt.in)
			assert.NoError(t, err)
			out, err := os.ReadFile(filepath.Join("testdata", "metrics", tt.out))
			require.NoError(t, err)
			expected := strings.ReplaceAll(string(out), "\r", "")
			assert.Equal(t, expected, string(got))
		})
	}
}

func TestExpoHistoOverflowMapping(t *testing.T) {
	// Compute the string value of 0x1p+1024
	expect := expectGreatestBoundary()

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
	require.Equal(t, "1.000000", invalid.stringLowerBoundary(0, false))
	require.Equal(t, "-1.000000", invalid.stringLowerBoundary(0, true))
}

func TestExpoHistoUnderflowMapping(t *testing.T) {
	// For all valid scales
	for scale := int32(-10); scale <= 20; scale++ {
		m := newExpoHistoMapping(scale)

		// The following +1 gives us the index whose lower
		// boundary equals 0x1p-1022.
		idx := m.mapping.MapToIndex(0x1p-1022) + 1
		lb, err := m.mapping.LowerBoundary(idx)
		require.NoError(t, err)

		require.Equal(t, fmt.Sprintf("%f", lb), m.stringLowerBoundary(idx, false))
		require.Equal(t, fmt.Sprintf("-%f", lb), m.stringLowerBoundary(idx, true))

		require.Equal(t, "UNDERFLOW", m.stringLowerBoundary(idx-1, false))
		require.Equal(t, "-UNDERFLOW", m.stringLowerBoundary(idx-1, true))
	}

	// For an invalid scale, any positive index overflows
	invalid := newExpoHistoMapping(100)

	require.Equal(t, "UNDERFLOW", invalid.stringLowerBoundary(-1, false))
	require.Equal(t, "-UNDERFLOW", invalid.stringLowerBoundary(-1, true))
}

// expectGreatestBoundary is the definitive logic to compute the
// decimal value for 0x1p1024, which has float64 representation equal
// to +Inf so we can't use ordinary logic to format.
func expectGreatestBoundary() string {
	b := big.NewFloat(1)
	b.SetMantExp(b, 1024)
	return b.Text('g', 6)
}

// TestGreatestBoundary verifies greatestBoundary is hard-coded
// correctly.
func TestGreatestBoundary(t *testing.T) {
	expect := expectGreatestBoundary()

	require.Equal(t, expect, greatestBoundary)
}
