// Copyright 2020 OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testdata

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

var (
	TestMetricStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestMetricStartTimestamp = data.TimestampUnixNano(TestMetricStartTime.UnixNano())

	TestMetricExemplarTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestMetricExemplarTimestamp = data.TimestampUnixNano(TestMetricExemplarTime.UnixNano())

	TestMetricTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestMetricTimestamp = data.TimestampUnixNano(TestMetricTime.UnixNano())
)

const (
	TestGaugeDoubleMetricName         = "gauge-double"
	TestGaugeIntMetricName            = "gauge-int"
	TestCounterDoubleMetricName       = "counter-double"
	TestCounterIntMetricName          = "counter-int"
	TestGaugeHistogramMetricName      = "gauge-histogram"
	TestCumulativeHistogramMetricName = "cumulative-histogram"
	TestSummaryMetricName             = "summary"
)

func GenerateMetricDataOneEmptyResourceMetrics() data.MetricData {
	md := data.NewMetricData()
	md.ResourceMetrics().Resize(1)
	return md
}

func GenerateMetricDataNoLibraries() data.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().At(0)
	initResource1(ms0.Resource())
	return md
}

func GenerateMetricDataNoMetrics() data.MetricData {
	md := GenerateMetricDataNoLibraries()
	md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().Resize(1)
	return md
}

func GenerateMetricDataAllTypesNoDataPoints() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	ilm0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	ms := ilm0.Metrics()
	ms.Resize(7)
	initMetricDescriptor(
		ms.At(0).MetricDescriptor(), TestGaugeDoubleMetricName, data.MetricTypeGaugeDouble)
	initMetricDescriptor(
		ms.At(1).MetricDescriptor(), TestGaugeIntMetricName, data.MetricTypeGaugeInt64)
	initMetricDescriptor(
		ms.At(2).MetricDescriptor(), TestCounterDoubleMetricName, data.MetricTypeCounterDouble)
	initMetricDescriptor(
		ms.At(3).MetricDescriptor(), TestCounterIntMetricName, data.MetricTypeCounterInt64)
	initMetricDescriptor(
		ms.At(4).MetricDescriptor(), TestGaugeHistogramMetricName, data.MetricTypeGaugeHistogram)
	initMetricDescriptor(
		ms.At(5).MetricDescriptor(), TestCumulativeHistogramMetricName, data.MetricTypeCumulativeHistogram)
	initMetricDescriptor(
		ms.At(6).MetricDescriptor(), TestSummaryMetricName, data.MetricTypeSummary)
	return md
}

func GenerateMetricDataWithCountersHistogramAndSummary() data.MetricData {
	metricData := data.NewMetricData()
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

func GenerateMetricDataOneMetricNoResource() data.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().At(0)
	rm0.InstrumentationLibraryMetrics().Resize(1)
	rm0ils0 := rm0.InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func GenerateMetricDataOneMetric() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(1)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	return md
}

func GenerateMetricDataTwoMetrics() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	rm0ils0 := md.ResourceMetrics().At(0).InstrumentationLibraryMetrics().At(0)
	rm0ils0.Metrics().Resize(2)
	initCounterIntMetric(rm0ils0.Metrics().At(0))
	initCounterDoubleMetric(rm0ils0.Metrics().At(1))
	return md
}

func initCounterIntMetric(im data.Metric) {
	initMetricDescriptor(im.MetricDescriptor(), TestCounterIntMetricName, data.MetricTypeCounterInt64)

	idps := im.Int64DataPoints()
	idps.Resize(2)
	idp0 := idps.At(0)
	idp0.LabelsMap().InitFromMap(map[string]string{"int-label-1": "int-label-value-1"})
	idp0.SetStartTime(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
	idp1 := idps.At(1)
	idp1.LabelsMap().InitFromMap(map[string]string{"int-label-2": "int-label-value-2"})
	idp1.SetStartTime(TestMetricStartTimestamp)
	idp1.SetTimestamp(TestMetricTimestamp)
	idp1.SetValue(456)
}

func initCounterDoubleMetric(dm data.Metric) {
	initMetricDescriptor(dm.MetricDescriptor(), TestCounterDoubleMetricName, data.MetricTypeCounterDouble)

	ddps := dm.DoubleDataPoints()
	ddps.Resize(1)
	ddp0 := ddps.At(0)
	ddp0.SetStartTime(TestMetricStartTimestamp)
	ddp0.SetTimestamp(TestMetricTimestamp)
	ddp0.SetValue(1.23)
}

func initCumulativeHistogramMetric(hm data.Metric) {
	initMetricDescriptor(hm.MetricDescriptor(), TestCumulativeHistogramMetricName, data.MetricTypeCumulativeHistogram)

	hdps := hm.HistogramDataPoints()
	hdps.Resize(2)
	hdp0 := hdps.At(0)
	hdp0.LabelsMap().InitFromMap(map[string]string{
		"histogram-label-1": "histogram-label-value-1",
		"histogram-label-3": "histogram-label-value-3",
	})
	hdp0.SetStartTime(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hdps.At(1)
	hdp1.LabelsMap().InitFromMap(map[string]string{
		"histogram-label-2": "histogram-label-value-2",
	})
	hdp1.SetStartTime(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.Buckets().Resize(2)
	hdp1.Buckets().At(0).SetCount(0)
	hdp1.Buckets().At(1).SetCount(1)
	hdp1.Buckets().At(1).Exemplar().InitEmpty()
	hdp1.Buckets().At(1).Exemplar().SetTimestamp(TestMetricExemplarTimestamp)
	hdp1.Buckets().At(1).Exemplar().SetValue(15)
	hdp1.Buckets().At(1).Exemplar().Attachments().InitFromMap(map[string]string{
		"exemplar-attachment": "exemplar-attachment-value",
	})
	hdp1.SetExplicitBounds([]float64{1})
}

func initSummaryMetric(sm data.Metric) {
	initMetricDescriptor(sm.MetricDescriptor(), TestSummaryMetricName, data.MetricTypeSummary)

	sdps := sm.SummaryDataPoints()
	sdps.Resize(2)
	sdp0 := sdps.At(0)
	sdp0.LabelsMap().InitFromMap(map[string]string{
		"summary-label": "summary-label-value-1",
	})
	sdp0.SetStartTime(TestMetricStartTimestamp)
	sdp0.SetTimestamp(TestMetricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)
	sdp1 := sdps.At(1)
	sdp1.LabelsMap().InitFromMap(map[string]string{
		"summary-label": "summary-label-value-2",
	})
	sdp1.SetStartTime(TestMetricStartTimestamp)
	sdp1.SetTimestamp(TestMetricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)
	sdp1.ValueAtPercentiles().Resize(1)
	sdp1.ValueAtPercentiles().At(0).SetPercentile(1)
	sdp1.ValueAtPercentiles().At(0).SetValue(15)
}

func initMetricDescriptor(md data.MetricDescriptor, name string, ty data.MetricType) {
	md.InitEmpty()
	md.SetName(name)
	md.SetDescription("")
	md.SetUnit("1")
	md.SetType(ty)
}
