/*
 * Copyright The OpenTelemetry Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package consumermock

import (
	"context"
	"sync"

	"go.uber.org/atomic"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerdata"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/consumer/pdatautil"
)

var _ consumer.MetricsConsumer = (*Metric)(nil)
var _ consumer.MetricsConsumerOld = (*Metric)(nil)

type Metric struct {
	sync.Mutex
	consumeMetricsError error // to be returned by Consume, if set
	metricsReceived     atomic.Uint64
	receivedMetrics     []pdata.Metrics
	receivedMetricsOld  []consumerdata.MetricsData
}

func (mc *Metric) ConsumeMetrics(_ context.Context, md pdata.Metrics) error {
	if mc.consumeMetricsError != nil {
		return mc.consumeMetricsError
	}
	_, dataPoints := pdatautil.MetricAndDataPointCount(md)
	mc.metricsReceived.Add(uint64(dataPoints))

	mc.Lock()
	defer mc.Unlock()
	mc.receivedMetrics = append(mc.receivedMetrics, md)

	return nil
}

// ConsumeMetricOld consumes metric data in old representation
func (mc *Metric) ConsumeMetricsData(ctx context.Context, md consumerdata.MetricsData) error {
	if mc.consumeMetricsError != nil {
		return mc.consumeMetricsError
	}

	dataPoints := 0
	for _, metric := range md.Metrics {
		if metric == nil {
			continue
		}
		for _, ts := range metric.Timeseries {
			dataPoints += len(ts.Points)
		}
	}

	mc.metricsReceived.Add(uint64(dataPoints))

	mc.Lock()
	defer mc.Unlock()
	mc.receivedMetricsOld = append(mc.receivedMetricsOld, md)

	return nil
}

// ClearReceivedItems clears the list of received traces and metrics. Note: counters
// return by DataItemsReceived() are not cleared, they are cumulative.
func (mc *Metric) ClearReceivedItems() {
	mc.Lock()
	defer mc.Unlock()
	mc.receivedMetrics = nil
	mc.receivedMetricsOld = nil
}

// MetricsReceived returns number of spans received by the consumer.
func (mc *Metric) MetricsReceived() uint64 {
	return mc.metricsReceived.Load()
}

func (mc *Metric) MetricsOld() []consumerdata.MetricsData {
	mc.Lock()
	defer mc.Unlock()
	return mc.receivedMetricsOld
}

func (mc *Metric) Metrics() []pdata.Metrics {
	mc.Lock()
	defer mc.Unlock()
	return mc.receivedMetrics
}

func (mc *Metric) SetConsumeMetricsError(err error) {
	mc.consumeMetricsError = err
}
