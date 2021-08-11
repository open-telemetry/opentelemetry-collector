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
	"fmt"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/model/pdata"
)

type collector struct {
	accumulator accumulator
	logger      *zap.Logger

	sendTimestamps bool
	namespace      string
	constLabels    prometheus.Labels
}

func newCollector(config *Config, logger *zap.Logger) *collector {
	return &collector{
		accumulator:    newAccumulator(logger, config.MetricExpiration),
		logger:         logger,
		namespace:      sanitize(config.Namespace),
		sendTimestamps: config.SendTimestamps,
		constLabels:    config.ConstLabels,
	}
}

// Describe is a no-op, because the collector dynamically allocates metrics.
// https://github.com/prometheus/client_golang/blob/v1.9.0/prometheus/collector.go#L28-L40
func (c *collector) Describe(_ chan<- *prometheus.Desc) {}

/*
	Processing
*/
func (c *collector) processMetrics(rm pdata.ResourceMetrics) (n int) {
	return c.accumulator.Accumulate(rm)
}

var errUnknownMetricType = fmt.Errorf("unknown metric type")

func (c *collector) convertMetric(metric pdata.Metric) (prometheus.Metric, error) {
	switch metric.DataType() {
	case pdata.MetricDataTypeGauge:
		return c.convertGauge(metric)
	case pdata.MetricDataTypeSum:
		return c.convertSum(metric)
	case pdata.MetricDataTypeHistogram:
		return c.convertDoubleHistogram(metric)
	case pdata.MetricDataTypeSummary:
		return c.convertSummary(metric)
	}

	return nil, errUnknownMetricType
}

func metricName(namespace string, metric pdata.Metric) string {
	if namespace != "" {
		return namespace + "_" + sanitize(metric.Name())
	}
	return sanitize(metric.Name())
}

func (c *collector) getMetricMetadata(metric pdata.Metric, attributes pdata.AttributeMap) (*prometheus.Desc, []string) {
	keys := make([]string, 0, attributes.Len())
	values := make([]string, 0, attributes.Len())

	attributes.Range(func(k string, v pdata.AttributeValue) bool {
		keys = append(keys, sanitize(k))
		values = append(values, v.StringVal())
		return true
	})

	return prometheus.NewDesc(
		metricName(c.namespace, metric),
		metric.Description(),
		keys,
		c.constLabels,
	), values
}

func (c *collector) convertGauge(metric pdata.Metric) (prometheus.Metric, error) {
	ip := metric.Gauge().DataPoints().At(0)

	desc, attributes := c.getMetricMetadata(metric, ip.Attributes())
	var value float64
	switch ip.Type() {
	case pdata.MetricValueTypeInt:
		value = float64(ip.IntVal())
	case pdata.MetricValueTypeDouble:
		value = ip.DoubleVal()
	}
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, attributes...)
	if err != nil {
		return nil, err
	}

	if c.sendTimestamps {
		return prometheus.NewMetricWithTimestamp(ip.Timestamp().AsTime(), m), nil
	}
	return m, nil
}

func (c *collector) convertSum(metric pdata.Metric) (prometheus.Metric, error) {
	ip := metric.Sum().DataPoints().At(0)

	metricType := prometheus.GaugeValue
	if metric.Sum().IsMonotonic() {
		metricType = prometheus.CounterValue
	}

	desc, attributes := c.getMetricMetadata(metric, ip.Attributes())
	var value float64
	switch ip.Type() {
	case pdata.MetricValueTypeInt:
		value = float64(ip.IntVal())
	case pdata.MetricValueTypeDouble:
		value = ip.DoubleVal()
	}
	m, err := prometheus.NewConstMetric(desc, metricType, value, attributes...)
	if err != nil {
		return nil, err
	}

	if c.sendTimestamps {
		return prometheus.NewMetricWithTimestamp(ip.Timestamp().AsTime(), m), nil
	}
	return m, nil
}

func (c *collector) convertSummary(metric pdata.Metric) (prometheus.Metric, error) {
	// TODO: In the off chance that we have multiple points
	// within the same metric, how should we handle them?
	point := metric.Summary().DataPoints().At(0)

	quantiles := make(map[float64]float64)
	qv := point.QuantileValues()
	for j := 0; j < qv.Len(); j++ {
		qvj := qv.At(j)
		// There should be EXACTLY one quantile value lest it is an invalid exposition.
		quantiles[qvj.Quantile()] = qvj.Value()
	}

	desc, attributes := c.getMetricMetadata(metric, point.Attributes())
	m, err := prometheus.NewConstSummary(desc, point.Count(), point.Sum(), quantiles, attributes...)
	if err != nil {
		return nil, err
	}
	if c.sendTimestamps {
		return prometheus.NewMetricWithTimestamp(point.Timestamp().AsTime(), m), nil
	}
	return m, nil
}

func (c *collector) convertDoubleHistogram(metric pdata.Metric) (prometheus.Metric, error) {
	ip := metric.Histogram().DataPoints().At(0)
	desc, attributes := c.getMetricMetadata(metric, ip.Attributes())

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

	m, err := prometheus.NewConstHistogram(desc, ip.Count(), ip.Sum(), points, attributes...)
	if err != nil {
		return nil, err
	}

	if c.sendTimestamps {
		return prometheus.NewMetricWithTimestamp(ip.Timestamp().AsTime(), m), nil
	}
	return m, nil
}

/*
	Reporting
*/
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	c.logger.Debug("collect called")

	inMetrics := c.accumulator.Collect()

	for _, pMetric := range inMetrics {
		m, err := c.convertMetric(pMetric)
		if err != nil {
			c.logger.Error(fmt.Sprintf("failed to convert metric %s: %s", pMetric.Name(), err.Error()))
			continue
		}

		ch <- m
		c.logger.Debug(fmt.Sprintf("metric served: %s", m.Desc().String()))
	}
}
