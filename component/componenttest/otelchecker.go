// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

func checkScraperMetrics(reader *sdkmetric.ManualReader, receiver component.ID, scraper component.ID, scrapedMetricPoints, erroredMetricPoints int64) error {
	scraperAttrs := attributesForScraperMetrics(receiver, scraper)
	return multierr.Combine(
		checkIntSum(reader, "otelcol_scraper_scraped_metric_points", scrapedMetricPoints, scraperAttrs),
		checkIntSum(reader, "otelcol_scraper_errored_metric_points", erroredMetricPoints, scraperAttrs))
}

func checkReceiverTraces(reader *sdkmetric.ManualReader, receiver component.ID, protocol string, accepted, dropped int64) error {
	return checkReceiver(reader, receiver, "spans", protocol, accepted, dropped)
}

func checkReceiverLogs(reader *sdkmetric.ManualReader, receiver component.ID, protocol string, accepted, dropped int64) error {
	return checkReceiver(reader, receiver, "log_records", protocol, accepted, dropped)
}

func checkReceiverMetrics(reader *sdkmetric.ManualReader, receiver component.ID, protocol string, accepted, dropped int64) error {
	return checkReceiver(reader, receiver, "metric_points", protocol, accepted, dropped)
}

func checkReceiver(reader *sdkmetric.ManualReader, receiver component.ID, datatype, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		checkIntSum(reader, fmt.Sprintf("otelcol_receiver_accepted_%s", datatype), acceptedMetricPoints, receiverAttrs),
		checkIntSum(reader, fmt.Sprintf("otelcol_receiver_refused_%s", datatype), droppedMetricPoints, receiverAttrs))
}

func checkExporterTraces(reader *sdkmetric.ManualReader, exporter component.ID, sent, sendFailed int64) error {
	return checkExporter(reader, exporter, "spans", sent, sendFailed)
}

func checkExporterLogs(reader *sdkmetric.ManualReader, exporter component.ID, sent, sendFailed int64) error {
	return checkExporter(reader, exporter, "log_records", sent, sendFailed)
}

func checkExporterMetrics(reader *sdkmetric.ManualReader, exporter component.ID, sent, sendFailed int64) error {
	return checkExporter(reader, exporter, "metric_points", sent, sendFailed)
}

func checkExporter(reader *sdkmetric.ManualReader, exporter component.ID, datatype string, sent, sendFailed int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	errs := checkIntSum(reader, fmt.Sprintf("otelcol_exporter_sent_%s", datatype), sent, exporterAttrs)
	if sendFailed > 0 {
		errs = multierr.Append(errs,
			checkIntSum(reader, fmt.Sprintf("otelcol_exporter_send_failed_%s", datatype), sendFailed, exporterAttrs))
	}
	return errs
}

func checkExporterEnqueueFailed(reader *sdkmetric.ManualReader, exporter component.ID, datatype string, enqueueFailed int64) error {
	if enqueueFailed == 0 {
		return nil
	}
	exporterAttrs := attributesForExporterMetrics(exporter)
	return checkIntSum(reader, fmt.Sprintf("otelcol_exporter_enqueue_failed_%s", datatype), enqueueFailed, exporterAttrs)
}

func checkIntGauge(reader *sdkmetric.ManualReader, metric string, expected int64, expectedAttrs attribute.Set) error {
	dp, err := getGaugeDataPoint[int64](reader, metric, expectedAttrs)
	if err != nil {
		return err
	}

	if dp.Value != expected {
		return fmt.Errorf("values for metric '%s' did not match, expected '%d' got '%d'", metric, expected, dp.Value)
	}

	return nil
}

func checkIntSum(reader *sdkmetric.ManualReader, expectedMetric string, expected int64, expectedAttrs attribute.Set) error {
	dp, err := getSumDataPoint[int64](reader, expectedMetric, expectedAttrs)
	if err != nil {
		return err
	}

	if dp.Value != expected {
		return fmt.Errorf("values for metric '%s' did not match, expected '%d' got '%d'", expectedMetric, expected, dp.Value)
	}

	return nil
}

func getSumDataPoint[N int64 | float64](reader *sdkmetric.ManualReader, expectedName string, expectedAttrs attribute.Set) (metricdata.DataPoint[N], error) {
	m, err := getMetric(reader, expectedName)
	if err != nil {
		return metricdata.DataPoint[N]{}, err
	}

	switch a := m.Data.(type) {
	case metricdata.Sum[N]:
		return getDataPoint(a.DataPoints, expectedName, expectedAttrs)
	default:
		return metricdata.DataPoint[N]{}, fmt.Errorf("unknown metric type: %T", a)
	}
}

func getGaugeDataPoint[N int64 | float64](reader *sdkmetric.ManualReader, expectedName string, expectedAttrs attribute.Set) (metricdata.DataPoint[N], error) {
	m, err := getMetric(reader, expectedName)
	if err != nil {
		return metricdata.DataPoint[N]{}, err
	}

	switch a := m.Data.(type) {
	case metricdata.Gauge[N]:
		return getDataPoint(a.DataPoints, expectedName, expectedAttrs)
	default:
		return metricdata.DataPoint[N]{}, fmt.Errorf("unknown metric type: %T", a)
	}
}

func getDataPoint[N int64 | float64](dps []metricdata.DataPoint[N], expectedName string, expectedAttrs attribute.Set) (metricdata.DataPoint[N], error) {
	for _, dp := range dps {
		if expectedAttrs.Equals(&dp.Attributes) {
			return dp, nil
		}
	}
	return metricdata.DataPoint[N]{}, fmt.Errorf("metric '%s' doesn't have a data point with the given attributes: %s", expectedName, expectedAttrs.Encoded(attribute.DefaultEncoder()))
}

func getMetric(reader *sdkmetric.ManualReader, expectedName string) (metricdata.Metrics, error) {
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		return metricdata.Metrics{}, err
	}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == expectedName {
				return m, nil
			}
		}
	}
	return metricdata.Metrics{}, fmt.Errorf("metric '%s' not found", expectedName)
}

func attributesForScraperMetrics(receiver component.ID, scraper component.ID) attribute.Set {
	return attribute.NewSet(
		attribute.String(receiverTag, receiver.String()),
		attribute.String(scraperTag, scraper.String()),
	)
}

// attributesForReceiverMetrics returns the attributes that are needed for the receiver metrics.
func attributesForReceiverMetrics(receiver component.ID, transport string) attribute.Set {
	return attribute.NewSet(
		attribute.String(receiverTag, receiver.String()),
		attribute.String(transportTag, transport),
	)
}

// attributesForExporterMetrics returns the attributes that are needed for the receiver metrics.
func attributesForExporterMetrics(exporter component.ID, extraAttrs ...attribute.KeyValue) attribute.Set {
	attrs := []attribute.KeyValue{attribute.String(exporterTag, exporter.String())}
	attrs = append(attrs, extraAttrs...)
	return attribute.NewSet(attrs...)
}
