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
