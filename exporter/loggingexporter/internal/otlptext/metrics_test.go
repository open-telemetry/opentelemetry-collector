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
	"testing"

	"github.com/stretchr/testify/assert"

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
