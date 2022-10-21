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

package obsreporttest // import "go.opentelemetry.io/collector/obsreport/obsreporttest"

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/config"
)

func newStubPromChecker() prometheusChecker {
	return prometheusChecker{
		promHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`
# HELP receiver_accepted_log_records Number of log records successfully pushed into the pipeline.
# TYPE receiver_accepted_log_records counter
receiver_accepted_log_records{receiver="fakeReceiver",transport="fakeTransport"} 102
# HELP receiver_accepted_metric_points Number of metric points successfully pushed into the pipeline.
# TYPE receiver_accepted_metric_points counter
receiver_accepted_metric_points{receiver="fakeReceiver",transport="fakeTransport"} 7
# HELP receiver_accepted_spans Number of spans successfully pushed into the pipeline.
# TYPE receiver_accepted_spans counter
receiver_accepted_spans{receiver="fakeReceiver",transport="fakeTransport"} 42
# HELP receiver_refused_log_records Number of log records that could not be pushed into the pipeline.
# TYPE receiver_refused_log_records counter
receiver_refused_log_records{receiver="fakeReceiver",transport="fakeTransport"} 35
# HELP receiver_refused_metric_points Number of metric points that could not be pushed into the pipeline.
# TYPE receiver_refused_metric_points counter
receiver_refused_metric_points{receiver="fakeReceiver",transport="fakeTransport"} 41
# HELP receiver_refused_spans Number of spans that could not be pushed into the pipeline.
# TYPE receiver_refused_spans counter
receiver_refused_spans{receiver="fakeReceiver",transport="fakeTransport"} 13
# HELP gauge_metric A simple gauge metric
# TYPE gauge_metric gauge
gauge_metric 49
`))
		}),
	}
}

func TestPromChecker(t *testing.T) {
	pc := newStubPromChecker()
	receiver := config.NewComponentID("fakeReceiver")
	transport := "fakeTransport"

	assert.NoError(t,
		pc.checkCounter("receiver_accepted_spans", 42, []attribute.KeyValue{attribute.String("receiver", receiver.String()), attribute.String("transport", transport)}),
		"correct assertion should return no error",
	)

	assert.Error(t,
		pc.checkCounter("receiver_accepted_spans", 15, []attribute.KeyValue{attribute.String("receiver", receiver.String()), attribute.String("transport", transport)}),
		"invalid value should return error",
	)

	assert.Error(t,
		pc.checkCounter("invalid_name", 42, []attribute.KeyValue{attribute.String("receiver", receiver.String()), attribute.String("transport", transport)}),
		"invalid name should return error",
	)

	assert.Error(t,
		pc.checkCounter("receiver_accepted_spans", 42, []attribute.KeyValue{attribute.String("receiver", "notFakeReceiver"), attribute.String("transport", transport)}),
		"invalid attributes should return error",
	)

	assert.Error(t,
		pc.checkCounter("gauge_metric", 49, nil),
		"invalid metric type should return error",
	)

	assert.NoError(t,
		pc.checkReceiverTraces(receiver, transport, 42, 13),
		"metrics from Receiver Traces should be valid",
	)

	assert.NoError(t,
		pc.checkReceiverMetrics(receiver, transport, 7, 41),
		"metrics from Receiver Metrics should be valid",
	)

	assert.NoError(t,
		pc.checkReceiverLogs(receiver, transport, 102, 35),
		"metrics from Receiver Logs should be valid",
	)
}
