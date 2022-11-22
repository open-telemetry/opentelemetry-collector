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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"go.opentelemetry.io/collector/component"
)

func newStubPromChecker() (prometheusChecker, error) {
	promBytes, err := os.ReadFile(filepath.Join("testdata", "prometheus_response"))
	if err != nil {
		return prometheusChecker{}, err
	}

	promResponse := strings.ReplaceAll(string(promBytes), "\r\n", "\n")

	return prometheusChecker{
		promHandler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(promResponse))
		}),
	}, nil
}

func TestPromChecker(t *testing.T) {
	pc, err := newStubPromChecker()
	require.NoError(t, err)

	scraper := component.NewID("fakeScraper")
	receiver := component.NewID("fakeReceiver")
	processor := component.NewID("fakeProcessor")
	exporter := component.NewID("fakeExporter")
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
		pc.checkScraperMetrics(receiver, scraper, 7, 41),
		"metrics from Scraper Metrics should be valid",
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

	assert.NoError(t,
		pc.checkProcessorTraces(processor, 42, 13, 7),
		"metrics from Receiver Traces should be valid",
	)

	assert.NoError(t,
		pc.checkProcessorMetrics(processor, 7, 41, 13),
		"metrics from Receiver Metrics should be valid",
	)

	assert.NoError(t,
		pc.checkProcessorLogs(processor, 102, 35, 14),
		"metrics from Receiver Logs should be valid",
	)

	assert.NoError(t,
		pc.checkExporterTraces(exporter, 43, 14),
		"metrics from Exporter Traces should be valid",
	)

	assert.NoError(t,
		pc.checkExporterMetrics(exporter, 8, 42),
		"metrics from Exporter Metrics should be valid",
	)

	assert.NoError(t,
		pc.checkExporterLogs(exporter, 103, 36),
		"metrics from Exporter Logs should be valid",
	)
}
