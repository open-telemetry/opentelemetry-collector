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
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.opencensus.io/stats/view"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/config"
)

// prometheusChecker is used to assert exported metrics from a prometheus handler.
type prometheusChecker struct {
	promHandler http.Handler
}

func (pc *prometheusChecker) checkReceiverTraces(receiver config.ComponentID, protocol string, acceptedSpans, droppedSpans int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_spans", acceptedSpans, receiverAttrs),
		pc.checkCounter("receiver_refused_spans", droppedSpans, receiverAttrs))
}

func (pc *prometheusChecker) checkReceiverLogs(receiver config.ComponentID, protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_log_records", acceptedLogRecords, receiverAttrs),
		pc.checkCounter("receiver_refused_log_records", droppedLogRecords, receiverAttrs))
}

func (pc *prometheusChecker) checkReceiverMetrics(receiver config.ComponentID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_metric_points", acceptedMetricPoints, receiverAttrs),
		pc.checkCounter("receiver_refused_metric_points", droppedMetricPoints, receiverAttrs))
}

func (pc *prometheusChecker) checkCounter(expectedMetric string, value int64, attrs []attribute.KeyValue) error {
	// Forces a flush for the opencensus view data.
	_, _ = view.RetrieveData(expectedMetric)

	ts, err := pc.getMetric(expectedMetric, io_prometheus_client.MetricType_COUNTER, attrs)
	if err != nil {
		return err
	}

	expected := float64(value)
	if math.Abs(expected-ts.GetCounter().GetValue()) > 0.0001 {
		return fmt.Errorf("values for metric '%s' did no match, expected '%f' got '%f'", expectedMetric, expected, ts.GetCounter().GetValue())
	}

	return nil
}

// getMetric returns the metric time series that matches the given name, type and set of attributes
// it fetches data from the prometheus endpoint and parse them, ideally OTel Go should provide a MeterRecorder of some kind.
func (pc *prometheusChecker) getMetric(expectedName string, expectedType io_prometheus_client.MetricType, expectedAttrs []attribute.KeyValue) (*io_prometheus_client.Metric, error) {
	parsed, err := fetchPrometheusMetrics(pc.promHandler)
	if err != nil {
		return nil, err
	}

	metricFamily, ok := parsed[expectedName]
	if !ok {
		// OTel Go adds `_total` suffix for all monotonic sum.
		metricFamily, ok = parsed[expectedName+"_total"]
		if !ok {
			return nil, fmt.Errorf("metric '%s' not found", expectedName)
		}
	}

	if metricFamily.Type.String() != expectedType.String() {
		return nil, fmt.Errorf("metric '%v' has type '%s' instead of '%s'", expectedName, metricFamily.Type.String(), expectedType.String())
	}

	expectedSet := attribute.NewSet(expectedAttrs...)

	for _, metric := range metricFamily.Metric {
		var attrs []attribute.KeyValue

		for _, label := range metric.Label {
			attrs = append(attrs, attribute.String(label.GetName(), label.GetValue()))
		}
		set := attribute.NewSet(attrs...)

		if expectedSet.Equals(&set) {
			return metric, nil
		}
	}

	return nil, fmt.Errorf("metric '%s' doesn't have a timeseries with the given attributes: %s", expectedName, expectedSet.Encoded(attribute.DefaultEncoder()))
}

func fetchPrometheusMetrics(handler http.Handler) (map[string]*io_prometheus_client.MetricFamily, error) {
	req, err := http.NewRequest("GET", "/metrics", nil)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(rr.Body)
}

// attributesForReceiverMetrics returns the attributes that are needed for the receiver metrics.
func attributesForReceiverMetrics(receiver config.ComponentID, transport string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(receiverTag.Name(), receiver.String()),
		attribute.String(transportTag.Name(), transport),
	}
}
