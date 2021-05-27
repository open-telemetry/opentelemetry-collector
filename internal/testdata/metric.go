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

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpcollectormetrics "go.opentelemetry.io/collector/internal/data/protogen/collector/metrics/v1"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/protogen/metrics/v1"
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

func generateMetricsOtlpOneEmptyResourceMetrics() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{},
		},
	}
}

func GenerateMetricsNoLibraries() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().At(0)
	initResource1(ms0.Resource())
	return md
}

func generateMetricsOtlpNoLibraries() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
			},
		},
	}
}

func GenerateMetricsOneEmptyInstrumentationLibrary() pdata.Metrics {
	md := GenerateMetricsNoLibraries()
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().AppendEmpty()
	return md
}

// generateMetricsOtlpOneEmptyInstrumentationLibrary returns the OTLP representation of the GenerateMetricsOneEmptyInstrumentationLibrary.
func generateMetricsOtlpOneEmptyInstrumentationLibrary() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{},
				},
			},
		},
	}
}

func GenerateMetricsOneMetricNoResource() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().At(0)
	rm0ils0 := rm0.InstrumentationLibraryMetrics().AppendEmpty()
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func generateMetricsOtlpOneMetricNoResource() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							generateOtlpCounterIntMetric(),
						},
					},
				},
			},
		},
	}
}

func GenerateMetricsOneMetric() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	initCounterIntMetric(rm0ils0.Metrics().AppendEmpty())
	return md
}

func generateMetricsOtlpOneMetric() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							generateOtlpCounterIntMetric(),
						},
					},
				},
			},
		},
	}
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

func generateMetricsOtlpTwoMetrics() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							generateOtlpCounterIntMetric(),
							generateOtlpCounterIntMetric(),
						},
					},
				},
			},
		},
	}
}

func GenerateMetricsOneMetricNoLabels() pdata.Metrics {
	md := GenerateMetricsOneMetric()
	dps := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints()
	dps.At(0).LabelsMap().InitFromMap(map[string]string{})
	dps.At(1).LabelsMap().InitFromMap(map[string]string{})
	return md
}

func generateMetricsOtlpOneMetricNoLabels() *otlpcollectormetrics.ExportMetricsServiceRequest {
	md := generateMetricsOtlpOneMetric()
	mis := md.ResourceMetrics[0].InstrumentationLibraryMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_IntSum).IntSum
	mis.DataPoints[0].Labels = nil
	mis.DataPoints[1].Labels = nil
	return md
}

func GenerateMetricsAllTypesNoDataPoints() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	initMetric(ms.AppendEmpty(), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	initMetric(ms.AppendEmpty(), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	initMetric(ms.AppendEmpty(), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
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
	initMetric(doubleGauge, TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	doubleGauge.DoubleGauge().DataPoints().AppendEmpty()
	intGauge := ms.AppendEmpty()
	initMetric(intGauge, TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	intGauge.IntGauge().DataPoints().AppendEmpty()
	doubleSum := ms.AppendEmpty()
	initMetric(doubleSum, TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	doubleSum.DoubleSum().DataPoints().AppendEmpty()
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

func generateMetricsOtlpAllTypesNoDataPoints() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							generateOtlpMetric(TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge),
							generateOtlpMetric(TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge),
							generateOtlpMetric(TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum),
							generateOtlpMetric(TestCounterIntMetricName, pdata.MetricDataTypeIntSum),
							generateOtlpMetric(TestDoubleHistogramMetricName, pdata.MetricDataTypeHistogram),
							generateOtlpMetric(TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram),
							generateOtlpMetric(TestDoubleSummaryMetricName, pdata.MetricDataTypeSummary),
						},
					},
				},
			},
		},
	}
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

func generateMetricsOtlpAllTypesWithSampleDatapoints() *otlpcollectormetrics.ExportMetricsServiceRequest {
	return &otlpcollectormetrics.ExportMetricsServiceRequest{
		ResourceMetrics: []*otlpmetrics.ResourceMetrics{
			{
				Resource: generateOtlpResource1(),
				InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
					{
						Metrics: []*otlpmetrics.Metric{
							generateOtlpCounterIntMetric(),
							generateOtlpSumDoubleMetric(),
							generateOtlpDoubleHistogramMetric(),
							generateOtlpIntHistogramMetric(),
							generateOTLPDoubleSummaryMetric(),
						},
					},
				},
			},
		},
	}
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

func generateOtlpCounterIntMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	m.Data.(*otlpmetrics.Metric_IntSum).IntSum.DataPoints =
		[]*otlpmetrics.IntDataPoint{
			{
				Labels:            generateOtlpMetricLabels1(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Value:             123,
			},
			{
				Labels:            generateOtlpMetricLabels2(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Value:             456,
			},
		}
	return m
}

func initSumDoubleMetric(dm pdata.Metric) {
	initMetric(dm, TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)

	ddps := dm.DoubleSum().DataPoints()
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

func generateOtlpSumDoubleMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	m.Data.(*otlpmetrics.Metric_DoubleSum).DoubleSum.DataPoints =
		[]*otlpmetrics.DoubleDataPoint{
			{
				Labels:            generateOtlpMetricLabels12(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Value:             1.23,
			},
			{
				Labels:            generateOtlpMetricLabels13(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Value:             4.56,
			},
		}
	return m
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

func generateOtlpDoubleHistogramMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestDoubleHistogramMetricName, pdata.MetricDataTypeHistogram)
	m.Data.(*otlpmetrics.Metric_DoubleHistogram).DoubleHistogram.DataPoints =
		[]*otlpmetrics.DoubleHistogramDataPoint{
			{
				Labels:            generateOtlpMetricLabels13(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
			},
			{
				Labels:            generateOtlpMetricLabels2(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
				BucketCounts:      []uint64{0, 1},
				ExplicitBounds:    []float64{1},
				Exemplars: []otlpmetrics.DoubleExemplar{
					{
						FilteredLabels: generateOtlpMetricAttachment(),
						TimeUnixNano:   uint64(TestMetricExemplarTimestamp),
						Value:          15,
					},
				},
			},
		}
	return m
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

func generateOtlpIntHistogramMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	m.Data.(*otlpmetrics.Metric_IntHistogram).IntHistogram.DataPoints =
		[]*otlpmetrics.IntHistogramDataPoint{
			{
				Labels:            generateOtlpMetricLabels13(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
			},
			{
				Labels:            generateOtlpMetricLabels2(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
				BucketCounts:      []uint64{0, 1},
				ExplicitBounds:    []float64{1},
				Exemplars: []otlpmetrics.IntExemplar{
					{
						FilteredLabels: generateOtlpMetricAttachment(),
						TimeUnixNano:   uint64(TestMetricExemplarTimestamp),
						Value:          15,
					},
				},
			},
		}
	return m
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

func generateOTLPDoubleSummaryMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestDoubleSummaryMetricName, pdata.MetricDataTypeSummary)
	m.Data.(*otlpmetrics.Metric_DoubleSummary).DoubleSummary.DataPoints =
		[]*otlpmetrics.DoubleSummaryDataPoint{
			{
				Labels:            generateOtlpMetricLabels13(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
			},
			{
				Labels:            generateOtlpMetricLabels2(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
				QuantileValues: []*otlpmetrics.DoubleSummaryDataPoint_ValueAtQuantile{
					{
						Quantile: 0.01,
						Value:    15,
					},
				},
			},
		}
	return m
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
	case pdata.MetricDataTypeDoubleSum:
		sum := m.DoubleSum()
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

func generateOtlpMetric(name string, ty pdata.MetricDataType) *otlpmetrics.Metric {
	m := &otlpmetrics.Metric{
		Name:        name,
		Description: "",
		Unit:        "1",
	}
	switch ty {
	case pdata.MetricDataTypeIntGauge:
		m.Data = &otlpmetrics.Metric_IntGauge{IntGauge: &otlpmetrics.IntGauge{}}
	case pdata.MetricDataTypeDoubleGauge:
		m.Data = &otlpmetrics.Metric_DoubleGauge{DoubleGauge: &otlpmetrics.DoubleGauge{}}
	case pdata.MetricDataTypeIntSum:
		m.Data = &otlpmetrics.Metric_IntSum{IntSum: &otlpmetrics.IntSum{
			IsMonotonic:            true,
			AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		}}
	case pdata.MetricDataTypeDoubleSum:
		m.Data = &otlpmetrics.Metric_DoubleSum{DoubleSum: &otlpmetrics.DoubleSum{
			IsMonotonic:            true,
			AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		}}
	case pdata.MetricDataTypeIntHistogram:
		m.Data = &otlpmetrics.Metric_IntHistogram{IntHistogram: &otlpmetrics.IntHistogram{
			AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		}}
	case pdata.MetricDataTypeHistogram:
		m.Data = &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{
			AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		}}
	case pdata.MetricDataTypeSummary:
		m.Data = &otlpmetrics.Metric_DoubleSummary{DoubleSummary: &otlpmetrics.DoubleSummary{}}
	}
	return m
}

func GenerateMetricsManyMetricsSameResource(metricsCount int) pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rs0ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rs0ilm0.Metrics().Resize(metricsCount)
	for i := 0; i < metricsCount; i++ {
		initCounterIntMetric(rs0ilm0.Metrics().At(i))
	}
	return md
}
