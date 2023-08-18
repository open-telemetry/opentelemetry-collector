// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testdata

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

var (
	metricStartTimestamp    = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC))
	metricExemplarTimestamp = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC))
	metricTimestamp         = pcommon.NewTimestampFromTime(time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC))
)

const (
	TestGaugeDoubleMetricName          = "gauge-double"
	TestGaugeIntMetricName             = "gauge-int"
	TestSumDoubleMetricName            = "sum-double"
	TestSumIntMetricName               = "sum-int"
	TestHistogramMetricName            = "histogram"
	TestExponentialHistogramMetricName = "exponential-histogram"
	TestSummaryMetricName              = "summary"
)

func generateMetricsOneEmptyInstrumentationScope() pmetric.Metrics {
	md := pmetric.NewMetrics()
	initResource(md.ResourceMetrics().AppendEmpty().Resource())
	md.ResourceMetrics().At(0).ScopeMetrics().AppendEmpty()
	return md
}

func GenerateMetricsAllTypesEmpty() pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()

	doubleGauge := ms.AppendEmpty()
	initMetric(doubleGauge, TestGaugeDoubleMetricName, pmetric.MetricTypeGauge)
	doubleGauge.Gauge().DataPoints().AppendEmpty()
	intGauge := ms.AppendEmpty()
	initMetric(intGauge, TestGaugeIntMetricName, pmetric.MetricTypeGauge)
	intGauge.Gauge().DataPoints().AppendEmpty()
	doubleSum := ms.AppendEmpty()
	initMetric(doubleSum, TestSumDoubleMetricName, pmetric.MetricTypeSum)
	doubleSum.Sum().DataPoints().AppendEmpty()
	intSum := ms.AppendEmpty()
	initMetric(intSum, TestSumIntMetricName, pmetric.MetricTypeSum)
	intSum.Sum().DataPoints().AppendEmpty()
	histogram := ms.AppendEmpty()
	initMetric(histogram, TestHistogramMetricName, pmetric.MetricTypeHistogram)
	histogram.Histogram().DataPoints().AppendEmpty()
	summary := ms.AppendEmpty()
	initMetric(summary, TestSummaryMetricName, pmetric.MetricTypeSummary)
	summary.Summary().DataPoints().AppendEmpty()
	return md
}

func GenerateMetricsMetricTypeInvalid() pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	initMetric(md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics().AppendEmpty(), TestSumIntMetricName, pmetric.MetricTypeEmpty)
	return md
}

func GenerateMetricsAllTypes() pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	initGaugeIntMetric(ms.AppendEmpty())
	initGaugeDoubleMetric(ms.AppendEmpty())
	initSumIntMetric(ms.AppendEmpty())
	initSumDoubleMetric(ms.AppendEmpty())
	initHistogramMetric(ms.AppendEmpty())
	initExponentialHistogramMetric(ms.AppendEmpty())
	initSummaryMetric(ms.AppendEmpty())
	return md
}

func GenerateMetrics(count int) pmetric.Metrics {
	md := generateMetricsOneEmptyInstrumentationScope()
	ms := md.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	ms.EnsureCapacity(count)
	for i := 0; i < count; i++ {
		switch i % 7 {
		case 0:
			initGaugeIntMetric(ms.AppendEmpty())
		case 1:
			initGaugeDoubleMetric(ms.AppendEmpty())
		case 2:
			initSumIntMetric(ms.AppendEmpty())
		case 3:
			initSumDoubleMetric(ms.AppendEmpty())
		case 4:
			initHistogramMetric(ms.AppendEmpty())
		case 5:
			initExponentialHistogramMetric(ms.AppendEmpty())
		case 6:
			initSummaryMetric(ms.AppendEmpty())
		}
	}
	return md
}

func initGaugeIntMetric(im pmetric.Metric) {
	initMetric(im, TestGaugeIntMetricName, pmetric.MetricTypeGauge)

	idps := im.Gauge().DataPoints()
	idp0 := idps.AppendEmpty()
	initMetricAttributes1(idp0.Attributes())
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetIntValue(123)
	idp1 := idps.AppendEmpty()
	initMetricAttributes2(idp1.Attributes())
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetIntValue(456)
}

func initGaugeDoubleMetric(im pmetric.Metric) {
	initMetric(im, TestGaugeDoubleMetricName, pmetric.MetricTypeGauge)

	idps := im.Gauge().DataPoints()
	idp0 := idps.AppendEmpty()
	initMetricAttributes12(idp0.Attributes())
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetDoubleValue(1.23)
	idp1 := idps.AppendEmpty()
	initMetricAttributes13(idp1.Attributes())
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetDoubleValue(4.56)
}

func initSumIntMetric(im pmetric.Metric) {
	initMetric(im, TestSumIntMetricName, pmetric.MetricTypeSum)

	idps := im.Sum().DataPoints()
	idp0 := idps.AppendEmpty()
	initMetricAttributes1(idp0.Attributes())
	idp0.SetStartTimestamp(metricStartTimestamp)
	idp0.SetTimestamp(metricTimestamp)
	idp0.SetIntValue(123)
	idp1 := idps.AppendEmpty()
	initMetricAttributes2(idp1.Attributes())
	idp1.SetStartTimestamp(metricStartTimestamp)
	idp1.SetTimestamp(metricTimestamp)
	idp1.SetIntValue(456)
}

func initSumDoubleMetric(dm pmetric.Metric) {
	initMetric(dm, TestSumDoubleMetricName, pmetric.MetricTypeSum)

	ddps := dm.Sum().DataPoints()
	ddp0 := ddps.AppendEmpty()
	initMetricAttributes12(ddp0.Attributes())
	ddp0.SetStartTimestamp(metricStartTimestamp)
	ddp0.SetTimestamp(metricTimestamp)
	ddp0.SetDoubleValue(1.23)

	ddp1 := ddps.AppendEmpty()
	initMetricAttributes13(ddp1.Attributes())
	ddp1.SetStartTimestamp(metricStartTimestamp)
	ddp1.SetTimestamp(metricTimestamp)
	ddp1.SetDoubleValue(4.56)
}

func initHistogramMetric(hm pmetric.Metric) {
	initMetric(hm, TestHistogramMetricName, pmetric.MetricTypeHistogram)

	hdps := hm.Histogram().DataPoints()
	hdp0 := hdps.AppendEmpty()
	initMetricAttributes13(hdp0.Attributes())
	hdp0.SetStartTimestamp(metricStartTimestamp)
	hdp0.SetTimestamp(metricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)

	hdp1 := hdps.AppendEmpty()
	initMetricAttributes2(hdp1.Attributes())
	hdp1.SetStartTimestamp(metricStartTimestamp)
	hdp1.SetTimestamp(metricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetMin(15)
	hdp1.SetMax(15)
	hdp1.BucketCounts().FromRaw([]uint64{0, 1})
	exemplar := hdp1.Exemplars().AppendEmpty()
	exemplar.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	exemplar.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetTimestamp(metricExemplarTimestamp)
	exemplar.SetDoubleValue(15)
	initMetricExemplarAttributes(exemplar.FilteredAttributes())
	hdp1.ExplicitBounds().FromRaw([]float64{1})
}

func initExponentialHistogramMetric(hm pmetric.Metric) {
	initMetric(hm, TestExponentialHistogramMetricName, pmetric.MetricTypeExponentialHistogram)

	hdps := hm.ExponentialHistogram().DataPoints()
	hdp0 := hdps.AppendEmpty()
	initMetricAttributes13(hdp0.Attributes())
	hdp0.SetStartTimestamp(metricStartTimestamp)
	hdp0.SetTimestamp(metricTimestamp)
	hdp0.SetCount(5)
	hdp0.SetSum(0.15)
	hdp0.SetZeroCount(1)
	hdp0.SetScale(1)

	// positive index 1 and 2 are values sqrt(2), 2 at scale 1
	hdp0.Positive().SetOffset(1)
	hdp0.Positive().BucketCounts().FromRaw([]uint64{1, 1})
	// negative index -1 and 0 are values -1/sqrt(2), -1 at scale 1
	hdp0.Negative().SetOffset(-1)
	hdp0.Negative().BucketCounts().FromRaw([]uint64{1, 1})

	// The above will print:
	// Bucket (-1.414214, -1.000000], Count: 1
	// Bucket (-1.000000, -0.707107], Count: 1
	// Bucket [0, 0], Count: 1
	// Bucket [0.707107, 1.000000), Count: 1
	// Bucket [1.000000, 1.414214), Count: 1

	hdp1 := hdps.AppendEmpty()
	initMetricAttributes2(hdp1.Attributes())
	hdp1.SetStartTimestamp(metricStartTimestamp)
	hdp1.SetTimestamp(metricTimestamp)
	hdp1.SetCount(3)
	hdp1.SetSum(1.25)
	hdp1.SetMin(0)
	hdp1.SetMax(1)
	hdp1.SetZeroCount(1)
	hdp1.SetScale(-1)

	// index -1 and 0 are values 0.25, 1 at scale -1
	hdp1.Positive().SetOffset(-1)
	hdp1.Positive().BucketCounts().FromRaw([]uint64{1, 1})

	// The above will print:
	// Bucket [0, 0], Count: 1
	// Bucket [0.250000, 1.000000), Count: 1
	// Bucket [1.000000, 4.000000), Count: 1

	exemplar := hdp1.Exemplars().AppendEmpty()
	exemplar.SetTraceID([16]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10})
	exemplar.SetSpanID([8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18})
	exemplar.SetTimestamp(metricExemplarTimestamp)
	exemplar.SetIntValue(15)
	initMetricExemplarAttributes(exemplar.FilteredAttributes())
}

func initSummaryMetric(sm pmetric.Metric) {
	initMetric(sm, TestSummaryMetricName, pmetric.MetricTypeSummary)

	sdps := sm.Summary().DataPoints()
	sdp0 := sdps.AppendEmpty()
	initMetricAttributes13(sdp0.Attributes())
	sdp0.SetStartTimestamp(metricStartTimestamp)
	sdp0.SetTimestamp(metricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)

	sdp1 := sdps.AppendEmpty()
	initMetricAttributes2(sdp1.Attributes())
	sdp1.SetStartTimestamp(metricStartTimestamp)
	sdp1.SetTimestamp(metricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)

	quantile := sdp1.QuantileValues().AppendEmpty()
	quantile.SetQuantile(0.01)
	quantile.SetValue(15)
}

func initMetric(m pmetric.Metric, name string, ty pmetric.MetricType) {
	m.SetName(name)
	m.SetDescription("")
	m.SetUnit("1")
	switch ty {
	case pmetric.MetricTypeGauge:
		m.SetEmptyGauge()
	case pmetric.MetricTypeSum:
		sum := m.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	case pmetric.MetricTypeHistogram:
		histo := m.SetEmptyHistogram()
		histo.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	case pmetric.MetricTypeExponentialHistogram:
		histo := m.SetEmptyExponentialHistogram()
		histo.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
	case pmetric.MetricTypeSummary:
		m.SetEmptySummary()
	}
}
