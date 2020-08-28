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

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
	"go.opentelemetry.io/collector/internal/dataold"
	"go.opentelemetry.io/collector/internal/goldendataset"
)

func TestSameMetrics(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	actual := goldendataset.DefaultMetricData()
	diffs := diffMetricData(expected, actual)
	assert.Nil(t, diffs)
}

func diffMetricData(expected dataold.MetricData, actual dataold.MetricData) []*MetricDiff {
	expectedRMSlice := expected.ResourceMetrics()
	actualRMSlice := actual.ResourceMetrics()
	return diffRMSlices(toSlice(expectedRMSlice), toSlice(actualRMSlice))
}

func toSlice(s dataold.ResourceMetricsSlice) (out []dataold.ResourceMetrics) {
	for i := 0; i < s.Len(); i++ {
		out = append(out, s.At(i))
	}
	return out
}

func TestDifferentValues(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	cfg := goldendataset.DefaultCfg()
	cfg.PtVal = 2
	actual := goldendataset.MetricDataFromCfg(cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

func TestDifferentNumPts(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	cfg := goldendataset.DefaultCfg()
	cfg.NumPtsPerMetric = 2
	actual := goldendataset.MetricDataFromCfg(cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 1, len(diffs))
}

func TestDifferentPtTypes(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	cfg := goldendataset.DefaultCfg()
	cfg.MetricDescriptorType = dataold.MetricTypeDouble
	actual := goldendataset.MetricDataFromCfg(cfg)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 3, len(diffs))
}

func TestHistogram(t *testing.T) {
	cfg1 := goldendataset.DefaultCfg()
	cfg1.MetricDescriptorType = dataold.MetricTypeHistogram
	expected := goldendataset.MetricDataFromCfg(cfg1)
	cfg2 := goldendataset.DefaultCfg()
	cfg2.MetricDescriptorType = dataold.MetricTypeHistogram
	cfg2.PtVal = 2
	actual := goldendataset.MetricDataFromCfg(cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 3, len(diffs))
}

func TestSummary(t *testing.T) {
	cfg1 := goldendataset.DefaultCfg()
	cfg1.MetricDescriptorType = dataold.MetricTypeSummary
	expected := goldendataset.MetricDataFromCfg(cfg1)
	cfg2 := goldendataset.DefaultCfg()
	cfg2.MetricDescriptorType = dataold.MetricTypeSummary
	cfg2.PtVal = 2
	actual := goldendataset.MetricDataFromCfg(cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Equal(t, 3, len(diffs))
}

func TestPDMToPDRM(t *testing.T) {
	md := dataold.NewMetricData()
	md.ResourceMetrics().Resize(1)
	rm := pdmToPDRM([]pdata.Metrics{pdatautil.MetricsFromOldInternalMetrics(md)})
	require.Equal(t, 1, len(rm))
}
