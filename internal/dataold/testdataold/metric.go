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

package testdataold

import (
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
	otlpmetrics "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/metrics/v1old"
	"go.opentelemetry.io/collector/internal/data/testdata"
	"go.opentelemetry.io/collector/internal/dataold"
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
	TestSummaryMetricName = "summary"
	NumMetricTests        = 14
)

func GenerateMetricDataEmpty() dataold.MetricData {
	md := dataold.NewMetricData()
	return md
}

func generateMetricOtlpEmpty() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics(nil)
}

func GenerateMetricDataOneEmptyResourceMetrics() dataold.MetricData {
	md := GenerateMetricDataEmpty()
	md.ResourceMetrics().Resize(1)
	return md
}

func generateMetricOtlpOneEmptyResourceMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{},
	}
}

func GenerateMetricDataOneEmptyOneNilResourceMetrics() dataold.MetricData {
	return dataold.MetricDataFromOtlp(generateMetricOtlpOneEmptyOneNilResourceMetrics())
}

func generateMetricOtlpOneEmptyOneNilResourceMetrics() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{},
		nil,
	}
}

func GenerateMetricDataNoLibraries() dataold.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().At(0)
	initResource1(ms0.Resource())
	return md
}

func generateMetricOtlpNoLibraries() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
		},
	}
}

func GenerateMetricDataOneEmptyInstrumentationLibrary() dataold.MetricData {
	md := GenerateMetricDataNoLibraries()
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	return md
}

// generateMetricOtlpOneEmptyInstrumentationLibrary returns the OTLP representation of the GenerateMetricDataOneEmptyInstrumentationLibrary.
func generateMetricOtlpOneEmptyInstrumentationLibrary() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{},
			},
		},
	}
}

func GenerateMetricDataOneEmptyOneNilInstrumentationLibrary() dataold.MetricData {
	return dataold.MetricDataFromOtlp(generateMetricOtlpOneEmptyOneNilInstrumentationLibrary())
}

func generateMetricOtlpOneEmptyOneNilInstrumentationLibrary() []*otlpmetrics.ResourceMetrics {
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

func GenerateMetricDataOneMetricNoResource() dataold.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().At(0)
	rm0.InstrumentationLibraryMetrics().Resize(1)
	rm0ils0 := rm0.InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func generateMetricOtlpOneMetricNoResource() []*otlpmetrics.ResourceMetrics {
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

func GenerateMetricDataOneMetric() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func generateMetricOtlpOneMetric() []*otlpmetrics.ResourceMetrics {
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

func GenerateMetricDataOneMetricOneDataPoint() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initGaugeIntMetricOneDataPoint(rm0ils0.Metrics().At(0))
	return md
}

func GenerateMetricDataTwoMetrics() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(2)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	initCounterIntMetric(rm0ils0.Metrics().At(1))
	return md
}

func GenerateMetricOtlpTwoMetrics() []*otlpmetrics.ResourceMetrics {
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

func GenerateMetricDataOneMetricOneNil() dataold.MetricData {
	return dataold.MetricDataFromOtlp(generateMetricOtlpOneMetricOneNil())
}

func generateMetricOtlpOneMetricOneNil() []*otlpmetrics.ResourceMetrics {
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

func GenerateMetricDataOneMetricNoLabels() dataold.MetricData {
	md := GenerateMetricDataOneMetric()
	dps := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0).Metrics().At(0).Int64DataPoints()
	dps.At(0).LabelsMap().InitFromMap(map[string]string{})
	dps.At(1).LabelsMap().InitFromMap(map[string]string{})
	return md
}

func generateMetricOtlpOneMetricNoLabels() []*otlpmetrics.ResourceMetrics {
	md := generateMetricOtlpOneMetric()
	m := md[0].InstrumentationLibraryMetrics[0].Metrics[0]
	m.Int64DataPoints[0].Labels = nil
	m.Int64DataPoints[1].Labels = nil
	return md
}

func GenerateMetricDataOneMetricOneNilPoint() dataold.MetricData {
	return dataold.MetricDataFromOtlp(generateMetricOtlpOneMetricOneNilPoint())
}

func generateMetricOtlpOneMetricOneNilPoint() []*otlpmetrics.ResourceMetrics {
	md := generateMetricOtlpOneMetric()
	md[0].InstrumentationLibraryMetrics[0].Metrics[0].Int64DataPoints =
		append(md[0].InstrumentationLibraryMetrics[0].Metrics[0].Int64DataPoints, nil)
	return md
}

func GenerateMetricDataAllTypesNoDataPoints() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)
	initMetricDescriptor(
		ms.At(0).MetricDescriptor(), testdata.TestGaugeDoubleMetricName, dataold.MetricTypeDouble)
	initMetricDescriptor(
		ms.At(1).MetricDescriptor(), testdata.TestGaugeIntMetricName, dataold.MetricTypeInt64)
	initMetricDescriptor(
		ms.At(2).MetricDescriptor(), testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble)
	initMetricDescriptor(
		ms.At(3).MetricDescriptor(), testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64)
	initMetricDescriptor(
		ms.At(4).MetricDescriptor(), testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram)
	initMetricDescriptor(
		ms.At(5).MetricDescriptor(), TestSummaryMetricName, dataold.MetricTypeSummary)
	return md
}

func GenerateMetricDataAllTypesNilDataPoint() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)

	nilInt64 := dataold.NewInt64DataPoint()
	nilDouble := dataold.NewDoubleDataPoint()
	nilHistogram := dataold.NewHistogramDataPoint()
	nilSummary := dataold.NewSummaryDataPoint()

	initMetricDescriptor(
		ms.At(0).MetricDescriptor(), testdata.TestGaugeDoubleMetricName, dataold.MetricTypeDouble)
	ms.At(0).DoubleDataPoints().Append(&nilDouble)
	initMetricDescriptor(
		ms.At(1).MetricDescriptor(), testdata.TestGaugeIntMetricName, dataold.MetricTypeInt64)
	ms.At(1).Int64DataPoints().Append(&nilInt64)
	initMetricDescriptor(
		ms.At(2).MetricDescriptor(), testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble)
	ms.At(2).DoubleDataPoints().Append(&nilDouble)
	initMetricDescriptor(
		ms.At(3).MetricDescriptor(), testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64)
	ms.At(3).Int64DataPoints().Append(&nilInt64)
	initMetricDescriptor(
		ms.At(4).MetricDescriptor(), testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram)
	ms.At(4).HistogramDataPoints().Append(&nilHistogram)
	initMetricDescriptor(
		ms.At(5).MetricDescriptor(), TestSummaryMetricName, dataold.MetricTypeSummary)
	ms.At(5).SummaryDataPoints().Append(&nilSummary)
	return md
}

func GenerateMetricDataAllTypesEmptyDataPoint() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(6)

	emptyInt64 := dataold.NewInt64DataPoint()
	emptyInt64.InitEmpty()
	emptyDouble := dataold.NewDoubleDataPoint()
	emptyDouble.InitEmpty()
	emptyHistogram := dataold.NewHistogramDataPoint()
	emptyHistogram.InitEmpty()
	emptySummary := dataold.NewSummaryDataPoint()
	emptySummary.InitEmpty()

	initMetricDescriptor(
		ms.At(0).MetricDescriptor(), testdata.TestGaugeDoubleMetricName, dataold.MetricTypeDouble)
	ms.At(0).DoubleDataPoints().Append(&emptyDouble)
	initMetricDescriptor(
		ms.At(1).MetricDescriptor(), testdata.TestGaugeIntMetricName, dataold.MetricTypeInt64)
	ms.At(1).Int64DataPoints().Append(&emptyInt64)
	initMetricDescriptor(
		ms.At(2).MetricDescriptor(), testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble)
	ms.At(2).DoubleDataPoints().Append(&emptyDouble)
	initMetricDescriptor(
		ms.At(3).MetricDescriptor(), testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64)
	ms.At(3).Int64DataPoints().Append(&emptyInt64)
	initMetricDescriptor(
		ms.At(4).MetricDescriptor(), testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram)
	ms.At(4).HistogramDataPoints().Append(&emptyHistogram)
	initMetricDescriptor(
		ms.At(5).MetricDescriptor(), TestSummaryMetricName, dataold.MetricTypeSummary)
	ms.At(5).SummaryDataPoints().Append(&emptySummary)
	return md
}

func GenerateMetricDataNilMetricDescriptor() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(1)
	return md
}

func GenerateMetricDataMetricTypeInvalid() dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(1)

	initMetricDescriptor(
		ms.At(0).MetricDescriptor(), testdata.TestGaugeDoubleMetricName, dataold.MetricTypeInvalid)
	return md
}

func generateMetricOtlpAllTypesNoDataPoints() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						{
							MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestGaugeDoubleMetricName, dataold.MetricTypeDouble),
						},
						{
							MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestGaugeIntMetricName, dataold.MetricTypeInt64),
						},
						{
							MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble),
						},
						{
							MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64),
						},
						{
							MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram),
						},
						{
							MetricDescriptor: generateOtlpMetricDescriptor(TestSummaryMetricName, dataold.MetricTypeSummary),
						},
					},
				},
			},
		},
	}
}

func GenerateMetricDataWithCountersHistogramAndSummary() dataold.MetricData {
	metricData := dataold.NewMetricData()
	metricData.ResourceMetrics().Resize(1)

	rms := metricData.ResourceMetrics()

	rms.At(0).InstrumentationLibraryMetrics().Resize(1)
	initResource1(rms.At(0).Resource())

	ilms := rms.At(0).InstrumentationLibraryMetrics()
	ilms.At(0).Metrics().Resize(4)
	ms := ilms.At(0).Metrics()
	initCounterIntMetric(ms.At(0))
	initCounterDoubleMetric(ms.At(1))
	initCumulativeHistogramMetric(ms.At(2))
	initSummaryMetric(ms.At(3))

	return metricData
}

func generateMetricOtlpWithCountersHistogramAndSummary() []*otlpmetrics.ResourceMetrics {
	return []*otlpmetrics.ResourceMetrics{
		{
			Resource: generateOtlpResource1(),
			InstrumentationLibraryMetrics: []*otlpmetrics.InstrumentationLibraryMetrics{
				{
					Metrics: []*otlpmetrics.Metric{
						generateOtlpCounterIntMetric(),
						generateOtlpCounterDoubleMetric(),
						generateOtlpCumulativeHistogramMetric(),
						generateOtlpSummaryMetric(),
					},
				},
			},
		},
	}
}

func initCounterIntMetric(im dataold.Metric) {
	initMetricDescriptor(im.MetricDescriptor(), testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64)

	idps := im.Int64DataPoints()
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

func initGaugeIntMetricOneDataPoint(im dataold.Metric) {
	initMetricDescriptor(im.MetricDescriptor(), testdata.TestCounterIntMetricName, dataold.MetricTypeInt64)
	idps := im.Int64DataPoints()
	idps.Resize(1)
	idp0 := idps.At(0)
	initMetricLabels1(idp0.LabelsMap())
	idp0.SetStartTime(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
}

func generateOtlpCounterIntMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestCounterIntMetricName, dataold.MetricTypeMonotonicInt64),
		Int64DataPoints: []*otlpmetrics.Int64DataPoint{
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
		},
	}
}

func initCounterDoubleMetric(dm dataold.Metric) {
	initMetricDescriptor(dm.MetricDescriptor(), testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble)

	ddps := dm.DoubleDataPoints()
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

func generateOtlpCounterDoubleMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestCounterDoubleMetricName, dataold.MetricTypeMonotonicDouble),
		DoubleDataPoints: []*otlpmetrics.DoubleDataPoint{
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
		},
	}
}

func initCumulativeHistogramMetric(hm dataold.Metric) {
	initMetricDescriptor(hm.MetricDescriptor(), testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram)

	hdps := hm.HistogramDataPoints()
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
	hdp1.Buckets().Resize(2)
	hdp1.Buckets().At(0).SetCount(0)
	hdp1.Buckets().At(1).SetCount(1)
	exemplar := hdp1.Buckets().At(1).Exemplar()
	exemplar.InitEmpty()
	exemplar.SetTimestamp(TestMetricExemplarTimestamp)
	exemplar.SetValue(15)
	initMetricAttachment(exemplar.Attachments())
	hdp1.SetExplicitBounds([]float64{1})
}

func generateOtlpCumulativeHistogramMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: generateOtlpMetricDescriptor(testdata.TestDoubleHistogramMetricName, dataold.MetricTypeHistogram),
		HistogramDataPoints: []*otlpmetrics.HistogramDataPoint{
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
				Buckets: []*otlpmetrics.HistogramDataPoint_Bucket{
					{
						Count: 0,
					},
					{
						Count: 1,
						Exemplar: &otlpmetrics.HistogramDataPoint_Bucket_Exemplar{
							TimeUnixNano: uint64(TestMetricExemplarTimestamp),
							Value:        15,
							Attachments:  generateOtlpMetricAttachment(),
						},
					},
				},
				ExplicitBounds: []float64{1},
			},
		},
	}
}

func initSummaryMetric(sm dataold.Metric) {
	initMetricDescriptor(sm.MetricDescriptor(), TestSummaryMetricName, dataold.MetricTypeSummary)

	sdps := sm.SummaryDataPoints()
	sdps.Resize(2)
	sdp0 := sdps.At(0)
	initMetricLabelValue1(sdp0.LabelsMap())
	sdp0.SetStartTime(TestMetricStartTimestamp)
	sdp0.SetTimestamp(TestMetricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)
	sdp1 := sdps.At(1)
	initMetricLabelValue2(sdp1.LabelsMap())
	sdp1.SetStartTime(TestMetricStartTimestamp)
	sdp1.SetTimestamp(TestMetricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)
	sdp1.ValueAtPercentiles().Resize(1)
	sdp1.ValueAtPercentiles().At(0).SetPercentile(1)
	sdp1.ValueAtPercentiles().At(0).SetValue(15)
}

func generateOtlpSummaryMetric() *otlpmetrics.Metric {
	return &otlpmetrics.Metric{
		MetricDescriptor: generateOtlpMetricDescriptor(TestSummaryMetricName, dataold.MetricTypeSummary),
		SummaryDataPoints: []*otlpmetrics.SummaryDataPoint{
			{
				Labels:            generateOtlpMetricLabelValue1(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
			},
			{
				Labels:            generateOtlpMetricLabelValue2(),
				StartTimeUnixNano: uint64(TestMetricStartTimestamp),
				TimeUnixNano:      uint64(TestMetricTimestamp),
				Count:             1,
				Sum:               15,
				PercentileValues: []*otlpmetrics.SummaryDataPoint_ValueAtPercentile{
					{
						Percentile: 1,
						Value:      15,
					},
				},
			},
		},
	}
}

func initMetricDescriptor(md dataold.MetricDescriptor, name string, ty dataold.MetricType) {
	md.InitEmpty()
	md.SetName(name)
	md.SetDescription("")
	md.SetUnit("1")
	md.SetType(ty)
}

func generateOtlpMetricDescriptor(name string, ty dataold.MetricType) *otlpmetrics.MetricDescriptor {
	return &otlpmetrics.MetricDescriptor{
		Name:        name,
		Description: "",
		Unit:        "1",
		Type:        otlpmetrics.MetricDescriptor_Type(ty),
	}
}

func GenerateMetricDataManyMetricsSameResource(metricsCount int) dataold.MetricData {
	md := GenerateMetricDataOneEmptyInstrumentationLibrary()
	rs0ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rs0ilm0.Metrics().Resize(metricsCount)
	for i := 0; i < metricsCount; i++ {
		initCounterIntMetric(rs0ilm0.Metrics().At(i))
	}
	return md
}
