// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

func TestExporterFailureAttributesDetailed(t *testing.T) {
	otelPort := getFreePort(t)
	metricsPort := getFreePort(t)

	t.Setenv("METRICS_PORT", metricsPort)
	t.Setenv("OTEL_PORT", otelPort)

	collector, err := otelcol.NewCollector(otelcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: func() (otelcol.Factories, error) {
			return otelcol.Factories{
				Receivers:  map[component.Type]receiver.Factory{otlpreceiver.NewFactory().Type(): otlpreceiver.NewFactory()},
				Processors: map[component.Type]processor.Factory{batchprocessor.NewFactory().Type(): batchprocessor.NewFactory()},
				Exporters: map[component.Type]exporter.Factory{
					debugexporter.NewFactory().Type():    debugexporter.NewFactory(),
					otlphttpexporter.NewFactory().Type(): otlphttpexporter.NewFactory(),
				},
				Telemetry: otelconftelemetry.NewFactory(),
			}, nil
		},
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filepath.Join("testdata", "exporter_failure_attributes_test.yaml")},
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

	require.NoError(t, sendTestTraces(otelPort))
	require.NoError(t, sendTestMetrics(otelPort))
	require.NoError(t, sendTestLogs(otelPort))

	require.Eventually(t, func() bool {
		return verifyFailureAttributes(t, metricsPort, "otelcol_exporter_send_failed_spans") &&
			verifyFailureAttributes(t, metricsPort, "otelcol_exporter_send_failed_metric_points") &&
			verifyFailureAttributes(t, metricsPort, "otelcol_exporter_send_failed_log_records")
	}, 5*time.Second, 200*time.Millisecond, "metrics should have detailed failure attributes")
}

func verifyFailureAttributes(t *testing.T, metricsPort string, metricName string) bool {
	t.Helper()

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/metrics", metricsPort))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	parser := expfmt.NewTextParser(model.UTF8Validation)
	parsed, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return false
	}

	metricFamily, ok := parsed[metricName]
	if !ok {
		metricFamily, ok = parsed[metricName+"_total"]
	}
	if !ok || len(metricFamily.Metric) == 0 {
		return false
	}

	var foundMetric *dto.Metric
	for _, metric := range metricFamily.Metric {
		for _, label := range metric.Label {
			if label.GetName() == "exporter" && label.GetValue() == "otlphttp/fail" {
				foundMetric = metric
				break
			}
		}
		if foundMetric != nil {
			break
		}
	}
	if foundMetric == nil {
		return false
	}

	hasErrorType := false
	hasFailurePermanent := false
	for _, label := range foundMetric.Label {
		labelName := label.GetName()
		labelValue := label.GetValue()
		if labelName == "error_type" {
			if labelValue == "" {
				return false
			}
			hasErrorType = true
		}
		if labelName == "failure_permanent" {
			if labelValue != "true" && labelValue != "false" {
				return false
			}
			hasFailurePermanent = true
		}
	}

	return hasErrorType && hasFailurePermanent
}
