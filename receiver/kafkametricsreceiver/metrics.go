// Copyright  The OpenTelemetry Authors
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

package kafkametricsreceiver

import (
	"time"

	"go.opentelemetry.io/collector/consumer/pdata"
)

const (
	brokersName        = "kafka_brokers"
	brokersDescription = "number of brokers in cluster"
)

type brokerMetrics struct {
	brokers *pdata.Metric
}

func initializeBrokerMetrics(metrics *pdata.MetricSlice) *brokerMetrics {
	metrics.Resize(0)
	metrics.Resize(1)

	brokers := metrics.At(0)
	initializeMetric(&brokers, brokersName, brokersDescription)

	return &brokerMetrics{brokers: &brokers}
}

func initializeMetric(m *pdata.Metric, name string, description string) {
	m.SetName(name)
	m.SetDescription(description)
	m.SetDataType(pdata.MetricDataTypeIntGauge)
}

func addBrokersToMetric(brokers int64, m *pdata.Metric) {
	dpLen := m.IntGauge().DataPoints().Len()
	m.IntGauge().DataPoints().Resize(dpLen + 1)
	dp := m.IntGauge().DataPoints().At(dpLen)
	dp.SetValue(brokers)
	dp.SetTimestamp(timeToUnixNano(time.Now()))
}

func timeToUnixNano(t time.Time) pdata.TimestampUnixNano {
	return pdata.TimestampUnixNano(uint64(t.UnixNano()))
}
