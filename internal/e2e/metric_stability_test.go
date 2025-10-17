// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

func assertMetrics(t *testing.T, metricsAddr string, expectedMetrics map[string]bool) bool {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://%s/metrics", metricsAddr))
	if err != nil {
		return false
	}

	if resp.StatusCode != http.StatusOK {
		return false
	}

	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	parser := expfmt.NewTextParser(model.UTF8Validation)
	parsed, err := parser.TextToMetricFamilies(reader)
	if err != nil {
		return false
	}

	for metricName, metricFamily := range parsed {
		if _, ok := expectedMetrics[metricName]; ok {
			expectedMetrics[metricName] = true
			assert.GreaterOrEqual(t, len(metricFamily.Metric), 1,
				"metric %s should have at least one data point", metricName)
		}
	}

	for metricName, found := range expectedMetrics {
		if !found {
			t.Logf("expected metric %s was not found", metricName)
			return false
		}
	}

	return true
}

func TestMetricStability(t *testing.T) {
	tests := []struct {
		name            string
		configFile      string
		expectedMetrics map[string]bool
		otelPort        string
		metricsPort     string
	}{
		{
			name:       "No metric readers (default)",
			configFile: "metric_stability_test_no_readers.yaml",
			expectedMetrics: map[string]bool{
				// Process metrics
				"otelcol_process_uptime":                         false,
				"otelcol_process_cpu_seconds":                    false,
				"otelcol_process_memory_rss":                     false,
				"otelcol_process_runtime_heap_alloc_bytes":       false,
				"otelcol_process_runtime_total_alloc_bytes":      false,
				"otelcol_process_runtime_total_sys_memory_bytes": false,

				// Batch processor metrics
				"otelcol_processor_batch_batch_send_size":       false,
				"otelcol_processor_batch_batch_send_size_bytes": false,
				"otelcol_processor_batch_metadata_cardinality":  false,
				"otelcol_processor_batch_timeout_trigger_send":  false,

				// HTTP server metrics
				"http_server_request_body_size":  false,
				"http_server_request_duration":   false,
				"http_server_response_body_size": false,

				// Exporter metrics
				"otelcol_exporter_sent_metric_points":        false,
				"otelcol_exporter_send_failed_metric_points": false,
				"otelcol_exporter_sent_spans":                false,
				"otelcol_exporter_send_failed_spans":         false,
				"otelcol_exporter_sent_log_records":          false,
				"otelcol_exporter_send_failed_log_records":   false,

				// Receiver metrics
				"otelcol_receiver_accepted_metric_points": false,
				"otelcol_receiver_refused_metric_points":  false,
				"otelcol_receiver_accepted_spans":         false,
				"otelcol_receiver_refused_spans":          false,
				"otelcol_receiver_accepted_log_records":   false,
				"otelcol_receiver_refused_log_records":    false,

				// Other metrics
				"promhttp_metric_handler_errors_total": false,
				"target_info":                          false,
			},
			otelPort:    getFreePort(t),
			metricsPort: "8888", // default metrics port
		},
		{
			name:       "Metric readers",
			configFile: "metric_stability_test_readers.yaml",
			expectedMetrics: map[string]bool{
				// Process metrics
				"otelcol_process_uptime_seconds_total":            false,
				"otelcol_process_cpu_seconds_total":               false,
				"otelcol_process_memory_rss_bytes":                false,
				"otelcol_process_runtime_heap_alloc_bytes":        false,
				"otelcol_process_runtime_total_alloc_bytes_total": false,
				"otelcol_process_runtime_total_sys_memory_bytes":  false,

				// Batch processor metrics
				"otelcol_processor_batch_batch_send_size":            false,
				"otelcol_processor_batch_batch_send_size_bytes":      false,
				"otelcol_processor_batch_metadata_cardinality":       false,
				"otelcol_processor_batch_timeout_trigger_send_total": false,

				// HTTP server metrics
				"http_server_request_body_size_bytes":  false,
				"http_server_request_duration_seconds": false,
				"http_server_response_body_size_bytes": false,

				// Exporter metrics - Metrics
				"otelcol_exporter_sent_metric_points_total":        false,
				"otelcol_exporter_send_failed_metric_points_total": false,

				// Exporter metrics - Traces
				"otelcol_exporter_sent_spans_total":        false,
				"otelcol_exporter_send_failed_spans_total": false,

				// Exporter metrics - Logs
				"otelcol_exporter_sent_log_records_total":        false,
				"otelcol_exporter_send_failed_log_records_total": false,

				// Receiver metrics
				"otelcol_receiver_accepted_metric_points_total": false,
				"otelcol_receiver_refused_metric_points_total":  false,

				// Receiver metrics - Traces
				"otelcol_receiver_accepted_spans_total": false,
				"otelcol_receiver_refused_spans_total":  false,

				// Receiver metrics - Logs
				"otelcol_receiver_accepted_log_records_total": false,
				"otelcol_receiver_refused_log_records_total":  false,

				// Other metrics
				"promhttp_metric_handler_errors_total": false,
				"target_info":                          false,
			},
			otelPort:    getFreePort(t),
			metricsPort: getFreePort(t),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testMetricStability(t, test.configFile, test.expectedMetrics, test.metricsPort, test.otelPort)
		})
	}
}

func testMetricStability(t *testing.T, configFile string, expectedMetrics map[string]bool, metricsPort, otelPort string) {
	t.Setenv("METRICS_PORT", metricsPort)
	t.Setenv("OTEL_PORT", otelPort)

	collector, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: func() (otelcol.Factories, error) {
			return otelcol.Factories{
				Receivers:  map[component.Type]receiver.Factory{otlpreceiver.NewFactory().Type(): otlpreceiver.NewFactory()},
				Processors: map[component.Type]processor.Factory{batchprocessor.NewFactory().Type(): batchprocessor.NewFactory()},
				Exporters:  map[component.Type]exporter.Factory{debugexporter.NewFactory().Type(): debugexporter.NewFactory()},
				Telemetry:  otelconftelemetry.NewFactory(),
			}, nil
		},
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filepath.Join("testdata", configFile)},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					yamlprovider.NewFactory(),
					envprovider.NewFactory(),
				},
			},
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := collector.Run(ctx)
		if err != nil {
			t.Logf("Collector stopped with error: %v", err)
		}
	}()

	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/metrics", metricsPort))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 5*time.Second, 100*time.Millisecond, "collector failed to start")

	for range 5 {
		sendTestData(t, otelPort)
	}

	require.Eventually(t, func() bool {
		return assertMetrics(t, "localhost:"+metricsPort, expectedMetrics)
	}, 10*time.Second, 200*time.Millisecond, "failed to verify metrics")
}

func sendTestData(t *testing.T, otelPort string) {
	require.NoError(t, sendTestMetrics(otelPort))
	require.NoError(t, sendTestTraces(otelPort))
	require.NoError(t, sendTestLogs(otelPort))
}

func sendTestMetrics(otelPort string) error {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	metric := sm.Metrics().AppendEmpty()
	metric.SetName("test_metric")
	metric.SetDescription("test metric")
	metric.SetUnit("1")
	dp := metric.SetEmptyGauge().DataPoints().AppendEmpty()
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	dp.SetDoubleValue(42.0)

	client := &http.Client{}

	metricsMarshaler := pmetric.ProtoMarshaler{}
	metricsBytes, err := metricsMarshaler.MarshalMetrics(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%s/v1/metrics", otelPort), bytes.NewReader(metricsBytes))
	if err != nil {
		return fmt.Errorf("failed to create metrics request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	resp.Body.Close()

	return nil
}

func sendTestTraces(otelPort string) error {
	traces := ptrace.NewTraces()
	rs := traces.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetName("test_span")
	now := time.Now()
	span.SetStartTimestamp(pcommon.NewTimestampFromTime(now))
	span.SetEndTimestamp(pcommon.NewTimestampFromTime(now))
	span.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})

	client := &http.Client{}

	tracesMarshaler := ptrace.ProtoMarshaler{}
	tracesBytes, err := tracesMarshaler.MarshalTraces(traces)
	if err != nil {
		return fmt.Errorf("failed to marshal traces: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%s/v1/traces", otelPort), bytes.NewReader(tracesBytes))
	if err != nil {
		return fmt.Errorf("failed to create traces request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send traces: %w", err)
	}
	resp.Body.Close()

	return nil
}

func sendTestLogs(otelPort string) error {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	log := sl.LogRecords().AppendEmpty()
	log.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	log.SetSeverityText("INFO")
	log.SetSeverityNumber(plog.SeverityNumberInfo)
	log.Body().SetStr("test log message")

	client := &http.Client{}

	logsMarshaler := plog.ProtoMarshaler{}
	logsBytes, err := logsMarshaler.MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%s/v1/logs", otelPort), bytes.NewReader(logsBytes))
	if err != nil {
		return fmt.Errorf("failed to create logs request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-protobuf")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs: %w", err)
	}
	resp.Body.Close()

	return nil
}

func getFreePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not get free port: %v", err)
	}
	defer l.Close()
	return strconv.Itoa(l.Addr().(*net.TCPAddr).Port)
}
