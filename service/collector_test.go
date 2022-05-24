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

// Package collector handles the command-line, configuration, and runs the OC collector.
package service

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/mapconverter/overwritepropertiesmapconverter"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/service/featuregate"
)

func TestStateString(t *testing.T) {
	assert.Equal(t, "Starting", Starting.String())
	assert.Equal(t, "Running", Running.String())
	assert.Equal(t, "Closing", Closing.String())
	assert.Equal(t, "Closed", Closed.String())
	assert.Equal(t, "UNKNOWN", State(13).String())
}

func TestCollectorStartAsGoRoutine(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
		"yaml:service::telemetry::metrics::address: " + testutil.GetAvailableLocalAddress(t),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	}
	col, err := New(set)
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()
	col.Shutdown()
	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorCancelContext(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
		"yaml:service::telemetry::metrics::address: " + testutil.GetAvailableLocalAddress(t),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	}
	col, err := New(set)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	wg := startCollector(ctx, t, col)

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	cancel()
	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorReportError(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.service.host.ReportFatalError(errors.New("err2"))

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorFailedShutdown(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      &mockColTelemetry{},
	})
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.EqualError(t, col.Run(context.Background()), "failed to shutdown collector telemetry: err1")
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func testCollectorStartHelper(t *testing.T, telemetry collectorTelemetryExporter) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)
	var once sync.Once
	loggingHookCalled := false
	hook := func(entry zapcore.Entry) error {
		once.Do(func() {
			loggingHookCalled = true
		})
		return nil
	}

	metricsAddr := testutil.GetAvailableLocalAddress(t)
	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
	})
	cfgSet.MapConverters = append([]config.MapConverter{
		overwritepropertiesmapconverter.New(
			[]string{"service.telemetry.metrics.address=" + metricsAddr},
		)},
		cfgSet.MapConverters...,
	)
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		LoggingOptions: []zap.Option{zap.Hooks(hook)},
		telemetry:      telemetry,
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)
	assert.Equal(t, col.telemetry.Logger, col.GetLogger())
	assert.True(t, loggingHookCalled)

	// All labels added to all collector metrics by default are listed below.
	// These labels are hard coded here in order to avoid inadvertent changes:
	// at this point changing labels should be treated as a breaking changing
	// and requires a good justification. The reason is that changes to metric
	// names or labels can break alerting, dashboards, etc that are used to
	// monitor the Collector in production deployments.
	mandatoryLabels := []string{
		"service_instance_id",
	}
	assertMetrics(t, metricsAddr, mandatoryLabels)

	assertZPages(t)
	col.signalsChannel <- syscall.SIGTERM

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorStartWithOpenCensusMetrics(t *testing.T) {
	testCollectorStartHelper(t, newColTelemetry(featuregate.NewRegistry()))
}

func TestCollectorStartWithOpenTelemetryMetrics(t *testing.T) {
	colTel := newColTelemetry(featuregate.NewRegistry())
	colTel.registry.Apply(map[string]bool{
		useOtelForInternalMetricsfeatureGateID: true,
	})
	testCollectorStartHelper(t, colTel)
}

func TestCollectorShutdownBeforeRun(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-config.yaml"),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	}
	col, err := New(set)
	require.NoError(t, err)

	// Calling shutdown before collector is running should cause it to return quickly
	require.NotPanics(t, func() { col.Shutdown() })

	wg := startCollector(context.Background(), t, col)

	col.Shutdown()
	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorClosedStateOnStartUpError(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	cfgSet := newDefaultConfigProviderSettings([]string{
		filepath.Join("testdata", "otelcol-invalid.yaml"),
	})
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	// Load a bad config causing startup to fail
	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	}
	col, err := New(set)
	require.NoError(t, err)

	// Expect run to error
	require.Error(t, col.Run(context.Background()))

	// Expect state to be closed
	assert.Equal(t, Closed, col.GetState())
}

type mockColTelemetry struct{}

func (tel *mockColTelemetry) init(*Collector) error {
	return nil
}

func (tel *mockColTelemetry) shutdown() error {
	return errors.New("err1")
}

func assertMetrics(t *testing.T, metricsAddr string, mandatoryLabels []string) {
	client := &http.Client{}
	resp, err := client.Get("http://" + metricsAddr + "/metrics")
	require.NoError(t, err)

	t.Cleanup(func() {
		assert.NoError(t, resp.Body.Close())
	})
	reader := bufio.NewReader(resp.Body)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(reader)
	require.NoError(t, err)

	prefix := "otelcol"
	for metricName, metricFamily := range parsed {
		// require is used here so test fails with a single message.
		require.True(
			t,
			strings.HasPrefix(metricName, prefix),
			"expected prefix %q but string starts with %q",
			prefix,
			metricName[:len(prefix)+1]+"...")

		for _, metric := range metricFamily.Metric {
			var labelNames []string
			for _, labelPair := range metric.Label {
				labelNames = append(labelNames, *labelPair.Name)
			}

			for _, mandatoryLabel := range mandatoryLabels {
				// require is used here so test fails with a single message.
				require.Contains(t, labelNames, mandatoryLabel, "mandatory label %q not present", mandatoryLabel)
			}
		}
	}
}

func assertZPages(t *testing.T) {
	paths := []string{
		"/debug/tracez",
		// TODO: enable this when otel-metrics is used and this page is available.
		// "/debug/rpcz",
		"/debug/pipelinez",
		"/debug/servicez",
		"/debug/extensionz",
	}

	const defaultZPagesPort = "55679"

	testZPagePathFn := func(t *testing.T, path string) {
		client := &http.Client{}
		resp, err := client.Get("http://localhost:" + defaultZPagesPort + path)
		if !assert.NoError(t, err, "error retrieving zpage at %q", path) {
			return
		}
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unsuccessful zpage %q GET", path)
		assert.NoError(t, resp.Body.Close())
	}

	for _, path := range paths {
		testZPagePathFn(t, path)
	}
}

func startCollector(ctx context.Context, t *testing.T, col *Collector) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, col.Run(ctx))
	}()
	return wg
}
