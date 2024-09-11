// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

// prometheusChecker is used to assert exported metrics from a prometheus handler.
type prometheusChecker struct {
	otelHandler http.Handler
}

func (pc *prometheusChecker) checkScraperMetrics(receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	scraperAttrs := attributesForScraperMetrics(receiver, scraper)
	return multierr.Combine(
		pc.checkCounter("scraper_scraped_metric_points", scrapedMetricPoints, scraperAttrs),
		pc.checkCounter("scraper_errored_metric_points", erroredMetricPoints, scraperAttrs))
}

func (pc *prometheusChecker) checkReceiverTraces(receiver component.ID, protocol string, accepted, dropped int64) error {
	return pc.checkReceiver(receiver, "spans", protocol, accepted, dropped)
}

func (pc *prometheusChecker) checkReceiverLogs(receiver component.ID, protocol string, accepted, dropped int64) error {
	return pc.checkReceiver(receiver, "log_records", protocol, accepted, dropped)
}

func (pc *prometheusChecker) checkReceiverMetrics(receiver component.ID, protocol string, accepted, dropped int64) error {
	return pc.checkReceiver(receiver, "metric_points", protocol, accepted, dropped)
}

func (pc *prometheusChecker) checkReceiver(receiver component.ID, datatype, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		pc.checkCounter(fmt.Sprintf("receiver_accepted_%s", datatype), acceptedMetricPoints, receiverAttrs),
		pc.checkCounter(fmt.Sprintf("receiver_refused_%s", datatype), droppedMetricPoints, receiverAttrs))
}

func (pc *prometheusChecker) checkProcessorTraces(processor component.ID, accepted, refused, dropped, inserted int64) error {
	return pc.checkProcessor(processor, "spans", accepted, refused, dropped, inserted)
}

func (pc *prometheusChecker) checkProcessorMetrics(processor component.ID, accepted, refused, dropped, inserted int64) error {
	return pc.checkProcessor(processor, "metric_points", accepted, refused, dropped, inserted)
}

func (pc *prometheusChecker) checkProcessorLogs(processor component.ID, accepted, refused, dropped, inserted int64) error {
	return pc.checkProcessor(processor, "log_records", accepted, refused, dropped, inserted)
}

func (pc *prometheusChecker) checkProcessor(processor component.ID, datatype string, accepted, refused, dropped, inserted int64) error {
	processorAttrs := attributesForProcessorMetrics(processor)
	return multierr.Combine(
		pc.checkCounter(fmt.Sprintf("processor_accepted_%s", datatype), accepted, processorAttrs),
		pc.checkCounter(fmt.Sprintf("processor_refused_%s", datatype), refused, processorAttrs),
		pc.checkCounter(fmt.Sprintf("processor_dropped_%s", datatype), dropped, processorAttrs),
		pc.checkCounter(fmt.Sprintf("processor_inserted_%s", datatype), inserted, processorAttrs),
	)
}

func (pc *prometheusChecker) checkExporterTraces(exporter component.ID, sent, sendFailed int64) error {
	return pc.checkExporter(exporter, "spans", sent, sendFailed)
}

func (pc *prometheusChecker) checkExporterLogs(exporter component.ID, sent, sendFailed int64) error {
	return pc.checkExporter(exporter, "log_records", sent, sendFailed)
}

func (pc *prometheusChecker) checkExporterMetrics(exporter component.ID, sent, sendFailed int64) error {
	return pc.checkExporter(exporter, "metric_points", sent, sendFailed)
}

func (pc *prometheusChecker) checkExporter(exporter component.ID, datatype string, sent, sendFailed int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	errs := pc.checkCounter(fmt.Sprintf("exporter_sent_%s", datatype), sent, exporterAttrs)
	if sendFailed > 0 {
		errs = multierr.Append(errs,
			pc.checkCounter(fmt.Sprintf("exporter_send_failed_%s", datatype), sendFailed, exporterAttrs))
	}
	return errs
}

func (pc *prometheusChecker) checkExporterEnqueueFailed(exporter component.ID, datatype string, enqueueFailed int64) error {
	if enqueueFailed == 0 {
		return nil
	}
	exporterAttrs := attributesForExporterMetrics(exporter)
	return pc.checkCounter(fmt.Sprintf("exporter_enqueue_failed_%s", datatype), enqueueFailed, exporterAttrs)
}

func (pc *prometheusChecker) checkGauge(metric string, val int64, attrs []attribute.KeyValue) error {
	ts, err := pc.getMetric(metric, io_prometheus_client.MetricType_GAUGE, attrs)
	if err != nil {
		return err
	}

	expected := float64(val)
	if math.Abs(ts.GetGauge().GetValue()-expected) > 0.0001 {
		return fmt.Errorf("values for metric '%s' did not match, expected '%f' got '%f'", metric, expected, ts.GetGauge().GetValue())
	}

	return nil
}

func (pc *prometheusChecker) checkCounter(expectedMetric string, value int64, attrs []attribute.KeyValue) error {
	ts, err := pc.getMetric(fmt.Sprintf("otelcol_%s", expectedMetric), io_prometheus_client.MetricType_COUNTER, attrs)
	if err != nil {
		return err
	}

	expected := float64(value)
	if math.Abs(expected-ts.GetCounter().GetValue()) > 0.0001 {
		return fmt.Errorf("values for metric '%s' did not match, expected '%f' got '%f'", expectedMetric, expected, ts.GetCounter().GetValue())
	}

	return nil
}

// getMetric returns the metric time series that matches the given name, type and set of attributes
// it fetches data from the prometheus endpoint and parse them, ideally OTel Go should provide a MeterRecorder of some kind.
func (pc *prometheusChecker) getMetric(expectedName string, expectedType io_prometheus_client.MetricType, expectedAttrs []attribute.KeyValue) (*io_prometheus_client.Metric, error) {
	parsed, err := fetchPrometheusMetrics(pc.otelHandler)
	if err != nil {
		return nil, err
	}

	metricFamily, ok := parsed[expectedName]
	if !ok {
		return nil, fmt.Errorf("metric '%s' not found", expectedName)
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

// attributesForExporterMetrics returns the attributes that are needed for the receiver metrics.
func attributesForExporterMetrics(exporter component.ID) []attribute.KeyValue {
	return []attribute.KeyValue{attribute.String(exporterTag, exporter.String())}
}
