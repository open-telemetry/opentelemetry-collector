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

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestInvalidDataType(t *testing.T) {
	a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)
	metric := pdata.NewMetric()
	metric.SetDataType(-100)
	n := a.addMetric(metric, pdata.NewInstrumentationLibrary())
	require.Zero(t, n)
}

func TestAccumulateDeltaAggregation(t *testing.T) {
	tests := []struct {
		name   string
		metric func(time.Time) pdata.Metric
	}{
		{
			name: "IntSum",
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			},
		},
		{
			name: "DoubleSum",
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			},
		},
		{
			name: "IntHistogram",
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewIntHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().DataPoints().Append(dp)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			},
		},
		{
			name: "DoubleHistogram",
			metric: func(ts time.Time) (metric pdata.Metric) {
				dp := pdata.NewDoubleHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
				metric.DoubleHistogram().DataPoints().Append(dp)
				metric.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				metric.SetDescription("test description")

				return
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.metric(time.Now())

			ilm := pdata.NewInstrumentationLibraryMetrics()
			ilm.InstrumentationLibrary().SetName("test")

			resourceMetrics := pdata.NewResourceMetrics()
			resourceMetrics.InstrumentationLibraryMetrics().Append(ilm)
			ilm.Metrics().Append(m)

			a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)
			n := a.Accumulate(resourceMetrics)
			require.Equal(t, 0, n)

			signature := timeseriesSignature(ilm.InstrumentationLibrary().Name(), m, pdata.NewStringMap())
			v, ok := a.registeredMetrics.Load(signature)
			require.False(t, ok)
			require.Nil(t, v)
		})
	}
}

func TestAccumulateMetrics(t *testing.T) {
	tests := []struct {
		name   string
		metric func(time.Time, float64) pdata.Metric
	}{
		{
			name: "IntGauge",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntGauge)
				metric.IntGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "DoubleGauge",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metric.DoubleGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "IntSum",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(false)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "DoubleSum",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(false)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "MonotonicIntSum",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(true)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "MonotonicDoubleSum",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(true)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "IntHistogram",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewIntHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(int64(v))
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().DataPoints().Append(dp)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
		{
			name: "DoubleHistogram",
			metric: func(ts time.Time, v float64) (metric pdata.Metric) {
				dp := pdata.NewDoubleHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(v)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimestampFromTime(ts))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
				metric.DoubleHistogram().DataPoints().Append(dp)
				metric.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ts1 := time.Now().Add(-3 * time.Second)
			ts2 := time.Now().Add(-2 * time.Second)
			ts3 := time.Now().Add(-1 * time.Second)
			m1 := tt.metric(ts1, 13)
			m2 := tt.metric(ts2, 21)
			m3 := tt.metric(ts3, 34)

			ilm := pdata.NewInstrumentationLibraryMetrics()
			ilm.InstrumentationLibrary().SetName("test")

			resourceMetrics := pdata.NewResourceMetrics()
			resourceMetrics.InstrumentationLibraryMetrics().Append(ilm)
			ilm.Metrics().Append(m2)
			ilm.Metrics().Append(m1)

			a := newAccumulator(zap.NewNop(), 1*time.Hour).(*lastValueAccumulator)

			// 2 metric arrived
			n := a.Accumulate(resourceMetrics)
			require.Equal(t, 1, n)

			m2Labels, _, m2Value, m2Temporality, m2IsMonotonic := getMerticProperties(m2)

			signature := timeseriesSignature(ilm.InstrumentationLibrary().Name(), m2, m2Labels)
			m, ok := a.registeredMetrics.Load(signature)
			require.True(t, ok)

			v := m.(*accumulatedValue)
			vLabels, vTS, vValue, vTemporality, vIsMonotonic := getMerticProperties(m2)

			require.Equal(t, v.instrumentationLibrary.Name(), ilm.InstrumentationLibrary().Name())
			require.Equal(t, v.value.DataType(), m2.DataType())
			vLabels.ForEach(func(k, v string) {
				r, _ := m2Labels.Get(k)
				require.Equal(t, r, v)
			})
			require.Equal(t, m2Labels.Len(), vLabels.Len())
			require.Equal(t, m2Value, vValue)
			require.Equal(t, ts2.Unix(), vTS.Unix())
			require.Greater(t, v.stored.Unix(), vTS.Unix())
			require.Equal(t, m2Temporality, vTemporality)
			require.Equal(t, m2IsMonotonic, vIsMonotonic)

			// 3 metrics arrived
			ilm.Metrics().Resize(0)
			ilm.Metrics().Append(m2)
			ilm.Metrics().Append(m3)
			ilm.Metrics().Append(m1)

			_, _, m3Value, _, _ := getMerticProperties(m3)

			n = a.Accumulate(resourceMetrics)
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
	case pdata.MetricDataTypeDoubleHistogram:
		labels = metric.DoubleHistogram().DataPoints().At(0).LabelsMap()
		ts = metric.DoubleHistogram().DataPoints().At(0).Timestamp().AsTime()
		value = metric.DoubleHistogram().DataPoints().At(0).Sum()
		temporality = metric.DoubleHistogram().AggregationTemporality()
		isMonotonic = true
	default:
		log.Panicf("Invalid data type %s", metric.DataType().String())
	}

	return
}
