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
	"bytes"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/consumer/pdata"
)

type metricHolder struct {
	desc         *prometheus.Desc
	metricValues map[string]*metricValue
}

type metricValue struct {
	labelValues []string
	metricType  prometheus.ValueType
	timestamp   time.Time
	updated     time.Time
	isHistogram bool

	value float64

	histogramPoints map[float64]uint64
	histogramSum    float64
	histogramCount  uint64
}

type collector struct {
	config            *Config
	mu                sync.Mutex
	registeredMetrics map[string]*metricHolder
	logger            *zap.Logger
}

func newCollector(config *Config, logger *zap.Logger) *collector {
	return &collector{
		config:            config,
		registeredMetrics: make(map[string]*metricHolder),
		logger:            logger,
	}
}

/*
	Processing
*/
func (c *collector) processMetrics(rm pdata.ResourceMetrics) {
	ilms := rm.InstrumentationLibraryMetrics()

	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)

		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			metric := metrics.At(j)
			lk := collectLabelKeys(metric)

			signature := metricSignature(c.config.Namespace, metric, lk.keys)

			c.mu.Lock()
			holder, ok := c.registeredMetrics[signature]
			if !ok {
				holder = &metricHolder{
					desc: prometheus.NewDesc(
						metricName(c.config.Namespace, metric),
						metric.Description(),
						lk.keys,
						c.config.ConstLabels,
					),
					metricValues: make(map[string]*metricValue, 1),
				}
				c.registeredMetrics[signature] = holder
				c.logger.Debug(fmt.Sprintf("new metric: %s", holder.desc.String()))
			}

			c.accumulateMetrics(metric, lk)
			c.logger.Debug(fmt.Sprintf("metric accumulated: %s", holder.desc.String()))
			c.mu.Unlock()
		}
	}
}

func (c *collector) accumulateMetrics(metric pdata.Metric, lk *labelKeys) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		c.accumulateIntGauge(metric, lk)
	case pdata.MetricDataTypeIntSum:
		c.accumulateIntSum(metric, lk)
	case pdata.MetricDataTypeDoubleGauge:
		c.accumulateDoubleGauge(metric, lk)
	case pdata.MetricDataTypeDoubleSum:
		c.accumulateDoubleSum(metric, lk)
	case pdata.MetricDataTypeIntHistogram:
		c.accumulateIntHistogram(metric, lk)
	case pdata.MetricDataTypeDoubleHistogram:
		c.accumulateDoubleHistogram(metric, lk)
	}
}

func (c *collector) accumulateIntGauge(metric pdata.Metric, lk *labelKeys) {
	dps := metric.IntGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}
		v.value = float64(ip.Value())
		v.timestamp = ts
		v.updated = time.Now()
	}
}

func (c *collector) accumulateDoubleGauge(metric pdata.Metric, lk *labelKeys) {
	dps := metric.DoubleGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}
		v.value = ip.Value()
		v.timestamp = ts
		v.updated = time.Now()
	}
}

func (c *collector) accumulateIntSum(metric pdata.Metric, lk *labelKeys) {
	m := metric.IntSum()

	// Drop metrics with non-cumulative aggregations
	if m.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}

		if m.IsMonotonic() {
			v.metricType = prometheus.CounterValue
		} else {
			v.metricType = prometheus.GaugeValue
		}

		v.value = float64(ip.Value())
		v.timestamp = ts

		v.updated = time.Now()
	}
}

func (c *collector) accumulateDoubleSum(metric pdata.Metric, lk *labelKeys) {
	m := metric.DoubleSum()

	// Drop metrics with non-cumulative aggregations
	if m.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}

		if m.IsMonotonic() {
			v.metricType = prometheus.CounterValue
		} else {
			v.metricType = prometheus.GaugeValue
		}

		v.value = ip.Value()
		v.timestamp = ts
		v.updated = time.Now()
	}
}

func (c *collector) accumulateIntHistogram(metric pdata.Metric, lk *labelKeys) {
	m := metric.IntHistogram()

	// Drop metrics with non-cumulative aggregations
	if m.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		indicesMap := make(map[float64]int)
		buckets := make([]float64, 0, len(ip.BucketCounts()))
		for index, bucket := range ip.ExplicitBounds() {
			if _, added := indicesMap[bucket]; !added {
				indicesMap[bucket] = index
				buckets = append(buckets, bucket)
			}
		}
		sort.Float64s(buckets)

		cumCount := uint64(0)

		points := make(map[float64]uint64)
		for _, bucket := range buckets {
			index := indicesMap[bucket]
			var countPerBucket uint64
			if len(ip.ExplicitBounds()) > 0 && index < len(ip.ExplicitBounds()) {
				countPerBucket = ip.BucketCounts()[index]
			}
			cumCount += countPerBucket
			points[bucket] = cumCount
		}

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, isHistogram: true}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}

		v.histogramPoints = points
		v.histogramSum = float64(ip.Sum())
		v.histogramCount = ip.Count()
		v.timestamp = ts

		v.updated = time.Now()
	}
}

func (c *collector) accumulateDoubleHistogram(metric pdata.Metric, lk *labelKeys) {
	m := metric.DoubleHistogram()

	// Drop metrics with non-cumulative aggregations
	if m.AggregationTemporality() != pdata.AggregationTemporalityCumulative {
		return
	}

	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, labelValues)

		indicesMap := make(map[float64]int)
		buckets := make([]float64, 0, len(ip.BucketCounts()))
		for index, bucket := range ip.ExplicitBounds() {
			if _, added := indicesMap[bucket]; !added {
				indicesMap[bucket] = index
				buckets = append(buckets, bucket)
			}
		}
		sort.Float64s(buckets)

		cumCount := uint64(0)

		points := make(map[float64]uint64)
		for _, bucket := range buckets {
			index := indicesMap[bucket]
			var countPerBucket uint64
			if len(ip.ExplicitBounds()) > 0 && index < len(ip.ExplicitBounds()) {
				countPerBucket = ip.BucketCounts()[index]
			}
			cumCount += countPerBucket
			points[bucket] = cumCount
		}

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{value: 0, labelValues: labelValues, isHistogram: true}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		} else if v.timestamp.Sub(ts) > 0 {
			continue
		}

		v.histogramPoints = points
		v.histogramSum = ip.Sum()
		v.histogramCount = ip.Count()
		v.timestamp = ts

		v.updated = time.Now()
	}
}

/*
	Reporting
*/
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debug("collect called")

	size := 0
	for _, metric := range c.registeredMetrics {
		size += len(metric.metricValues)
	}
	metrics := make([]prometheus.Metric, 0, size)

	c.mu.Lock()

	for signature, metric := range c.registeredMetrics {
		for k, v := range metric.metricValues {
			var m prometheus.Metric
			var err error
			if v.isHistogram {
				m, err = prometheus.NewConstHistogram(metric.desc, v.histogramCount, v.histogramSum, v.histogramPoints, v.labelValues...)
			} else {
				m, err = prometheus.NewConstMetric(metric.desc, v.metricType, v.value, v.labelValues...)
			}
			if err == nil {
				if !c.config.SendTimestamps {
					metrics = append(metrics, m)
				} else {
					metrics = append(metrics, prometheus.NewMetricWithTimestamp(v.timestamp, m))
				}
			}

			if time.Now().UnixNano()-v.updated.UnixNano() > c.config.MetricExpiration.Nanoseconds() {
				c.logger.Debug(fmt.Sprintf("value expired: %s", m.Desc().String()))
				delete(metric.metricValues, k)
			}
		}

		if len(metric.metricValues) == 0 {
			c.logger.Debug(fmt.Sprintf("metric expired: %s", metric.desc.String()))
			delete(c.registeredMetrics, signature)
		}
	}

	c.mu.Unlock()

	for _, m := range metrics {
		ch <- m
		c.logger.Debug(fmt.Sprintf("metric served: %s", m.Desc().String()))
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	registered := make(map[string]*prometheus.Desc)

	c.mu.Lock()

	for key, holder := range c.registeredMetrics {
		registered[key] = holder.desc
	}

	c.mu.Unlock()

	for _, desc := range registered {
		ch <- desc
	}
}

/*
	Helpers
*/

func metricSignature(namespace string, metric pdata.Metric, keys []string) string {
	var buf bytes.Buffer
	buf.WriteString(metricName(namespace, metric))
	for _, k := range keys {
		buf.WriteString("-" + k)
	}
	return buf.String()
}

func metricName(namespace string, metric pdata.Metric) string {
	var name string
	if namespace != "" {
		name = namespace + "_"
	}
	mName := metric.Name()
	return name + sanitize(mName)
}

type labelKeys struct {
	// ordered OC label keys
	keys []string
	// map from a label key literal
	// to its index in the slice above
	keyIndices map[string]int
}

func collectLabelKeys(metric pdata.Metric) *labelKeys {
	// First, collect a set of all labels present in the metric
	keySet := make(map[string]struct{})

	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		collectLabelKeysIntDataPoints(metric.IntGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleGauge:
		collectLabelKeysDoubleDataPoints(metric.DoubleGauge().DataPoints(), keySet)
	case pdata.MetricDataTypeIntSum:
		collectLabelKeysIntDataPoints(metric.IntSum().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleSum:
		collectLabelKeysDoubleDataPoints(metric.DoubleSum().DataPoints(), keySet)
	case pdata.MetricDataTypeIntHistogram:
		collectLabelKeysIntHistogramDataPoints(metric.IntHistogram().DataPoints(), keySet)
	case pdata.MetricDataTypeDoubleHistogram:
		collectLabelKeysDoubleHistogramDataPoints(metric.DoubleHistogram().DataPoints(), keySet)
	}

	if len(keySet) == 0 {
		return &labelKeys{}
	}

	// Sort keys
	sortedKeys := make([]string, 0, len(keySet))
	for key := range keySet {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	keys := make([]string, 0, len(sortedKeys))
	// Label values will have to match keys by index
	// so this map will help with fast lookups.
	indices := make(map[string]int, len(sortedKeys))
	for i, key := range sortedKeys {
		keys = append(keys, key)
		indices[key] = i
	}

	return &labelKeys{
		keys:       keys,
		keyIndices: indices,
	}
}

func collectLabelKeysIntDataPoints(ips pdata.IntDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ips.Len(); i++ {
		ip := ips.At(i)
		addLabelKeys(keySet, ip.LabelsMap())
	}
}

func collectLabelKeysDoubleDataPoints(dps pdata.DoubleDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		addLabelKeys(keySet, dp.LabelsMap())
	}
}

func collectLabelKeysIntHistogramDataPoints(ihdp pdata.IntHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ihdp.Len(); i++ {
		hp := ihdp.At(i)
		addLabelKeys(keySet, hp.LabelsMap())
	}
}

func collectLabelKeysDoubleHistogramDataPoints(dhdp pdata.DoubleHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		hp := dhdp.At(i)
		addLabelKeys(keySet, hp.LabelsMap())
	}
}

func addLabelKeys(keySet map[string]struct{}, labels pdata.StringMap) {
	labels.ForEach(func(k string, v string) {
		keySet[k] = struct{}{}
	})
}

func collectLabelValues(labels pdata.StringMap, lk *labelKeys) []string {
	if len(lk.keys) == 0 {
		return nil
	}

	labelValues := make([]string, len(lk.keys))
	labels.ForEach(func(k string, v string) {
		keyIndex := lk.keyIndices[k]
		labelValues[keyIndex] = v
	})

	return labelValues
}
