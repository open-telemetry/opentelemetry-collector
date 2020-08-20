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

package goldendataset

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/data"
)

func TestGenDefault(t *testing.T) {
	md := DefaultMetricData()
	mCount, ptCount := md.MetricAndDataPointCount()
	require.Equal(t, 1, mCount)
	require.Equal(t, 1, ptCount)
	rms := md.ResourceMetrics()
	rm := rms.At(0)
	resource := rm.Resource()
	rattrs := resource.Attributes()
	rattrs.Len()
	require.Equal(t, 1, rattrs.Len())
	val, _ := rattrs.Get("resource-attr-name-0")
	require.Equal(t, "resource-attr-val-0", val.StringVal())
	ilms := rm.InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	ms := ilms.At(0).Metrics()
	require.Equal(t, 1, ms.Len())
	pdm := ms.At(0)
	desc := pdm.MetricDescriptor()
	require.Equal(t, "metric_0", desc.Name())
	require.Equal(t, "my-md-description", desc.Description())
	require.Equal(t, "my-md-units", desc.Unit())

	pts := pdm.Int64DataPoints()
	require.Equal(t, 1, pts.Len())
	pt := pts.At(0)

	require.Equal(t, 1, pt.LabelsMap().Len())
	ptLabels, _ := pt.LabelsMap().Get("pt-label-key-0")
	require.Equal(t, "pt-label-val-0", ptLabels.Value())

	require.EqualValues(t, 940000000000000000, pt.StartTime())
	require.EqualValues(t, 940000000000000042, pt.Timestamp())
	require.EqualValues(t, 1, pt.Value())
}

func TestHistogramFunctions(t *testing.T) {
	pt := pdata.NewHistogramDataPoint()
	pt.InitEmpty()
	setHistogramBounds(pt, 1, 2, 3, 4, 5)
	require.Equal(t, 5, len(pt.ExplicitBounds()))
	require.Equal(t, 5, pt.Buckets().Len())

	addHistogramVal(pt, 1, 0)
	require.EqualValues(t, 1, pt.Count())
	require.EqualValues(t, 1, pt.Sum())
	require.EqualValues(t, 1, pt.Buckets().At(0).Count())

	addHistogramVal(pt, 2, 0)
	require.EqualValues(t, 2, pt.Count())
	require.EqualValues(t, 3, pt.Sum())
	require.EqualValues(t, 1, pt.Buckets().At(1).Count())

	addHistogramVal(pt, 2, 0)
	require.EqualValues(t, 3, pt.Count())
	require.EqualValues(t, 5, pt.Sum())
	require.EqualValues(t, 2, pt.Buckets().At(1).Count())
}

func TestGenHistogram(t *testing.T) {
	cfg := DefaultCfg()
	cfg.MetricDescriptorType = pdata.MetricTypeHistogram
	cfg.PtVal = 2
	md := MetricDataFromCfg(cfg)
	pts := getMetric(md).HistogramDataPoints()
	pt := pts.At(0)
	buckets := pt.Buckets()
	require.Equal(t, 5, buckets.Len())
	middleBucket := buckets.At(2)
	require.EqualValues(t, 2, middleBucket.Count())
	ex := pt.Buckets().At(0).Exemplar()
	ex.Value()
	require.EqualValues(t, 1, ex.Value())
}

func TestSummaryFunctions(t *testing.T) {
	pt := pdata.NewSummaryDataPoint()
	pt.InitEmpty()
	setSummaryPercentiles(pt, 0, 50, 95)
	addSummaryValue(pt, 55, 0)
	addSummaryValue(pt, 70, 1)
	addSummaryValue(pt, 90, 2)
	require.EqualValues(t, 3, pt.Count())
	v := pt.ValueAtPercentiles().At(2).Value()
	require.EqualValues(t, 1, v)
	pctile := pt.ValueAtPercentiles().At(2).Percentile()
	require.EqualValues(t, 95, pctile)
}

func TestGenSummary(t *testing.T) {
	cfg := DefaultCfg()
	cfg.MetricDescriptorType = pdata.MetricTypeSummary
	md := MetricDataFromCfg(cfg)
	metric := getMetric(md)
	pts := metric.SummaryDataPoints()
	require.Equal(t, 1, pts.Len())
	pt := pts.At(0)
	require.EqualValues(t, 3, pt.Count())
	require.EqualValues(t, 215, pt.Sum())
}

func TestGenDouble(t *testing.T) {
	cfg := DefaultCfg()
	cfg.MetricDescriptorType = pdata.MetricTypeDouble
	md := MetricDataFromCfg(cfg)
	metric := getMetric(md)
	pts := metric.DoubleDataPoints()
	require.Equal(t, 1, pts.Len())
	pt := pts.At(0)
	require.EqualValues(t, 1, pt.Value())
}

func getMetric(md data.MetricData) pdata.Metric {
	return md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0)
}
