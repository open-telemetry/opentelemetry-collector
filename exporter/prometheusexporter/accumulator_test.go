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

package prometheusexporter

import (
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

func TestInvalidDataType(t *testing.T) {
	a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)
	metric := pdata.NewMetric()
	metric.SetDataType(-100)
	n := a.addMetric(metric, pdata.NewInstrumentationLibrary(), time.Now())
	require.Zero(t, n)
}

func TestAccumulateDeltaAggregation(t *testing.T) {
	tests := []struct {
		name       string
		fillMetric func(time.Time, pdata.Metric)
	}{
		{
			name: "IntSum",
			fillMetric: func(ts time.Time, metric pdata.Metric) {
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				dp := metric.IntSum().DataPoints().AppendEmpty()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "DoubleSum",
			fillMetric: func(ts time.Time, metric pdata.Metric) {
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				dp := metric.DoubleSum().DataPoints().AppendEmpty()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "IntHistogram",
			fillMetric: func(ts time.Time, metric pdata.Metric) {
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				dp := metric.IntHistogram().DataPoints().AppendEmpty()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "Histogram",
			fillMetric: func(ts time.Time, metric pdata.Metric) {
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeHistogram)
				metric.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				metric.SetDescription("test description")
				dp := metric.Histogram().DataPoints().AppendEmpty()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceMetrics := pdata.NewResourceMetrics()
			ilm := resourceMetrics.InstrumentationLibraryMetrics().AppendEmpty()
			ilm.InstrumentationLibrary().SetName("test")
			tt.fillMetric(time.Now(), ilm.Metrics().AppendEmpty())

			a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)
			n := a.Accumulate(resourceMetrics)
			require.Equal(t, 0, n)

			signature := timeseriesSignature(ilm.InstrumentationLibrary().Name(), ilm.Metrics().At(0), pdata.NewStringMap())
			v, ok := a.registeredMetrics.Load(signature)
			require.False(t, ok)
			require.Nil(t, v)
		})
	}
}

func TestAccumulateMetrics(t *testing.T) {
	tests := []struct {
		name   string
		metric func(time.Time, float64, pdata.MetricSlice)
	}{
		{
			name: "IntGauge",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntGauge)
				metric.SetDescription("test description")
				dp := metric.IntGauge().DataPoints().AppendEmpty()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "DoubleGauge",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metric.SetDescription("test description")
				dp := metric.DoubleGauge().DataPoints().AppendEmpty()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "IntSum",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().SetIsMonotonic(false)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.IntSum().DataPoints().AppendEmpty()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "DoubleSum",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().SetIsMonotonic(false)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.DoubleSum().DataPoints().AppendEmpty()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "MonotonicIntSum",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().SetIsMonotonic(true)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.IntSum().DataPoints().AppendEmpty()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "MonotonicDoubleSum",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().SetIsMonotonic(true)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.DoubleSum().DataPoints().AppendEmpty()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "IntHistogram",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.IntHistogram().DataPoints().AppendEmpty()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
		{
			name: "Histogram",
			metric: func(ts time.Time, v float64, metrics pdata.MetricSlice) {
				metric := metrics.AppendEmpty()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeHistogram)
				metric.Histogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")
				dp := metric.Histogram().DataPoints().AppendEmpty()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ts1 := time.Now().Add(-3 * time.Second)
			ts2 := time.Now().Add(-2 * time.Second)
			ts3 := time.Now().Add(-1 * time.Second)

			resourceMetrics2 := pdata.NewResourceMetrics()
			ilm2 := resourceMetrics2.InstrumentationLibraryMetrics().AppendEmpty()
			ilm2.InstrumentationLibrary().SetName("test")
			tt.metric(ts2, 21, ilm2.Metrics())
			tt.metric(ts1, 13, ilm2.Metrics())

			a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)

			// 2 metric arrived
			n := a.Accumulate(resourceMetrics2)
			require.Equal(t, 1, n)

			m2Labels, _, m2Value, m2Temporality, m2IsMonotonic := getMerticProperties(ilm2.Metrics().At(0))

			signature := timeseriesSignature(ilm2.InstrumentationLibrary().Name(), ilm2.Metrics().At(0), m2Labels)
			m, ok := a.registeredMetrics.Load(signature)
			require.True(t, ok)

			v := m.(*accumulatedValue)
			vLabels, vTS, vValue, vTemporality, vIsMonotonic := getMerticProperties(ilm2.Metrics().At(0))

			require.Equal(t, v.instrumentationLibrary.Name(), "test")
			require.Equal(t, v.value.DataType(), ilm2.Metrics().At(0).DataType())
			vLabels.Range(func(k, v string) bool {
				r, _ := m2Labels.Get(k)
				require.Equal(t, r, v)
				return true
			})
			require.Equal(t, m2Labels.Len(), vLabels.Len())
			require.Equal(t, m2Value, vValue)
			require.Equal(t, ts2.Unix(), vTS.Unix())
			require.Greater(t, v.updated.Unix(), vTS.Unix())
			require.Equal(t, m2Temporality, vTemporality)
			require.Equal(t, m2IsMonotonic, vIsMonotonic)

			// 3 metrics arrived
			resourceMetrics3 := pdata.NewResourceMetrics()
			ilm3 := resourceMetrics3.InstrumentationLibraryMetrics().AppendEmpty()
			ilm3.InstrumentationLibrary().SetName("test")
			tt.metric(ts2, 21, ilm3.Metrics())
			tt.metric(ts3, 34, ilm3.Metrics())
			tt.metric(ts1, 13, ilm3.Metrics())

			_, _, m3Value, _, _ := getMerticProperties(ilm3.Metrics().At(1))

			n = a.Accumulate(resourceMetrics3)
			require.Equal(t, 2, n)

			m, ok = a.registeredMetrics.Load(signature)
			require.True(t, ok)
			v = m.(*accumulatedValue)
			_, vTS, vValue, _, _ = getMerticProperties(v.value)

			require.Equal(t, m3Value, vValue)
			require.Equal(t, ts3.Unix(), vTS.Unix())
		})
	}
}

func getMerticProperties(metric pdata.Metric) (
	labels pdata.StringMap,
	ts time.Time,
	value float64,
	temporality pdata.AggregationTemporality,
	isMonotonic bool,
) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		labels = metric.IntGauge().DataPoints().At(0).LabelsMap()
		ts = metric.IntGauge().DataPoints().At(0).Timestamp().AsTime()
		value = float64(metric.IntGauge().DataPoints().At(0).Value())
		temporality = pdata.AggregationTemporalityUnspecified
		isMonotonic = false
	case pdata.MetricDataTypeIntSum:
		labels = metric.IntSum().DataPoints().At(0).LabelsMap()
		ts = metric.IntSum().DataPoints().At(0).Timestamp().AsTime()
		value = float64(metric.IntSum().DataPoints().At(0).Value())
		temporality = metric.IntSum().AggregationTemporality()
		isMonotonic = metric.IntSum().IsMonotonic()
	case pdata.MetricDataTypeDoubleGauge:
		labels = metric.DoubleGauge().DataPoints().At(0).LabelsMap()
		ts = metric.DoubleGauge().DataPoints().At(0).Timestamp().AsTime()
		value = metric.DoubleGauge().DataPoints().At(0).Value()
		temporality = pdata.AggregationTemporalityUnspecified
		isMonotonic = false
	case pdata.MetricDataTypeDoubleSum:
		labels = metric.DoubleSum().DataPoints().At(0).LabelsMap()
		ts = metric.DoubleSum().DataPoints().At(0).Timestamp().AsTime()
		value = metric.DoubleSum().DataPoints().At(0).Value()
		temporality = metric.DoubleSum().AggregationTemporality()
		isMonotonic = metric.DoubleSum().IsMonotonic()
	case pdata.MetricDataTypeIntHistogram:
		labels = metric.IntHistogram().DataPoints().At(0).LabelsMap()
		ts = metric.IntHistogram().DataPoints().At(0).Timestamp().AsTime()
		value = float64(metric.IntHistogram().DataPoints().At(0).Sum())
		temporality = metric.IntHistogram().AggregationTemporality()
		isMonotonic = true
	case pdata.MetricDataTypeHistogram:
		labels = metric.Histogram().DataPoints().At(0).LabelsMap()
		ts = metric.Histogram().DataPoints().At(0).Timestamp().AsTime()
		value = metric.Histogram().DataPoints().At(0).Sum()
		temporality = metric.Histogram().AggregationTemporality()
		isMonotonic = true
	default:
		log.Panicf("Invalid data type %s", metric.DataType().String())
	}

	return
}
