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
	"path"
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
)

// TestCollector_StartAsGoRoutine must be the first unit test on the file,
// to test for initialization without setting CLI flags.
func TestCollector_StartAsGoRoutine(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := collectorTelemetry
	collectorTelemetry = &colTelemetry{}
	defer func() { collectorTelemetry = preservedAppTelemetry }()

	factories, err := testcomponents.DefaultFactories()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: NewDefaultConfigProvider(path.Join("testdata", "otelcol-config.yaml"), nil),
	}
	col, err := New(set)
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		colErr := col.Run(context.Background())
		if colErr != nil {
			err = colErr
		}
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, time.Second*2, time.Millisecond*200)

	col.Shutdown()
	col.Shutdown()
	<-colDone
	assert.Eventually(t, func() bool {
		return Closed == col.GetState()
	}, time.Second*2, time.Millisecond*200)
}

func TestCollector_Start(t *testing.T) {
	factories, err := testcomponents.DefaultFactories()
	require.NoError(t, err)
	var once sync.Once
	loggingHookCalled := false
	hook := func(entry zapcore.Entry) error {
		once.Do(func() {
			loggingHookCalled = true
		})
		return nil
	}

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: NewDefaultConfigProvider(path.Join("testdata", "otelcol-config.yaml"), nil),
		LoggingOptions: []zap.Option{zap.Hooks(hook)},
	})
	require.NoError(t, err)

	metricsPort := testutil.GetAvailablePort(t)
	require.NoError(t, flags().Parse([]string{
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
	}))

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		assert.NoError(t, col.Run(context.Background()))
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, time.Second*2, time.Millisecond*200)
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
	assert.Eventually(t, func() bool {
		return Closed == col.GetState()
	}, time.Second*2, time.Millisecond*200)
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

	factories, err := testcomponents.DefaultFactories()
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: NewDefaultConfigProvider(path.Join("testdata", "otelcol-config.yaml"), nil),
	})
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		assert.EqualError(t, col.Run(context.Background()), "failed to shutdown collector telemetry: err1")
	}()

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, time.Second*2, time.Millisecond*200)
	col.service.ReportFatalError(errors.New("err2"))
	<-colDone
	assert.Eventually(t, func() bool {
		return Closed == col.GetState()
	}, time.Second*2, time.Millisecond*200)
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
