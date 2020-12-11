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
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestMetricWithNoLabels(t *testing.T) {
	dp := pdata.NewIntDataPoint()
	dp.SetValue(42)

	metric := pdata.NewMetric()
	metric.SetName("test_metric")
	metric.SetDataType(pdata.MetricDataTypeIntGauge)
	metric.IntGauge().DataPoints().Append(dp)
	metric.SetDescription("test description")

	c := newCollector(&Config{}, zap.NewNop())

	lk := collectLabelKeys(metric)
	signature := metricSignature(c.config.Namespace, metric, lk.keys)
	holder := &metricHolder{
		desc: prometheus.NewDesc(
			metricName(c.config.Namespace, metric),
			metric.Description(),
			lk.keys,
			c.config.ConstLabels,
		),
		metricValues: make(map[string]*metricValue, 1),
	}
	c.registeredMetrics[signature] = holder
	c.accumulateMetrics(metric, lk)

	valueSignature := metricSignature(c.config.Namespace, metric, nil)
	v := holder.metricValues[valueSignature]
	require.NotNil(t, v)

	require.Equal(t, 42.0, v.value)
}

func TestAccumulateDeltaAggregation(t *testing.T) {
	tests := []struct {
		name   string
		metric pdata.Metric
	}{
		{
			name: "IntSum",
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			}(),
		},
		{
			name: "DoubleSum",
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			}(),
		},
		{
			name: "IntHistogram",
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().DataPoints().Append(dp)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)

				return
			}(),
		},
		{
			name: "DoubleHistogram",
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
				metric.DoubleHistogram().DataPoints().Append(dp)
				metric.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityDelta)
				metric.SetDescription("test description")

				return
			}(),
		},
	}
	c := newCollector(&Config{
		Namespace: "test-space",
	}, zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lk := collectLabelKeys(tt.metric)
			signature := metricSignature(c.config.Namespace, tt.metric, lk.keys)
			holder := &metricHolder{
				desc: prometheus.NewDesc(
					metricName(c.config.Namespace, tt.metric),
					tt.metric.Description(),
					lk.keys,
					c.config.ConstLabels,
				),
				metricValues: make(map[string]*metricValue, 1),
			}
			c.registeredMetrics[signature] = holder
			c.accumulateMetrics(tt.metric, lk)

			labelValues := []string{"1", "2"}
			valueSignature := metricSignature(c.config.Namespace, tt.metric, labelValues)
			v := holder.metricValues[valueSignature]

			require.Nil(t, v)
		})
	}
}

func TestAccumulateMetrics(t *testing.T) {
	tests := []struct {
		name       string
		metric     pdata.Metric
		metricType prometheus.ValueType
		value      float64
	}{
		{
			name:       "IntGauge",
			metricType: prometheus.GaugeValue,
			value:      42.0,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntGauge)
				metric.IntGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name:       "DoubleGauge",
			metricType: prometheus.GaugeValue,
			value:      42.42,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleGauge)
				metric.DoubleGauge().DataPoints().Append(dp)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name:       "IntSum",
			metricType: prometheus.GaugeValue,
			value:      42.0,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(false)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name:       "DoubleSum",
			metricType: prometheus.GaugeValue,
			value:      42.42,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(false)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name:       "MonotonicIntSum",
			metricType: prometheus.CounterValue,
			value:      42.0,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntDataPoint()
				dp.SetValue(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntSum)
				metric.IntSum().DataPoints().Append(dp)
				metric.IntSum().SetIsMonotonic(true)
				metric.IntSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name:       "MonotonicDoubleSum",
			metricType: prometheus.CounterValue,
			value:      42.42,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleDataPoint()
				dp.SetValue(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleSum)
				metric.DoubleSum().DataPoints().Append(dp)
				metric.DoubleSum().SetIsMonotonic(true)
				metric.DoubleSum().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
	}
	c := newCollector(&Config{
		Namespace: "test-space",
	}, zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lk := collectLabelKeys(tt.metric)
			signature := metricSignature(c.config.Namespace, tt.metric, lk.keys)
			holder := &metricHolder{
				desc: prometheus.NewDesc(
					metricName(c.config.Namespace, tt.metric),
					tt.metric.Description(),
					lk.keys,
					c.config.ConstLabels,
				),
				metricValues: make(map[string]*metricValue, 1),
			}
			c.registeredMetrics[signature] = holder
			c.accumulateMetrics(tt.metric, lk)

			labelValues := []string{"1", "2"}
			valueSignature := metricSignature(c.config.Namespace, tt.metric, labelValues)
			v := holder.metricValues[valueSignature]

			require.NotNil(t, v)
			require.Equal(t, tt.metricType, v.metricType)
			require.Equal(t, "1", v.labelValues[0])
			require.Equal(t, "2", v.labelValues[1])
			require.False(t, v.isHistogram)
			require.Greater(t, v.timestamp.Unix(), int64(0))
			require.GreaterOrEqual(t, v.updated.UnixNano(), v.timestamp.UnixNano())
			require.Equal(t, tt.value, v.value)
		})
	}
}

func TestAccumulateHistograms(t *testing.T) {
	tests := []struct {
		name   string
		metric pdata.Metric

		histogramPoints map[float64]uint64
		histogramSum    float64
		histogramCount  uint64
	}{
		{
			name: "IntHistogram",
			histogramPoints: map[float64]uint64{
				1.2:  5,
				10.0: 7,
			},
			histogramSum:   42.0,
			histogramCount: 7,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewIntHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{1.2, 10.0})
				dp.SetSum(42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeIntHistogram)
				metric.IntHistogram().DataPoints().Append(dp)
				metric.IntHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
		{
			name: "DoubleHistogram",
			histogramPoints: map[float64]uint64{
				3.5:  5,
				10.0: 7,
			},
			histogramSum:   42.42,
			histogramCount: 7,
			metric: func() (metric pdata.Metric) {
				dp := pdata.NewDoubleHistogramDataPoint()
				dp.SetBucketCounts([]uint64{5, 2})
				dp.SetCount(7)
				dp.SetExplicitBounds([]float64{3.5, 10.0})
				dp.SetSum(42.42)
				dp.LabelsMap().Insert("label_1", "1")
				dp.LabelsMap().Insert("label_2", "2")
				dp.SetTimestamp(pdata.TimeToUnixNano(time.Now()))

				metric = pdata.NewMetric()
				metric.SetName("test_metric")
				metric.SetDataType(pdata.MetricDataTypeDoubleHistogram)
				metric.DoubleHistogram().DataPoints().Append(dp)
				metric.DoubleHistogram().SetAggregationTemporality(pdata.AggregationTemporalityCumulative)
				metric.SetDescription("test description")

				return
			}(),
		},
	}
	c := newCollector(&Config{
		Namespace: "test-space",
	}, zap.NewNop())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lk := collectLabelKeys(tt.metric)
			signature := metricSignature(c.config.Namespace, tt.metric, lk.keys)
			holder := &metricHolder{
				desc: prometheus.NewDesc(
					metricName(c.config.Namespace, tt.metric),
					tt.metric.Description(),
					lk.keys,
					c.config.ConstLabels,
				),
				metricValues: make(map[string]*metricValue, 1),
			}
			c.registeredMetrics[signature] = holder
			c.accumulateMetrics(tt.metric, lk)

			labelValues := []string{"1", "2"}
			valueSignature := metricSignature(c.config.Namespace, tt.metric, labelValues)
			v := holder.metricValues[valueSignature]

			require.NotNil(t, v)
			require.Equal(t, "1", v.labelValues[0])
			require.Equal(t, "2", v.labelValues[1])
			require.True(t, v.isHistogram)
			require.Greater(t, v.timestamp.Unix(), int64(0))
			require.GreaterOrEqual(t, v.updated.UnixNano(), v.timestamp.UnixNano())
			require.Equal(t, tt.histogramCount, v.histogramCount)
			require.Equal(t, tt.histogramSum, v.histogramSum)
			require.Equal(t, len(tt.histogramPoints), len(v.histogramPoints))
			for b, p := range tt.histogramPoints {
				require.Equal(t, p, v.histogramPoints[b])
			}
		})
	}
}
