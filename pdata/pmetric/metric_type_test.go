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

package pmetric // import "go.opentelemetry.io/collector/pdata/pmetric"

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricTypeString(t *testing.T) {
	assert.Equal(t, "Empty", MetricTypeEmpty.String())
	assert.Equal(t, "Gauge", MetricTypeGauge.String())
	assert.Equal(t, "Sum", MetricTypeSum.String())
	assert.Equal(t, "Histogram", MetricTypeHistogram.String())
	assert.Equal(t, "ExponentialHistogram", MetricTypeExponentialHistogram.String())
	assert.Equal(t, "Summary", MetricTypeSummary.String())
	assert.Equal(t, "", (MetricTypeSummary + 1).String())
}
