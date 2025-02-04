// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package componenttest // import "go.opentelemetry.io/collector/component/componenttest"

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/multierr"

	"go.opentelemetry.io/collector/component"
)

func checkReceiver(tel *Telemetry, receiver component.ID, datatype, protocol string, acceptedMetricPoints, droppedMetricPoints int64) error {
	receiverAttrs := attributesForReceiverMetrics(receiver, protocol)
	return multierr.Combine(
		checkIntSum(tel, "otelcol_receiver_accepted_"+datatype, acceptedMetricPoints, receiverAttrs),
		checkIntSum(tel, "otelcol_receiver_refused_"+datatype, droppedMetricPoints, receiverAttrs))
}

func checkExporter(tel *Telemetry, exporter component.ID, datatype string, sent, sendFailed int64) error {
	exporterAttrs := attributesForExporterMetrics(exporter)
	errs := checkIntSum(tel, "otelcol_exporter_sent_"+datatype, sent, exporterAttrs)
	if sendFailed > 0 {
		errs = multierr.Append(errs,
			checkIntSum(tel, "otelcol_exporter_send_failed_"+datatype, sendFailed, exporterAttrs))
	}
	return errs
}

func checkExporterEnqueueFailed(tel *Telemetry, exporter component.ID, datatype string, enqueueFailed int64) error {
	if enqueueFailed == 0 {
		return nil
	}
	exporterAttrs := attributesForExporterMetrics(exporter)
	return checkIntSum(tel, "otelcol_exporter_enqueue_failed_"+datatype, enqueueFailed, exporterAttrs)
}

func checkIntGauge(tel *Telemetry, metric string, expected int64, expectedAttrs attribute.Set) error {
	dp, err := getGaugeDataPoint[int64](tel, metric, expectedAttrs)
	if err != nil {
		return err
	}

	if dp.Value != expected {
		return fmt.Errorf("values for metric '%s' did not match, expected '%d' got '%d'", metric, expected, dp.Value)
	}

	return nil
}

func checkIntSum(tel *Telemetry, expectedMetric string, expected int64, expectedAttrs attribute.Set) error {
	dp, err := getSumDataPoint[int64](tel, expectedMetric, expectedAttrs)
	if err != nil {
		return err
	}

	if dp.Value != expected {
		return fmt.Errorf("values for metric '%s' did not match, expected '%d' got '%d'", expectedMetric, expected, dp.Value)
	}

	return nil
}

func getSumDataPoint[N int64 | float64](tel *Telemetry, expectedName string, expectedAttrs attribute.Set) (metricdata.DataPoint[N], error) {
	m, err := tel.GetMetric(expectedName)
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

func getGaugeDataPoint[N int64 | float64](tel *Telemetry, expectedName string, expectedAttrs attribute.Set) (metricdata.DataPoint[N], error) {
	m, err := tel.GetMetric(expectedName)
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
