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
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
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

// TestCollector_StartAsGoRoutine must be the first unit test on the file,
// to test for initialization without setting CLI flags.
func TestCollector_StartAsGoRoutine(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &colTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-config.yaml")},
			[]string{"service.telemetry.metrics.address=localhost:" + strconv.FormatUint(uint64(testutil.GetAvailablePort(t)), 10)}),
	}
	col, err := New(set)
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		require.NoError(t, col.Run(context.Background()))
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()
	col.Shutdown()
	<-colDone
	assert.Equal(t, Closed, col.GetState())
}

func testCollectorStartHelper(t *testing.T) {
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

	metricsPort := testutil.GetAvailablePort(t)
	col, err := New(CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		ConfigProvider: MustNewDefaultConfigProvider(
			[]string{filepath.Join("testdata", "otelcol-config.yaml")},
			[]string{"service.telemetry.metrics.address=localhost:" + strconv.FormatUint(uint64(metricsPort), 10)}),
		LoggingOptions: []zap.Option{zap.Hooks(hook)},
	})
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		require.NoError(t, col.Run(context.Background()))
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)
	assert.Equal(t, col.logger, col.GetLogger())
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
	assertMetrics(t, metricsPort, mandatoryLabels)

	assertZPages(t)
	col.signalsChannel <- syscall.SIGTERM

	<-colDone
	assert.Equal(t, Closed, col.GetState())
}

// as telemetry instance is initialized only once, we need to reset it before each test so the metrics endpoint can
// have correct handler spawned
func resetCollectorTelemetry() {
	collectorTelemetry = &colTelemetry{}
}

func TestCollector_Start(t *testing.T) {
	resetCollectorTelemetry()
	testCollectorStartHelper(t)
}

func TestCollector_StartWithOtelInternalMetrics(t *testing.T) {
	resetCollectorTelemetry()
	originalFlag := featuregate.IsEnabled(useOtelForInternalMetricsfeatureGateID)
	defer func() {
		featuregate.Apply(map[string]bool{
			useOtelForInternalMetricsfeatureGateID: originalFlag,
		})
	}()
	featuregate.Apply(map[string]bool{
		useOtelForInternalMetricsfeatureGateID: true,
	})
	testCollectorStartHelper(t)
}

// TestCollector_ShutdownNoop verifies that shutdown can be called even if a collector
// has yet to be started and it will execute without error.
func TestCollector_ShutdownNoop(t *testing.T) {
	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-config.yaml")}, nil),
	}
	col, err := New(set)
	require.NoError(t, err)

	// Should be able to call Shutdown on an unstarted collector and nothing happens
	require.NotPanics(t, func() { col.Shutdown() })
}

func TestCollector_ShutdownBeforeRun(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &colTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-config.yaml")}, nil),
	}
	col, err := New(set)
	require.NoError(t, err)

	// Calling shutdown before collector is running should cause it to return quickly
	col.Shutdown()

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		require.NoError(t, col.Run(context.Background()))
	}()

	col.Shutdown()
	<-colDone
	assert.Equal(t, Closed, col.GetState())
}

// TestCollector_ClosedStateOnStartUpError tests the collector changes it's state to Closed when a startup error occurs
func TestCollector_ClosedStateOnStartUpError(t *testing.T) {
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &colTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	// Load a bad config causing startup to fail
	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-invalid.yaml")}, nil),
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

func TestCollector_ReportError(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &mockColTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-config.yaml")}, nil),
	})
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		assert.EqualError(t, col.Run(context.Background()), "failed to shutdown collector telemetry: err1")
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)
	col.service.ReportFatalError(errors.New("err2"))

	<-colDone
	assert.Equal(t, Closed, col.GetState())
}

// TestCollector_ContextCancel tests that the collector gracefully exits on context cancel
func TestCollector_ContextCancel(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &colTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.NewDefaultFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: factories,
		ConfigProvider: MustNewDefaultConfigProvider([]string{filepath.Join("testdata", "otelcol-config.yaml")},
			[]string{"service.telemetry.metrics.address=localhost:" + strconv.FormatUint(uint64(testutil.GetAvailablePort(t)), 10)}),
	}
	col, err := New(set)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		require.NoError(t, col.Run(ctx))
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	cancel()

	<-colDone
	assert.Equal(t, Closed, col.GetState())
}

func assertMetrics(t *testing.T, metricsPort uint16, mandatoryLabels []string) {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	require.NoError(t, err)

	defer resp.Body.Close()
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
