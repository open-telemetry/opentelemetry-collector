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
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1"
)

var (
	TestMetricStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestMetricStartTimestamp = pdata.TimestampUnixNano(TestMetricStartTime.UnixNano())

	TestMetricExemplarTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestMetricExemplarTimestamp = pdata.TimestampUnixNano(TestMetricExemplarTime.UnixNano())

	TestMetricTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestMetricTimestamp = pdata.TimestampUnixNano(TestMetricTime.UnixNano())
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

func GenerateMetricsEmpty() pdata.Metrics {
	md := pdata.NewMetrics()
	return md
}

func generateMetricsOtlpEmpty() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics(nil)
}

func GenerateMetricsOneEmptyResourceMetrics() pdata.Metrics {
	md := GenerateMetricsEmpty()
	md.ResourceMetrics().Resize(1)
	return md
}

func generateMetricsOtlpOneEmptyResourceMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{},
	}
}

func GenerateMetricsNoLibraries() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().At(0)
	initResource1(ms0.Resource())
	return md
}

func generateMetricsOtlpNoLibraries() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func GenerateMetricsOneEmptyInstrumentationLibrary() pdata.Metrics {
	md := GenerateMetricsNoLibraries()
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	return md
}

// generateMetricsOtlpOneEmptyInstrumentationLibrary returns the OTLP representation of the GenerateMetricsOneEmptyInstrumentationLibrary.
func generateMetricsOtlpOneEmptyInstrumentationLibrary() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{},
			},
		},
	}
}

func GenerateMetricsOneMetricNoResource() pdata.Metrics {
	md := GenerateMetricsOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().At(0)
	rm0.InstrumentationLibraryMetrics().Resize(1)
	rm0ils0 := rm0.InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func generateMetricsOtlpOneMetricNoResource() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						generateOtlpCounterIntMetric(),
					},
				},
			},
		},
	}
}

func GenerateMetricsOneMetric() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func generateMetricsOtlpOneMetric() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
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
	}
}

func GenerateMetricsOneMetricOneDataPoint() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initGaugeIntMetricOneDataPoint(rm0ils0.Metrics().At(0))
	return md
}

func GenerateMetricsTwoMetrics() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(2)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	initCounterIntMetric(rm0ils0.Metrics().At(1))
	return md
}

func GenerateMetricsOneCounterOneSummaryMetrics() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(2)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	initDoubleSummaryMetric(rm0ils0.Metrics().At(1))
	return md
}

func GenerateMetricsOtlpTwoMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
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
	}
}

func GenerateMetricsOneMetricNoLabels() pdata.Metrics {
	md := GenerateMetricsOneMetric()
	dps := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).IntSum().DataPoints()
	dps.At(0).LabelsMap().InitFromMap(map[string]string{})
	dps.At(1).LabelsMap().InitFromMap(map[string]string{})
	return md
}

func generateMetricsOtlpOneMetricNoLabels() []*otlpmetrics.ResourceMetrics {
	md := generateMetricsOtlpOneMetric()
	mis := md[0].InstrumentationLibraryMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_IntSum).IntSum
	mis.DataPoints[0].Labels = nil
	mis.DataPoints[1].Labels = nil
	return md
}

func GenerateMetricsAllTypesNoDataPoints() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(7)
	initMetric(ms.At(0), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	initMetric(ms.At(1), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	initMetric(ms.At(2), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	initMetric(ms.At(3), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	initMetric(ms.At(4), TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
	initMetric(ms.At(5), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	initMetric(ms.At(6), TestDoubleSummaryMetricName, pdata.MetricDataTypeDoubleSummary)
	return md
}

func GenerateMetricsAllTypesEmptyDataPoint() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(7)

	initMetric(ms.At(0), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	ms.At(0).DoubleGauge().DataPoints().Resize(1)
	initMetric(ms.At(1), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	ms.At(1).IntGauge().DataPoints().Resize(1)
	initMetric(ms.At(2), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	ms.At(2).DoubleSum().DataPoints().Resize(1)
	initMetric(ms.At(3), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	ms.At(3).IntSum().DataPoints().Resize(1)
	initMetric(ms.At(4), TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
	ms.At(4).DoubleHistogram().DataPoints().Resize(1)
	initMetric(ms.At(5), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	ms.At(5).IntHistogram().DataPoints().Resize(1)
	initMetric(ms.At(6), TestDoubleSummaryMetricName, pdata.MetricDataTypeDoubleSummary)
	ms.At(6).DoubleSummary().DataPoints().Resize(1)
	return md
}

func GenerateMetricsMetricTypeInvalid() pdata.Metrics {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(1)

	initMetric(ms.At(0), TestCounterIntMetricName, pdata.MetricDataTypeNone)
	return md
}

func generateMetricsOtlpAllTypesNoDataPoints() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						generateOtlpMetric(TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge),
						generateOtlpMetric(TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge),
						generateOtlpMetric(TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum),
						generateOtlpMetric(TestCounterIntMetricName, pdata.MetricDataTypeIntSum),
						generateOtlpMetric(TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram),
						generateOtlpMetric(TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram),
						generateOtlpMetric(TestDoubleSummaryMetricName, pdata.MetricDataTypeDoubleSummary),
					},
				},
			},
		},
	}
}

func GeneratMetricsAllTypesWithSampleDatapoints() pdata.Metrics {
	metricData := pdata.NewMetrics()
	metricData.ResourceMetrics().Resize(1)

	rms := metricData.ResourceMetrics()
	initResource1(rms.At(0).Resource())
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)

	ilms := rms.At(0).InstrumentationLibraryMetrics()
	ilms.At(0).Metrics().Resize(5)
	ms := ilms.At(0).Metrics()
	initCounterIntMetric(ms.At(0))
	initSumDoubleMetric(ms.At(1))
	initDoubleHistogramMetric(ms.At(2))
	initIntHistogramMetric(ms.At(3))
	initDoubleSummaryMetric(ms.At(4))

	return metricData
}

func generateMetricsOtlpAllTypesWithSampleDatapoints() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
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
	}
}

func initCounterIntMetric(im pdata.Metric) {
	initMetric(im, TestCounterIntMetricName, pdata.MetricDataTypeIntSum)

	idps := im.IntSum().DataPoints()
	idps.Resize(2)
	idp0 := idps.At(0)
	initMetricLabels1(idp0.LabelsMap())
	idp0.SetStartTime(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
	idp1 := idps.At(1)
	initMetricLabels2(idp1.LabelsMap())
	idp1.SetStartTime(TestMetricStartTimestamp)
	idp1.SetTimestamp(TestMetricTimestamp)
	idp1.SetValue(456)
}

func initGaugeIntMetricOneDataPoint(im pdata.Metric) {
	initMetric(im, TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)

	idps := im.IntGauge().DataPoints()
	idps.Resize(1)
	idp0 := idps.At(0)
	initMetricLabels1(idp0.LabelsMap())
	idp0.SetStartTime(TestMetricStartTimestamp)
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
	ddps.Resize(2)
	ddp0 := ddps.At(0)
	initMetricLabels12(ddp0.LabelsMap())
	ddp0.SetStartTime(TestMetricStartTimestamp)
	ddp0.SetTimestamp(TestMetricTimestamp)
	ddp0.SetValue(1.23)

	ddp1 := ddps.At(1)
	initMetricLabels13(ddp1.LabelsMap())
	ddp1.SetStartTime(TestMetricStartTimestamp)
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
	initMetric(hm, TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)

	hdps := hm.DoubleHistogram().DataPoints()
	hdps.Resize(2)
	hdp0 := hdps.At(0)
	initMetricLabels13(hdp0.LabelsMap())
	hdp0.SetStartTime(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hdps.At(1)
	initMetricLabels2(hdp1.LabelsMap())
	hdp1.SetStartTime(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetBucketCounts([]uint64{0, 1})
	exemplars := hdp1.Exemplars()
	exemplars.Resize(1)
	exemplar := exemplars.At(0)
	exemplar.SetTimestamp(TestMetricExemplarTimestamp)
	exemplar.SetValue(15)
	initMetricAttachment(exemplar.FilteredLabels())
	hdp1.SetExplicitBounds([]float64{1})
}

func generateOtlpDoubleHistogramMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
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
				Exemplars: []*otlpmetrics.DoubleExemplar{
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
	hdps.Resize(2)
	hdp0 := hdps.At(0)
	initMetricLabels13(hdp0.LabelsMap())
	hdp0.SetStartTime(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hdps.At(1)
	initMetricLabels2(hdp1.LabelsMap())
	hdp1.SetStartTime(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetBucketCounts([]uint64{0, 1})
	exemplars := hdp1.Exemplars()
	exemplars.Resize(1)
	exemplar := exemplars.At(0)
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
				Exemplars: []*otlpmetrics.IntExemplar{
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
	initMetric(sm, TestDoubleSummaryMetricName, pdata.MetricDataTypeDoubleSummary)

	sdps := sm.DoubleSummary().DataPoints()
	sdps.Resize(2)
	sdp0 := sdps.At(0)
	initMetricLabels13(sdp0.LabelsMap())
	sdp0.SetStartTime(TestMetricStartTimestamp)
	sdp0.SetTimestamp(TestMetricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)
	sdp1 := sdps.At(1)
	initMetricLabels2(sdp1.LabelsMap())
	sdp1.SetStartTime(TestMetricStartTimestamp)
	sdp1.SetTimestamp(TestMetricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)

	quantiles := pdata.NewValueAtQuantileSlice()
	quantiles.Resize(1)
	quantiles.At(0).SetQuantile(0.01)
	quantiles.At(0).SetValue(15)

	quantiles.CopyTo(sdp1.QuantileValues())
}

func generateOTLPDoubleSummaryMetric() *otlpmetrics.Metric {
	m := generateOtlpMetric(TestDoubleSummaryMetricName, pdata.MetricDataTypeDoubleSummary)
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
	case pdata.MetricDataTypeDoubleHistogram:
		histo := m.DoubleHistogram()
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
	case pdata.MetricDataTypeDoubleHistogram:
		m.Data = &otlpmetrics.Metric_DoubleHistogram{DoubleHistogram: &otlpmetrics.DoubleHistogram{
			AggregationTemporality: otlpmetrics.AggregationTemporality_AGGREGATION_TEMPORALITY_CUMULATIVE,
		}}
	case pdata.MetricDataTypeDoubleSummary:
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
