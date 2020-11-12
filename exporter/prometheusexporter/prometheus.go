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
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type metricValue struct {
	desc        *prometheus.Desc
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

type prometheusExporter struct {
	name         string
	shutdownFunc func() error
	collector    *collector
}

func (pe *prometheusExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pe *prometheusExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	var errstrings []string

	rmetrics := md.ResourceMetrics()
	for i := 0; i < rmetrics.Len(); i++ {
		rs := rmetrics.At(i)
		if rs.IsNil() {
			continue
		}

		if err := pe.collector.processMetrics(rs); err != nil {
			errstrings = append(errstrings, err.Error())
		}
	}

	if len(errstrings) > 0 {
		return fmt.Errorf(strings.Join(errstrings, "; "))
	}
	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (pe *prometheusExporter) Shutdown(context.Context) error {
	return pe.shutdownFunc()
}

// Collector
type collector struct {
	config              *Config
	mu                  sync.Mutex
	registry            *prometheus.Registry
	registerOnce        sync.Once
	registeredMetricsMu sync.Mutex
	registeredMetrics   map[string]*prometheus.Desc
	metricsValues       map[string]*metricValue
	logger              *zap.Logger
}

func (c *collector) ensureRegisteredOnce() error {
	var finalErr error
	c.registerOnce.Do(func() {
		if err := c.registry.Register(c); err != nil {
			finalErr = err
		}
	})
	return finalErr
}

// Process metrics
func (c *collector) processMetrics(rm pdata.ResourceMetrics) error {
	count := 0
	ilms := rm.InstrumentationLibraryMetrics()
	if ilms.Len() == 0 {
		return nil
	}

	for i := 0; i < ilms.Len(); i++ {
		ilm := ilms.At(i)
		if ilm.IsNil() {
			continue
		}

		metrics := ilm.Metrics()
		for j := 0; j < metrics.Len(); j++ {
			metric := metrics.At(j)
			if metric.IsNil() {
				continue
			}

			lk := collectLabelKeys(metric)
			desc, signature, ok := c.lookupPrometheusDesc(c.config.Namespace, metric, lk)

			if !ok {
				desc = prometheus.NewDesc(
					metricName(c.config.Namespace, metric),
					metric.Description(),
					lk.keys,
					c.config.ConstLabels,
				)
				c.registeredMetricsMu.Lock()
				c.registeredMetrics[signature] = desc
				c.registeredMetricsMu.Unlock()
				count++
			}

			c.accumulateMetrics(metric, lk, desc)
		}
	}

	if count == 0 {
		return nil
	}

	return c.ensureRegisteredOnce()
}

func (c *collector) lookupPrometheusDesc(namespace string, metric pdata.Metric, lk *labelKeys) (*prometheus.Desc, string, bool) {
	signature := metricSignature(namespace, metric, lk.keys)
	c.registeredMetricsMu.Lock()
	desc, ok := c.registeredMetrics[signature]
	c.registeredMetricsMu.Unlock()

	return desc, signature, ok
}

func (c *collector) accumulateMetrics(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	switch metric.DataType() {
	case pdata.MetricDataTypeIntGauge:
		c.accumulateIntGauge(metric, lk, desc)
	case pdata.MetricDataTypeIntSum:
		c.accumulateIntSum(metric, lk, desc)
	case pdata.MetricDataTypeDoubleGauge:
		c.accumulateDoubleGauge(metric, lk, desc)
	case pdata.MetricDataTypeDoubleSum:
		c.accumulateDoubleSum(metric, lk, desc)
	case pdata.MetricDataTypeIntHistogram:
		c.accumulateIntHistogram(metric, lk, desc)
	case pdata.MetricDataTypeDoubleHistogram:
		c.accumulateDoubleHistogram(metric, lk, desc)
	}

	c.logger.Debug(fmt.Sprintf("metric accumulated: %s", metric.Name()))
}

func (c *collector) accumulateIntGauge(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	dps := metric.IntGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.metricsValues[signature] = v
		}
		v.value = float64(ip.Value())
		v.timestamp = ts
		v.updated = time.Now()
		c.mu.Unlock()
	}
}

func (c *collector) accumulateDoubleGauge(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	dps := metric.DoubleGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.metricsValues[signature] = v
		}
		v.value = ip.Value()
		v.timestamp = ts
		v.updated = time.Now()
		c.mu.Unlock()
	}
}

func (c *collector) accumulateIntSum(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	m := metric.IntSum()
	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.metricsValues[signature] = v
		}

		if m.IsMonotonic() {
			v.metricType = prometheus.CounterValue
		} else {
			v.metricType = prometheus.GaugeValue
		}

		if m.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			v.value = float64(ip.Value())
			v.timestamp = ts
		} else {
			v.value += float64(ip.Value())
			if v.timestamp.UnixNano() < ts.UnixNano() {
				v.timestamp = ts
			}
		}

		v.updated = time.Now()
		c.mu.Unlock()
	}
}

func (c *collector) accumulateDoubleSum(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	m := metric.DoubleSum()
	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.metricsValues[signature] = v
		}

		if m.IsMonotonic() {
			v.metricType = prometheus.CounterValue
		} else {
			v.metricType = prometheus.GaugeValue
		}

		if m.AggregationTemporality() == pdata.AggregationTemporalityCumulative {
			v.value = ip.Value()
			v.timestamp = ts
		} else {
			v.value += ip.Value()
			if v.timestamp.UnixNano() < ts.UnixNano() {
				v.timestamp = ts
			}
		}
		v.updated = time.Now()
		c.mu.Unlock()
	}
}

func (c *collector) accumulateIntHistogram(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	m := metric.IntHistogram()
	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

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

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, isHistogram: true}
			c.metricsValues[signature] = v
		}
		if m.AggregationTemporality() == pdata.AggregationTemporalityCumulative || v.histogramPoints == nil {
			v.histogramPoints = points
			v.histogramSum = float64(ip.Sum())
			v.histogramCount = ip.Count()
			v.timestamp = ts
		} else {
			for b, c := range points {
				_, ok := v.histogramPoints[b]
				if !ok {
					v.histogramPoints[b] = c
				} else {
					v.histogramPoints[b] += c
				}
			}
			for b := range v.histogramPoints {
				if _, ok := points[b]; !ok {
					delete(v.histogramPoints, b)
				}
			}
			v.histogramSum += float64(ip.Sum())
			v.histogramCount += ip.Count()
			if v.timestamp.UnixNano() < ts.UnixNano() {
				v.timestamp = ts
			}
		}
		v.updated = time.Now()
		c.mu.Unlock()
	}
}

func (c *collector) accumulateDoubleHistogram(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	m := metric.DoubleHistogram()
	dps := m.DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)
		if ip.IsNil() {
			continue
		}

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

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

		c.mu.Lock()
		v, ok := c.metricsValues[signature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, isHistogram: true}
			c.metricsValues[signature] = v
		}
		if m.AggregationTemporality() == pdata.AggregationTemporalityCumulative || v.histogramPoints == nil {
			v.histogramPoints = points
			v.histogramSum = ip.Sum()
			v.histogramCount = ip.Count()
			v.timestamp = ts
		} else {
			for b, c := range points {
				_, ok := v.histogramPoints[b]
				if !ok {
					v.histogramPoints[b] = c
				} else {
					v.histogramPoints[b] += c
				}
			}
			for b := range v.histogramPoints {
				if _, ok := points[b]; !ok {
					delete(v.histogramPoints, b)
				}
			}
			v.histogramSum += ip.Sum()
			v.histogramCount += ip.Count()
			if v.timestamp.UnixNano() < ts.UnixNano() {
				v.timestamp = ts
			}
		}
		v.updated = time.Now()
		c.mu.Unlock()
	}
}

/*
	Reporting
*/
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debug("collect called")

	metrics := make([]prometheus.Metric, 0, len(c.metricsValues))

	c.mu.Lock()
	for k, v := range c.metricsValues {
		var m prometheus.Metric
		var err error
		if v.isHistogram {
			m, err = prometheus.NewConstHistogram(v.desc, v.histogramCount, v.histogramSum, v.histogramPoints, v.labelValues...)
		} else {
			m, err = prometheus.NewConstMetric(v.desc, v.metricType, v.value, v.labelValues...)
		}
		if err == nil {
			if !c.config.SendTimestamps {
				metrics = append(metrics, m)
			} else {
				metrics = append(metrics, prometheus.NewMetricWithTimestamp(v.timestamp, m))
			}
		}

		c.logger.Debug(fmt.Sprintf("metric served: %s", k))

		if time.Now().UnixNano()-v.updated.UnixNano() > c.config.MetricExpiration.Nanoseconds() {
			c.logger.Debug(fmt.Sprintf("metric expired: %s", k))
			delete(c.metricsValues, k)
		}
	}
	c.mu.Unlock()

	for _, m := range metrics {
		ch <- m
	}
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	c.registeredMetricsMu.Lock()
	registered := make(map[string]*prometheus.Desc)
	for key, desc := range c.registeredMetrics {
		registered[key] = desc
	}
	c.registeredMetricsMu.Unlock()

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
		if ip.IsNil() {
			continue
		}
		addLabelKeys(keySet, ip.LabelsMap())
	}
}

func collectLabelKeysDoubleDataPoints(dps pdata.DoubleDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dps.Len(); i++ {
		dp := dps.At(i)
		if dp.IsNil() {
			continue
		}
		addLabelKeys(keySet, dp.LabelsMap())
	}
}

func collectLabelKeysIntHistogramDataPoints(ihdp pdata.IntHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < ihdp.Len(); i++ {
		hp := ihdp.At(i)
		if hp.IsNil() {
			continue
		}
		addLabelKeys(keySet, hp.LabelsMap())
	}
}

func collectLabelKeysDoubleHistogramDataPoints(dhdp pdata.DoubleHistogramDataPointSlice, keySet map[string]struct{}) {
	for i := 0; i < dhdp.Len(); i++ {
		hp := dhdp.At(i)
		if hp.IsNil() {
			continue
		}
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
