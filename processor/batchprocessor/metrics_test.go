// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package batchprocessor

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.opentelemetry.io/collector/processor/processortest"
)

type testTelemetry struct {
	promHandler   http.Handler
	meterProvider *sdkmetric.MeterProvider
}

type expectedMetrics struct {
	// processor_batch_batch_send_size_count
	// processor_batch_batch_send_size_bytes_count
	sendCount float64
	// processor_batch_batch_send_size_sum
	sendSizeSum float64
	// processor_batch_batch_send_size_bytes_sum
	sendSizeBytesSum float64
	// processor_batch_batch_size_trigger_send
	sizeTrigger float64
	// processor_batch_batch_timeout_trigger_send
	timeoutTrigger float64
}

func telemetryTest(t *testing.T, testFunc func(t *testing.T, tel testTelemetry)) {
	t.Run("WithOTel", func(t *testing.T) {
		testFunc(t, setupTelemetry(t))
	})
}

func setupTelemetry(t *testing.T) testTelemetry {
	telemetry := testTelemetry{}

	promReg := prometheus.NewRegistry()
	exporter, err := otelprom.New(otelprom.WithRegisterer(promReg), otelprom.WithoutUnits(), otelprom.WithoutScopeInfo(), otelprom.WithoutCounterSuffixes())
	require.NoError(t, err)

	telemetry.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource.Empty()),
		sdkmetric.WithReader(exporter),
		sdkmetric.WithView(batchViews()...),
	)

	telemetry.promHandler = promhttp.HandlerFor(promReg, promhttp.HandlerOpts{})

	t.Cleanup(func() { assert.NoError(t, telemetry.meterProvider.Shutdown(context.Background())) })

	return telemetry
}

func (tt *testTelemetry) NewProcessorCreateSettings() processor.CreateSettings {
	settings := processortest.NewNopCreateSettings()
	settings.MeterProvider = tt.meterProvider
	settings.ID = component.MustNewID("batch")

	return settings
}

func (tt *testTelemetry) assertMetrics(t *testing.T, expected expectedMetrics) {
	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	tt.promHandler.ServeHTTP(rr, req)

	var parser expfmt.TextParser
	metrics, err := parser.TextToMetricFamilies(rr.Body)
	require.NoError(t, err)

	if expected.sendSizeBytesSum > 0 {
		name := "processor_batch_batch_send_size_bytes"
		metric := tt.getMetric(t, name, io_prometheus_client.MetricType_HISTOGRAM, metrics)

		assertFloat(t, expected.sendSizeBytesSum, metric.GetHistogram().GetSampleSum(), name)
		assertFloat(t, expected.sendCount, float64(metric.GetHistogram().GetSampleCount()), name)

		tt.assertBoundaries(t,
			[]float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
				100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
				1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000, math.Inf(1)},
			metric.GetHistogram(),
			name,
		)
	}

	if expected.sendSizeSum > 0 {
		name := "processor_batch_batch_send_size"
		metric := tt.getMetric(t, name, io_prometheus_client.MetricType_HISTOGRAM, metrics)

		assertFloat(t, expected.sendSizeSum, metric.GetHistogram().GetSampleSum(), name)
		assertFloat(t, expected.sendCount, float64(metric.GetHistogram().GetSampleCount()), name)

		tt.assertBoundaries(t,
			[]float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000, math.Inf(1)},
			metric.GetHistogram(),
			name,
		)
	}

	if expected.sizeTrigger > 0 {
		name := "processor_batch_batch_size_trigger_send"
		metric := tt.getMetric(t, name, io_prometheus_client.MetricType_COUNTER, metrics)

		assertFloat(t, expected.sizeTrigger, metric.GetCounter().GetValue(), name)
	}

	if expected.timeoutTrigger > 0 {
		name := "processor_batch_timeout_trigger_send"
		metric := tt.getMetric(t, name, io_prometheus_client.MetricType_COUNTER, metrics)

		assertFloat(t, expected.timeoutTrigger, metric.GetCounter().GetValue(), name)
	}
}

func (tt *testTelemetry) assertBoundaries(t *testing.T, expected []float64, histogram *io_prometheus_client.Histogram, metric string) {
	var got []float64
	for _, bucket := range histogram.GetBucket() {
		got = append(got, bucket.GetUpperBound())
	}

	if len(expected) != len(got) {
		assert.Failf(t, "different boundaries size", "metric '%s' expected boundaries '%x' but got '%x'", metric, expected, got)
		return
	}

	for i := range expected {
		if math.Abs(expected[i]-got[i]) > 0.00001 {
			assert.Failf(t, "unexpected boundary", "boundary for metric '%s' did not match, expected '%f' got '%f'", metric, expected[i], got[i])
		}
	}

}

func (tt *testTelemetry) getMetric(t *testing.T, name string, mtype io_prometheus_client.MetricType, got map[string]*io_prometheus_client.MetricFamily) *io_prometheus_client.Metric {
	metricFamily, ok := got[name]
	require.True(t, ok, "expected metric '%s' not found", name)
	require.Equal(t, mtype, metricFamily.GetType())

	metric, err := getSingleMetric(metricFamily)
	require.NoError(t, err)

	return metric
}

func getSingleMetric(metric *io_prometheus_client.MetricFamily) (*io_prometheus_client.Metric, error) {
	if l := len(metric.Metric); l != 1 {
		return nil, fmt.Errorf("expected metric '%s' with one set of attributes, but found %d", metric.GetName(), l)
	}
	first := metric.Metric[0]

	if len(first.Label) != 1 || "processor" != first.Label[0].GetName() || "batch" != first.Label[0].GetValue() {
		return nil, fmt.Errorf("expected metric '%s' with a single `batch=processor` attribute but got '%s'", metric.GetName(), first.GetLabel())
	}

	return first, nil
}

func assertFloat(t *testing.T, expected, got float64, metric string) {
	if math.Abs(expected-got) > 0.00001 {
		assert.Failf(t, "unexpected metric value", "value for metric '%s' did not match, expected '%f' got '%f'", metric, expected, got)
	}
}

func batchViews() []sdkmetric.View {
	return []sdkmetric.View{
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: processorhelper.BuildCustomMetricName("batch", "batch_send_size")},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000, 100000},
			}},
		),
		sdkmetric.NewView(
			sdkmetric.Instrument{Name: processorhelper.BuildCustomMetricName("batch", "batch_send_size_bytes")},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{10, 25, 50, 75, 100, 250, 500, 750, 1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 20000, 30000, 50000,
					100_000, 200_000, 300_000, 400_000, 500_000, 600_000, 700_000, 800_000, 900_000,
					1000_000, 2000_000, 3000_000, 4000_000, 5000_000, 6000_000, 7000_000, 8000_000, 9000_000},
			}},
		),
	}
}
