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

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/model/pdata"
)

var (
	TestMetricStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestMetricStartTimestamp = pdata.TimestampFromTime(TestMetricStartTime)

	TestMetricExemplarTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestMetricExemplarTimestamp = pdata.TimestampFromTime(TestMetricExemplarTime)

	TestMetricTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestMetricTimestamp = pdata.TimestampFromTime(TestMetricTime)
)

const (
	TestGaugeDoubleMetricName     = "gauge-double"
	TestGaugeIntMetricName        = "gauge-int"
	TestCounterDoubleMetricName   = "counter-double"
	TestCounterIntMetricName      = "counter-int"
	TestDoubleHistogramMetricName = "double-histogram"
	TestIntHistogramMetricName    = "int-histogram"
	TestDoubleSummaryMetricName   = "double-summary"
)

func GenerateMetricsOneEmptyResourceMetrics() pdata.Metrics {
	md := pdata.NewMetrics()
	md.ResourceMetrics().AppendEmpty()
	return md
}

func GenerateMetricsNoLibraries() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().At(0)
	initResource1(ms0.Resource())
	return md
}

func GenerateMetricsOneEmptyInstrumentationLibrary() pdata.Metrics {
	md := GenerateMetricsNoLibraries()
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().AppendEmpty()
	return md
}

func GenerateMetricsOneMetricNoResource() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().At(0)
	rm0ils0 := rm0.InstrumentationLibraryMetrics().AppendEmpty()
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func GenerateMetricsOneMetric() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func GenerateMetricsOneMetricOneDataPoint() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initGaugeIntMetricOneDataPoint(rm0ils0.Metrics().AppendEmpty())
	return md
}

func GenerateMetricsTwoMetrics() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func GenerateMetricsOneCounterOneSummaryMetrics() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	initDoubleSummaryMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func GenerateMetricsOneMetricNoLabels() pdata.Metrics {
	md := GenerateMetricsOneMetric()
	dps := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints()
	dps.At(0).LabelsMap().InitFromMap(map[string]string{})
	dps.At(1).LabelsMap().InitFromMap(map[string]string{})
	return md
}

func GenerateMetricsAllTypesNoDataPoints() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	initMetric(ms.AppendEmpty(), TestGaugeDoubleMetricName, pdata.MetricDataTypeGauge)
	initMetric(ms.AppendEmpty(), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	initMetric(ms.AppendEmpty(), TestCounterDoubleMetricName, pdata.MetricDataTypeSum)
	initMetric(ms.AppendEmpty(), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	initMetric(ms.AppendEmpty(), TestDoubleHistogramMetricName, pdata.MetricDataTypeHistogram)
	initMetric(ms.AppendEmpty(), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	initMetric(ms.AppendEmpty(), TestDoubleSummaryMetricName, pdata.MetricDataTypeSummary)
	return md
}

func GenerateMetricsAllTypesEmptyDataPoint() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()

	doubleGauge := ms.AppendEmpty()
	initMetric(doubleGauge, TestGaugeDoubleMetricName, pdata.MetricDataTypeGauge)
	doubleGauge.Gauge().DataPoints().AppendEmpty()
	intGauge := ms.AppendEmpty()
	initMetric(intGauge, TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	intGauge.IntGauge().DataPoints().AppendEmpty()
	doubleSum := ms.AppendEmpty()
	initMetric(doubleSum, TestCounterDoubleMetricName, pdata.MetricDataTypeSum)
	doubleSum.Sum().DataPoints().AppendEmpty()
	intSum := ms.AppendEmpty()
	initMetric(intSum, TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	intSum.IntSum().DataPoints().AppendEmpty()
	histogram := ms.AppendEmpty()
	initMetric(histogram, TestDoubleHistogramMetricName, pdata.MetricDataTypeHistogram)
	histogram.Histogram().DataPoints().AppendEmpty()
	intHistogram := ms.AppendEmpty()
	initMetric(intHistogram, TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	intHistogram.IntHistogram().DataPoints().AppendEmpty()
	summary := ms.AppendEmpty()
	initMetric(summary, TestDoubleSummaryMetricName, pdata.MetricDataTypeSummary)
	summary.Summary().DataPoints().AppendEmpty()
	return md
}

func GenerateMetricsMetricTypeInvalid() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initMetric(ilm0.Metrics().AppendEmpty(), TestCounterIntMetricName, pdata.MetricDataTypeNone)
	return md
}

func GeneratMetricsAllTypesWithSampleDatapoints() pdata.Metrics {
	metricData := pdata.NewMetrics()
	rm := metricData.ResourceMetrics().AppendEmpty()
	initResource1(rm.Resource())

	ilm := rm.InstrumentationLibraryMetrics().AppendEmpty()
	ms := ilm.Metrics()
	initCounterIntMetric(ms.AppendEmpty())
	initSumDoubleMetric(ms.AppendEmpty())
	initDoubleHistogramMetric(ms.AppendEmpty())
	initIntHistogramMetric(ms.AppendEmpty())
	initDoubleSummaryMetric(ms.AppendEmpty())

	return metricData
}

func initCounterIntMetric(im pdata.Metric) {
	initMetric(im, TestCounterIntMetricName, pdata.MetricDataTypeIntSum)

	idps := im.IntSum().DataPoints()
	idp0 := idps.AppendEmpty()
	initMetricLabels1(idp0.LabelsMap())
	idp0.SetStartTimestamp(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
	idp1 := idps.AppendEmpty()
	initMetricLabels2(idp1.LabelsMap())
	idp1.SetStartTimestamp(TestMetricStartTimestamp)
	idp1.SetTimestamp(TestMetricTimestamp)
	idp1.SetValue(456)
}

func initGaugeIntMetricOneDataPoint(im pdata.Metric) {
	initMetric(im, TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)

	idp0 := im.IntGauge().DataPoints().AppendEmpty()
	initMetricLabels1(idp0.LabelsMap())
	idp0.SetStartTimestamp(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
}

func initSumDoubleMetric(dm pdata.Metric) {
	initMetric(dm, TestCounterDoubleMetricName, pdata.MetricDataTypeSum)

	ddps := dm.Sum().DataPoints()
	ddp0 := ddps.AppendEmpty()
	initMetricLabels12(ddp0.LabelsMap())
	ddp0.SetStartTimestamp(TestMetricStartTimestamp)
	ddp0.SetTimestamp(TestMetricTimestamp)
	ddp0.SetValue(1.23)

	ddp1 := ddps.AppendEmpty()
	initMetricLabels13(ddp1.LabelsMap())
	ddp1.SetStartTimestamp(TestMetricStartTimestamp)
	ddp1.SetTimestamp(TestMetricTimestamp)
	ddp1.SetValue(4.56)
}

func initDoubleHistogramMetric(hm pdata.Metric) {
	initMetric(hm, TestDoubleHistogramMetricName, pdata.MetricDataTypeHistogram)

	hdps := hm.Histogram().DataPoints()
	hdp0 := hdps.AppendEmpty()
	initMetricLabels13(hdp0.LabelsMap())
	hdp0.SetStartTimestamp(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hdps.AppendEmpty()
	initMetricLabels2(hdp1.LabelsMap())
	hdp1.SetStartTimestamp(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetBucketCounts([]uint64{0, 1})
	exemplar := hdp1.Exemplars().AppendEmpty()
	exemplar.SetTimestamp(TestMetricExemplarTimestamp)
	exemplar.SetValue(15)
	initMetricAttachment(exemplar.FilteredLabels())
	hdp1.SetExplicitBounds([]float64{1})
}

func initIntHistogramMetric(hm pdata.Metric) {
	initMetric(hm, TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)

	hdps := hm.IntHistogram().DataPoints()
	hdp0 := hdps.AppendEmpty()
	initMetricLabels13(hdp0.LabelsMap())
	hdp0.SetStartTimestamp(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hdps.AppendEmpty()
	initMetricLabels2(hdp1.LabelsMap())
	hdp1.SetStartTimestamp(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetBucketCounts([]uint64{0, 1})
	exemplar := hdp1.Exemplars().AppendEmpty()
	exemplar.SetTimestamp(TestMetricExemplarTimestamp)
	exemplar.SetValue(15)
	initMetricAttachment(exemplar.FilteredLabels())
	hdp1.SetExplicitBounds([]float64{1})
}

func initDoubleSummaryMetric(sm pdata.Metric) {
	initMetric(sm, TestDoubleSummaryMetricName, pdata.MetricDataTypeSummary)

	sdps := sm.Summary().DataPoints()
	sdp0 := sdps.AppendEmpty()
	initMetricLabels13(sdp0.LabelsMap())
	sdp0.SetStartTimestamp(TestMetricStartTimestamp)
	sdp0.SetTimestamp(TestMetricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)
	sdp1 := sdps.AppendEmpty()
	initMetricLabels2(sdp1.LabelsMap())
	sdp1.SetStartTimestamp(TestMetricStartTimestamp)
	sdp1.SetTimestamp(TestMetricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)

	quantile := sdp1.QuantileValues().AppendEmpty()
	quantile.SetQuantile(0.01)
	quantile.SetValue(15)
}

func initMetric(m pdata.Metric, name string, ty pdata.MetricDataType) {
	m.SetName(name)
	m.SetDescription("")
	m.SetUnit("1")
	m.SetDataType(ty)
	switch ty {
	case pdata.MetricDataTypeIntSum:
		sum := m.IntSum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	case pdata.MetricDataTypeSum:
		sum := m.Sum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	case pdata.MetricDataTypeIntHistogram:
		histo := m.IntHistogram()
		histo.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	case pdata.MetricDataTypeHistogram:
		histo := m.Histogram()
		histo.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
	}
}

func GenerateMetricsManyMetricsSameResource(metricsCount int) pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rs0ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rs0ilm0.Metrics().EnsureCapacity(metricsCount)
	for i := 0; i < metricsCount; i++ {
		initCounterIntMetric(rs0ilm0.Metrics().AppendEmpty())
	}
	return md
}
