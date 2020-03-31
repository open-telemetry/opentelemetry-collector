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
	md.SetResourceMetrics(data.NewResourceMetricsSlice(1))
	return md
}

func GenerateMetricDataNoLibraries() data.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	ms0 := md.ResourceMetrics().Get(0)
	ms0.InitResourceIfNil()
	fillResource1(ms0.Resource())
	return md
}

func GenerateMetricDataNoMetrics() data.MetricData {
	md := GenerateMetricDataNoLibraries()
	md.ResourceMetrics().Get(0).SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	return md
}

func GenerateMetricDataAllTypesNoDataPoints() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	ilm0 := md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0)
	ilm0.SetMetrics(data.NewMetricSlice(7))
	ms := ilm0.Metrics()
	ms.Get(0).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(0).MetricDescriptor(), TestGaugeDoubleMetricName, data.MetricTypeGaugeDouble)
	ms.Get(1).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(1).MetricDescriptor(), TestGaugeIntMetricName, data.MetricTypeGaugeInt64)
	ms.Get(2).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(2).MetricDescriptor(), TestCounterDoubleMetricName, data.MetricTypeCounterDouble)
	ms.Get(3).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(3).MetricDescriptor(), TestCounterIntMetricName, data.MetricTypeCounterInt64)
	ms.Get(4).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(4).MetricDescriptor(), TestGaugeHistogramMetricName, data.MetricTypeGaugeHistogram)
	ms.Get(5).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(5).MetricDescriptor(), TestCumulativeHistogramMetricName, data.MetricTypeCumulativeHistogram)
	ms.Get(6).InitMetricDescriptorIfNil()
	fillMetricDescriptor(
		ms.Get(6).MetricDescriptor(), TestSummaryMetricName, data.MetricTypeSummary)
	return md
}

func GenerateMetricDataWithCountersHistogramAndSummary() data.MetricData {
	metricData := data.NewMetricData()
	metricData.SetResourceMetrics(data.NewResourceMetricsSlice(1))

	rms := metricData.ResourceMetrics()
	rms.Get(0).SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	rms.Get(0).InitResourceIfNil()
	fillResource1(rms.Get(0).Resource())

	ilms := rms.Get(0).InstrumentationLibraryMetrics()
	ilms.Get(0).SetMetrics(data.NewMetricSlice(4))
	ms := ilms.Get(0).Metrics()
	fillCounterIntMetric(ms.Get(0))
	fillCounterDoubleMetric(ms.Get(1))
	fillCumulativeHistogramMetric(ms.Get(2))
	fillSummaryMetric(ms.Get(3))

	return metricData
}

func GenerateMetricDataOneMetricNoResource() data.MetricData {
	md := GenerateMetricDataOneEmptyResourceMetrics()
	rm0 := md.ResourceMetrics().Get(0)
	rm0.SetInstrumentationLibraryMetrics(data.NewInstrumentationLibraryMetricsSlice(1))
	rm0ils0 := rm0.InstrumentationLibraryMetrics().Get(0)
	rm0ils0.SetMetrics(data.NewMetricSlice(1))
	fillCounterIntMetric(rm0ils0.Metrics().Get(0))
	return md
}

func GenerateMetricDataOneMetric() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	rm0ils0 := md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0)
	rm0ils0.SetMetrics(data.NewMetricSlice(1))
	fillCounterIntMetric(rm0ils0.Metrics().Get(0))
	return md
}

func GenerateMetricDataTwoMetrics() data.MetricData {
	md := GenerateMetricDataNoMetrics()
	rm0ils0 := md.ResourceMetrics().Get(0).InstrumentationLibraryMetrics().Get(0)
	rm0ils0.SetMetrics(data.NewMetricSlice(2))
	fillCounterIntMetric(rm0ils0.Metrics().Get(0))
	fillCounterDoubleMetric(rm0ils0.Metrics().Get(1))
	return md
}

func fillCounterIntMetric(im data.Metric) {
	im.InitMetricDescriptorIfNil()
	fillMetricDescriptor(im.MetricDescriptor(), TestCounterIntMetricName, data.MetricTypeCounterInt64)

	im.SetInt64DataPoints(data.NewInt64DataPointSlice(2))
	idp0 := im.Int64DataPoints().Get(0)
	idp0.SetLabelsMap(data.NewStringMap(map[string]string{"int-label-1": "int-label-value-1"}))
	idp0.SetStartTime(TestMetricStartTimestamp)
	idp0.SetTimestamp(TestMetricTimestamp)
	idp0.SetValue(123)
	idp1 := im.Int64DataPoints().Get(1)
	idp1.SetLabelsMap(data.NewStringMap(map[string]string{"int-label-2": "int-label-value-2"}))
	idp1.SetStartTime(TestMetricStartTimestamp)
	idp1.SetTimestamp(TestMetricTimestamp)
	idp1.SetValue(456)
}

func fillCounterDoubleMetric(dm data.Metric) {
	dm.InitMetricDescriptorIfNil()
	fillMetricDescriptor(dm.MetricDescriptor(), TestCounterDoubleMetricName, data.MetricTypeCounterDouble)

	dm.SetDoubleDataPoints(data.NewDoubleDataPointSlice(1))
	ddp0 := dm.DoubleDataPoints().Get(0)
	ddp0.SetStartTime(TestMetricStartTimestamp)
	ddp0.SetTimestamp(TestMetricTimestamp)
	ddp0.SetValue(1.23)
}

func fillCumulativeHistogramMetric(hm data.Metric) {
	hm.InitMetricDescriptorIfNil()
	fillMetricDescriptor(hm.MetricDescriptor(), TestCumulativeHistogramMetricName, data.MetricTypeCumulativeHistogram)

	hm.SetHistogramDataPoints(data.NewHistogramDataPointSlice(2))
	hdp0 := hm.HistogramDataPoints().Get(0)
	hdp0.SetLabelsMap(data.NewStringMap(map[string]string{
		"histogram-label-1": "histogram-label-value-1",
		"histogram-label-3": "histogram-label-value-3",
	}))
	hdp0.SetStartTime(TestMetricStartTimestamp)
	hdp0.SetTimestamp(TestMetricTimestamp)
	hdp0.SetCount(1)
	hdp0.SetSum(15)
	hdp1 := hm.HistogramDataPoints().Get(1)
	hdp1.SetLabelsMap(data.NewStringMap(map[string]string{
		"histogram-label-2": "histogram-label-value-2",
	}))
	hdp1.SetStartTime(TestMetricStartTimestamp)
	hdp1.SetTimestamp(TestMetricTimestamp)
	hdp1.SetCount(1)
	hdp1.SetSum(15)
	hdp1.SetBuckets(data.NewHistogramBucketSlice(2))
	hdp1.Buckets().Get(0).SetCount(0)
	hdp1.Buckets().Get(1).SetCount(1)
	hdp1.Buckets().Get(1).InitExemplarIfNil()
	hdp1.Buckets().Get(1).Exemplar().SetTimestamp(TestMetricExemplarTimestamp)
	hdp1.Buckets().Get(1).Exemplar().SetValue(15)
	hdp1.Buckets().Get(1).Exemplar().SetAttachments(data.NewStringMap(map[string]string{
		"exemplar-attachment": "exemplar-attachment-value",
	}))
	hdp1.SetExplicitBounds([]float64{1})
}

func fillSummaryMetric(sm data.Metric) {
	sm.InitMetricDescriptorIfNil()
	fillMetricDescriptor(sm.MetricDescriptor(), TestSummaryMetricName, data.MetricTypeSummary)

	sm.SetSummaryDataPoints(data.NewSummaryDataPointSlice(2))
	sdps := sm.SummaryDataPoints()
	sdp0 := sdps.Get(0)
	sdp0.SetLabelsMap(data.NewStringMap(map[string]string{
		"summary-label": "summary-label-value-1",
	}))
	sdp0.SetStartTime(TestMetricStartTimestamp)
	sdp0.SetTimestamp(TestMetricTimestamp)
	sdp0.SetCount(1)
	sdp0.SetSum(15)
	sdp1 := sdps.Get(1)
	sdp1.SetLabelsMap(data.NewStringMap(map[string]string{
		"summary-label": "summary-label-value-2",
	}))
	sdp1.SetStartTime(TestMetricStartTimestamp)
	sdp1.SetTimestamp(TestMetricTimestamp)
	sdp1.SetCount(1)
	sdp1.SetSum(15)
	sdp1.SetValueAtPercentiles(data.NewSummaryValueAtPercentileSlice(1))
	sdp1.ValueAtPercentiles().Get(0).SetPercentile(1)
	sdp1.ValueAtPercentiles().Get(0).SetValue(15)
}

func fillMetricDescriptor(md data.MetricDescriptor, name string, ty data.MetricType) {
	md.SetName(name)
	md.SetDescription("")
	md.SetUnit("1")
	md.SetType(ty)
}
