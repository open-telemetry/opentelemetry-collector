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

	"go.opentelemetry.io/collector/component"
)

// prometheusChecker is used to assert exported metrics from a prometheus handler.
type prometheusChecker struct {
	promHandler http.Handler
}

func (pc *prometheusChecker) checkScraperMetrics(receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	scraperAttrs := attributesForScraperMetrics(receiver, scraper)
	return multierr.Combine(
		pc.checkCounter("scraper_scraped_metric_points", scrapedMetricPoints, scraperAttrs),
		pc.checkCounter("scraper_errored_metric_points", erroredMetricPoints, scraperAttrs))
}

func (pc *prometheusChecker) checkReceiverTraces(receiver component.ID, protocol string, acceptedSpans, droppedSpans int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_spans", acceptedSpans, receiverAttrs),
		pc.checkCounter("receiver_refused_spans", droppedSpans, receiverAttrs))
}

func (pc *prometheusChecker) checkReceiverLogs(receiver component.ID, protocol string, acceptedLogRecords, droppedLogRecords int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_log_records", acceptedLogRecords, receiverAttrs),
		pc.checkCounter("receiver_refused_log_records", droppedLogRecords, receiverAttrs))
}

func (pc *prometheusChecker) checkReceiverMetrics(receiver component.ID, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter("receiver_accepted_metric_points", acceptedMetricPoints, receiverAttrs),
		pc.checkCounter("receiver_refused_metric_points", droppedMetricPoints, receiverAttrs))
}

func (pc *prometheusChecker) checkProcessorTraces(processor component.ID, acceptedSpans, refusedSpans, droppedSpans int64) error {
	processorAttrs := attributesForProcessorMetrics(processor)
	return multierr.Combine(
		pc.checkCounter("processor_accepted_spans", acceptedSpans, processorAttrs),
		pc.checkCounter("processor_refused_spans", refusedSpans, processorAttrs),
		pc.checkCounter("processor_dropped_spans", droppedSpans, processorAttrs))
}

func (pc *prometheusChecker) checkProcessorMetrics(processor component.ID, acceptedMetricPoints, refusedMetricPoints, droppedMetricPoints int64) error {
	processorAttrs := attributesForProcessorMetrics(processor)
	return multierr.Combine(
		pc.checkCounter("processor_accepted_metric_points", acceptedMetricPoints, processorAttrs),
		pc.checkCounter("processor_refused_metric_points", refusedMetricPoints, processorAttrs),
		pc.checkCounter("processor_dropped_metric_points", droppedMetricPoints, processorAttrs))
}

func (pc *prometheusChecker) checkProcessorLogs(processor component.ID, acceptedLogRecords, refusedLogRecords, droppedLogRecords int64) error {
	processorAttrs := attributesForProcessorMetrics(processor)
	return multierr.Combine(
		pc.checkCounter("processor_accepted_log_records", acceptedLogRecords, processorAttrs),
		pc.checkCounter("processor_refused_log_records", refusedLogRecords, processorAttrs),
		pc.checkCounter("processor_dropped_log_records", droppedLogRecords, processorAttrs))
}

func (pc *prometheusChecker) checkExporterTraces(exporter component.ID, sentSpans, sendFailedSpans int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	if sendFailedSpans > 0 {
		return multierr.Combine(
			pc.checkCounter("exporter_sent_spans", sentSpans, exporterAttrs),
			pc.checkCounter("exporter_send_failed_spans", sendFailedSpans, exporterAttrs))
	}
	return multierr.Combine(
		pc.checkCounter("exporter_sent_spans", sentSpans, exporterAttrs))
}

func (pc *prometheusChecker) checkExporterLogs(exporter component.ID, sentLogRecords, sendFailedLogRecords int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	if sendFailedLogRecords > 0 {
		return multierr.Combine(
			pc.checkCounter("exporter_sent_log_records", sentLogRecords, exporterAttrs),
			pc.checkCounter("exporter_send_failed_log_records", sendFailedLogRecords, exporterAttrs))
	}
	return multierr.Combine(
		pc.checkCounter("exporter_sent_log_records", sentLogRecords, exporterAttrs))
}

func (pc *prometheusChecker) checkExporterMetrics(exporter component.ID, sentMetricPoints, sendFailedMetricPoints int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	if sendFailedMetricPoints > 0 {
		return multierr.Combine(
			pc.checkCounter("exporter_sent_metric_points", sentMetricPoints, exporterAttrs),
			pc.checkCounter("exporter_send_failed_metric_points", sendFailedMetricPoints, exporterAttrs))
	}
	return multierr.Combine(
		pc.checkCounter("exporter_sent_metric_points", sentMetricPoints, exporterAttrs))
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
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	return parser.TextToMetricFamilies(rr.Body)
}

func attributesForScraperMetrics(receiver component.ID, scraper component.ID) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(receiverTag, receiver.String()),
		attribute.String(scraperTag, scraper.String()),
	}
}

// attributesForReceiverMetrics returns the attributes that are needed for the receiver metrics.
func attributesForReceiverMetrics(receiver component.ID, transport string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(receiverTag, receiver.String()),
		attribute.String(transportTag, transport),
	}
}

func attributesForProcessorMetrics(processor component.ID) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String(processorTag, processor.String())}
}

// attributesForReceiverMetrics returns the attributes that are needed for the receiver metrics.
func attributesForExporterMetrics(exporter component.ID) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String(exporterTag, exporter.String())}
}
