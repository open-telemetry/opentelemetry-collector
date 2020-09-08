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
	"go.opentelemetry.io/collector/internal/goldendataset"
)

func TestSameMetrics(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	actual := goldendataset.DefaultMetricData()
	diffs := diffMetricData(expected, actual)
	assert.Nil(t, diffs)
}

func diffMetricData(expected pdata.Metrics, actual pdata.Metrics) []*MetricDiff {
	expectedRMSlice := expected.ResourceMetrics()
	actualRMSlice := actual.ResourceMetrics()
	return diffRMSlices(toSlice(expectedRMSlice), toSlice(actualRMSlice))
}

func toSlice(s pdata.ResourceMetricsSlice) (out []pdata.ResourceMetrics) {
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
	assert.Len(t, diffs, 1)
}

func TestDifferentNumPts(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	cfg := goldendataset.DefaultCfg()
	cfg.NumPtsPerMetric = 2
	actual := goldendataset.MetricDataFromCfg(cfg)
	diffs := diffMetricData(expected, actual)
	assert.Len(t, diffs, 1)
}

func TestDifferentPtTypes(t *testing.T) {
	expected := goldendataset.DefaultMetricData()
	cfg := goldendataset.DefaultCfg()
	cfg.MetricDescriptorType = pdata.MetricDataTypeDoubleGauge
	actual := goldendataset.MetricDataFromCfg(cfg)
	diffs := diffMetricData(expected, actual)
	assert.Len(t, diffs, 1)
}

func TestDoubleHistogram(t *testing.T) {
	cfg1 := goldendataset.DefaultCfg()
	cfg1.MetricDescriptorType = pdata.MetricDataTypeDoubleHistogram
	expected := goldendataset.MetricDataFromCfg(cfg1)
	cfg2 := goldendataset.DefaultCfg()
	cfg2.MetricDescriptorType = pdata.MetricDataTypeDoubleHistogram
	cfg2.PtVal = 2
	actual := goldendataset.MetricDataFromCfg(cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Len(t, diffs, 3)
}

func TestIntHistogram(t *testing.T) {
	cfg1 := goldendataset.DefaultCfg()
	cfg1.MetricDescriptorType = pdata.MetricDataTypeIntHistogram
	expected := goldendataset.MetricDataFromCfg(cfg1)
	cfg2 := goldendataset.DefaultCfg()
	cfg2.MetricDescriptorType = pdata.MetricDataTypeIntHistogram
	cfg2.PtVal = 2
	actual := goldendataset.MetricDataFromCfg(cfg2)
	diffs := diffMetricData(expected, actual)
	assert.Len(t, diffs, 3)
}

func TestPDMToPDRM(t *testing.T) {
	md := pdata.NewMetrics()
	md.ResourceMetrics().Resize(2)
	rms := pdmToPDRM([]pdata.Metrics{md})
	require.Len(t, rms, 2)
}
