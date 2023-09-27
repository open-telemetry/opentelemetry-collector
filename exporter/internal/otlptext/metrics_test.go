// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otlptext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
