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
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/pdata"
)

type metricHolder struct {
	desc         *prometheus.Desc
	metricValues map[string]*metricValue
}

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
	handler      http.Handler
	collector    *collector
}

func newPrometheusExporter(cfg *Config, shutdownFunc func() error, logger *zap.Logger) (*prometheusExporter, error) {
	registry := prometheus.NewRegistry()
	collector := &collector{
		registry:          registry,
		config:            cfg,
		registeredMetrics: make(map[string]*metricHolder),
		logger:            logger,
	}

	if err := registry.Register(collector); err != nil {
		return nil, err
	}

	return &prometheusExporter{
		name:         cfg.Name(),
		shutdownFunc: shutdownFunc,
		collector:    collector,
		handler: promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				ErrorHandling: promhttp.ContinueOnError,
			},
		),
	}, nil
}

func (pe *prometheusExporter) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (pe *prometheusExporter) ConsumeMetrics(ctx context.Context, md pdata.Metrics) error {
	rmetrics := md.ResourceMetrics()
	for i := 0; i < rmetrics.Len(); i++ {
		rs := rmetrics.At(i)

		pe.collector.processMetrics(rs)
	}

	return nil
}

// Shutdown stops the exporter and is invoked during shutdown.
func (pe *prometheusExporter) Shutdown(context.Context) error {
	return pe.shutdownFunc()
}

// Collector
type collector struct {
	config            *Config
	mu                sync.Mutex
	registry          *prometheus.Registry
	registeredMetrics map[string]*metricHolder
	logger            *zap.Logger
}

/*
	Processing
*/
func (c *collector) processMetrics(rm pdata.ResourceMetrics) {
	count := 0
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
				count++
			}

			c.accumulateMetrics(metric, lk, holder.desc)
			c.mu.Unlock()
		}
	}
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

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		}
		v.value = float64(ip.Value())
		v.timestamp = ts
		v.updated = time.Now()
	}
}

func (c *collector) accumulateDoubleGauge(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
	dps := metric.DoubleGauge().DataPoints()
	for i := 0; i < dps.Len(); i++ {
		ip := dps.At(i)

		ts := pdata.UnixNanoToTime(ip.Timestamp())
		labelValues := collectLabelValues(ip.LabelsMap(), lk)
		signature := metricSignature(c.config.Namespace, metric, lk.keys)
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.GaugeValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		}
		v.value = ip.Value()
		v.timestamp = ts
		v.updated = time.Now()
	}
}

func (c *collector) accumulateIntSum(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
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
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
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

func (c *collector) accumulateDoubleSum(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
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
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

		v, ok := c.registeredMetrics[signature].metricValues[valueSignature]
		if !ok {
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, metricType: prometheus.CounterValue}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
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

func (c *collector) accumulateIntHistogram(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
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
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

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
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, isHistogram: true}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
		}

		v.histogramPoints = points
		v.histogramSum = float64(ip.Sum())
		v.histogramCount = ip.Count()
		v.timestamp = ts

		v.updated = time.Now()
	}
}

func (c *collector) accumulateDoubleHistogram(metric pdata.Metric, lk *labelKeys, desc *prometheus.Desc) {
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
		valueSignature := metricSignature(c.config.Namespace, metric, append(lk.keys, labelValues...))

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
			v = &metricValue{desc: desc, value: 0, labelValues: labelValues, isHistogram: true}
			c.registeredMetrics[signature].metricValues[valueSignature] = v
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

	metrics := make([]prometheus.Metric, 0, len(c.registeredMetrics))

	c.mu.Lock()

	for signature, metric := range c.registeredMetrics {
		for k, v := range metric.metricValues {
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

			if time.Now().UnixNano()-v.updated.UnixNano() > c.config.MetricExpiration.Nanoseconds() {
				c.logger.Debug(fmt.Sprintf("metric expired: %s", k))
				delete(metric.metricValues, k)
			}
		}

		if len(metric.metricValues) == 0 {
			delete(c.registeredMetrics, signature)
		}
	}

	c.mu.Unlock()

	for _, m := range metrics {
		ch <- m
		c.logger.Debug(fmt.Sprintf("metric served: %s", m))
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
