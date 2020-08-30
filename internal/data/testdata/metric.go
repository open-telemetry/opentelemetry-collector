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
	"go.opentelemetry.io/collector/internal/data"
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
	NumMetricTests                = 14
)

func GenerateMetricsEmpty() data.MetricData {
	md := data.NewMetricData()
	return md
}

func generateMetricsOtlpEmpty() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics(nil)
}

func GenerateMetricsOneEmptyResourceMetrics() data.MetricData {
	md := GenerateMetricsEmpty()
	md.ResourceMetrics().Resize(1)
	return md
}

func generateMetricsOtlpOneEmptyResourceMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{},
	}
}

func GenerateMetricsOneEmptyOneNilResourceMetrics() data.MetricData {
	return data.MetricDataFromOtlp(generateMetricsOtlpOneEmptyOneNilResourceMetrics())
}

func generateMetricsOtlpOneEmptyOneNilResourceMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{},
		nil,
	}
}

func GenerateMetricsNoLibraries() data.MetricData {
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

func GenerateMetricsOneEmptyInstrumentationLibrary() data.MetricData {
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

func GenerateMetricsOneEmptyOneNilInstrumentationLibrary() data.MetricData {
	return data.MetricDataFromOtlp(generateMetricsOtlpOneEmptyOneNilInstrumentationLibrary())
}

func generateMetricsOtlpOneEmptyOneNilInstrumentationLibrary() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{},
				nil,
			},
		},
	}
}

func GenerateMetricsOneMetricNoResource() data.MetricData {
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

func GenerateMetricsOneMetric() data.MetricData {
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

func GenerateMetricsOneMetricOneDataPoint() data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initGaugeIntMetricOneDataPoint(rm0ils0.Metrics().At(0))
	return md
}

func GenerateMetricsTwoMetrics() data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(2)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	initCounterIntMetric(rm0ils0.Metrics().At(1))
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

func GenerateMetricsOneMetricOneNil() data.MetricData {
	return data.MetricDataFromOtlp(generateMetricsOtlpOneMetricOneNil())
}

func generateMetricsOtlpOneMetricOneNil() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						generateOtlpCounterIntMetric(),
						nil,
					},
				},
			},
		},
	}
}

func GenerateMetricsOneMetricNoLabels() data.MetricData {
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

func GenerateMetricsOneMetricOneNilPoint() data.MetricData {
	return data.MetricDataFromOtlp(generateMetricsOtlpOneMetricOneNilPoint())
}

func generateMetricsOtlpOneMetricOneNilPoint() []*otlpmetrics.ResourceMetrics {
	md := generateMetricsOtlpOneMetric()
	mis := md[0].InstrumentationLibraryMetrics[0].Metrics[0].Data.(*otlpmetrics.Metric_IntSum).IntSum
	mis.DataPoints = append(mis.DataPoints, nil)
	return md
}

func GenerateMetricsAllTypesNoDataPoints() data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)
	initMetric(ms.At(0), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	initMetric(ms.At(1), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	initMetric(ms.At(2), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	initMetric(ms.At(3), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	initMetric(ms.At(4), TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
	initMetric(ms.At(5), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	return md
}

func GenerateMetricsAllTypesNilDataPoint() data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)

	nilInt := pdata.NewIntDataPoint()
	nilDouble := pdata.NewDoubleDataPoint()
	nilDoubleHistogram := pdata.NewDoubleHistogramDataPoint()
	nilIntHistogram := pdata.NewIntHistogramDataPoint()

	initMetric(ms.At(0), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	ms.At(0).DoubleGauge().DataPoints().Append(&nilDouble)
	initMetric(ms.At(1), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	ms.At(1).IntGauge().DataPoints().Append(&nilInt)
	initMetric(ms.At(2), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	ms.At(2).DoubleSum().DataPoints().Append(&nilDouble)
	initMetric(ms.At(3), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	ms.At(3).IntSum().DataPoints().Append(&nilInt)
	initMetric(ms.At(4), TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
	ms.At(4).DoubleHistogram().DataPoints().Append(&nilDoubleHistogram)
	initMetric(ms.At(5), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	ms.At(5).IntHistogram().DataPoints().Append(&nilIntHistogram)

	return md
}

func GenerateMetricsAllTypesEmptyDataPoint() data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)

	emptyInt := pdata.NewIntDataPoint()
	emptyInt.InitEmpty()
	emptyDouble := pdata.NewDoubleDataPoint()
	emptyDouble.InitEmpty()
	emptyDoubleHistogram := pdata.NewDoubleHistogramDataPoint()
	emptyDoubleHistogram.InitEmpty()
	emptyIntHistogram := pdata.NewIntHistogramDataPoint()
	emptyIntHistogram.InitEmpty()

	initMetric(ms.At(0), TestGaugeDoubleMetricName, pdata.MetricDataTypeDoubleGauge)
	ms.At(0).DoubleGauge().DataPoints().Append(&emptyDouble)
	initMetric(ms.At(1), TestGaugeIntMetricName, pdata.MetricDataTypeIntGauge)
	ms.At(1).IntGauge().DataPoints().Append(&emptyInt)
	initMetric(ms.At(2), TestCounterDoubleMetricName, pdata.MetricDataTypeDoubleSum)
	ms.At(2).DoubleSum().DataPoints().Append(&emptyDouble)
	initMetric(ms.At(3), TestCounterIntMetricName, pdata.MetricDataTypeIntSum)
	ms.At(3).IntSum().DataPoints().Append(&emptyInt)
	initMetric(ms.At(4), TestDoubleHistogramMetricName, pdata.MetricDataTypeDoubleHistogram)
	ms.At(4).DoubleHistogram().DataPoints().Append(&emptyDoubleHistogram)
	initMetric(ms.At(5), TestIntHistogramMetricName, pdata.MetricDataTypeIntHistogram)
	ms.At(5).IntHistogram().DataPoints().Append(&emptyIntHistogram)
	return md
}

func GenerateMetricsMetricTypeInvalid() data.MetricData {
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
					},
				},
			},
		},
	}
}

func GenerateMetricsWithCountersHistograms() data.MetricData {
	metricData := data.NewMetricData()
	metricData.ResourceMetrics().Resize(1)

	rms := metricData.ResourceMetrics()
	initResource1(rms.At(0).Resource())
	rms.At(0).InstrumentationLibraryMetrics().Resize(1)

	ilms := rms.At(0).InstrumentationLibraryMetrics()
	ilms.At(0).Metrics().Resize(4)
	ms := ilms.At(0).Metrics()
	initCounterIntMetric(ms.At(0))
	initSumDoubleMetric(ms.At(1))
	initDoubleHistogramMetric(ms.At(2))
	initIntHistogramMetric(ms.At(3))

	return metricData
}

func generateMetricsOtlpWithCountersHistograms() []*otlpmetrics.ResourceMetrics {
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

func initMetric(m pdata.Metric, name string, ty pdata.MetricDataType) {
	m.InitEmpty()
	m.SetName(name)
	m.SetDescription("")
	m.SetUnit("1")
	switch ty {
	case pdata.MetricDataTypeIntGauge:
		md := pdata.NewIntGauge()
		md.InitEmpty()
		m.SetIntGauge(md)
	case pdata.MetricDataTypeDoubleGauge:
		md := pdata.NewDoubleGauge()
		md.InitEmpty()
		m.SetDoubleGauge(md)
	case pdata.MetricDataTypeIntSum:
		md := pdata.NewIntSum()
		md.InitEmpty()
		md.SetIsMonotonic(true)
		md.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.SetIntSum(md)
	case pdata.MetricDataTypeDoubleSum:
		md := pdata.NewDoubleSum()
		md.InitEmpty()
		md.SetIsMonotonic(true)
		md.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.SetDoubleSum(md)
	case pdata.MetricDataTypeIntHistogram:
		md := pdata.NewIntHistogram()
		md.InitEmpty()
		md.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.SetIntHistogram(md)
	case pdata.MetricDataTypeDoubleHistogram:
		md := pdata.NewDoubleHistogram()
		md.InitEmpty()
		md.SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
		m.SetDoubleHistogram(md)
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
	}
	return m
}

func GenerateMetricsManyMetricsSameResource(metricsCount int) data.MetricData {
	md := GenerateMetricsOneEmptyInstrumentationLibrary()
	rs0ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rs0ilm0.Metrics().Resize(metricsCount)
	for i := 0; i < metricsCount; i++ {
		initCounterIntMetric(rs0ilm0.Metrics().At(i))
	}
	return md
}
