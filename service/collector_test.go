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
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/internal/testcomponents"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/service/parserprovider"
)

const configStr = `
receivers:
  otlp:
    protocols:
      grpc:
exporters:
  otlp:
    endpoint: "localhost:4317"
processors:
  batch:
extensions:
service:
  extensions:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
`

func TestCollector_Start(t *testing.T) {
	factories, err := testcomponents.DefaultComponents()
	require.NoError(t, err)

	loggingHookCalled := false
	hook := func(entry zapcore.Entry) error {
		loggingHookCalled = true
		return nil
	}

	col, err := New(CollectorSettings{
		BuildInfo:         component.NewDefaultBuildInfo(),
		Factories:         factories,
		ConfigMapProvider: parserprovider.NewFileMapProvider("testdata/otelcol-config.yaml"),
		LoggingOptions:    []zap.Option{zap.Hooks(hook)},
	})
	require.NoError(t, err)

	const testPrefix = "a_test"
	metricsPort := testutil.GetAvailablePort(t)
	require.NoError(t, flags().Parse([]string{
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
	}))

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		assert.NoError(t, col.Run(context.Background()))
	}()

	assert.Equal(t, Starting, <-col.GetStateChannel())
	assert.Equal(t, Running, <-col.GetStateChannel())
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
	assertMetrics(t, testPrefix, metricsPort, mandatoryLabels)

	assertZPages(t)

	// Trigger another configuration load.
	require.NoError(t, col.reloadService(context.Background()))

	col.signalsChannel <- syscall.SIGTERM
	<-colDone
	assert.Equal(t, Closing, <-col.GetStateChannel())
	assert.Equal(t, Closed, <-col.GetStateChannel())
}

type mockColTelemetry struct{}

func (tel *mockColTelemetry) init(chan<- error, uint64, *zap.Logger) error {
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

	factories, err := testcomponents.DefaultComponents()
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:         component.NewDefaultBuildInfo(),
		Factories:         factories,
		ConfigMapProvider: parserprovider.NewFileMapProvider("testdata/otelcol-config.yaml"),
	})
	require.NoError(t, err)

	colDone := make(chan struct{})
	go func() {
		defer close(colDone)
		assert.EqualError(t, col.Run(context.Background()), "failed to shutdown collector telemetry: err1")
	}()

	assert.Equal(t, Starting, <-col.GetStateChannel())
	assert.Equal(t, Running, <-col.GetStateChannel())
	col.service.ReportFatalError(errors.New("err2"))
	<-colDone
	assert.Equal(t, Closing, <-col.GetStateChannel())
	assert.Equal(t, Closed, <-col.GetStateChannel())
}

func TestCollector_StartAsGoRoutine(t *testing.T) {
	factories, err := testcomponents.DefaultComponents()
	require.NoError(t, err)

	set := CollectorSettings{
		BuildInfo:         component.NewDefaultBuildInfo(),
		Factories:         factories,
		ConfigMapProvider: parserprovider.NewInMemoryMapProvider(strings.NewReader(configStr)),
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

	assert.Equal(t, Starting, <-col.GetStateChannel())
	assert.Equal(t, Running, <-col.GetStateChannel())

	col.Shutdown()
	col.Shutdown()
	<-colDone
	assert.Equal(t, Closing, <-col.GetStateChannel())
	assert.Equal(t, Closed, <-col.GetStateChannel())
}

func assertMetrics(t *testing.T, prefix string, metricsPort uint16, mandatoryLabels []string) {
	client := &http.Client{}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	require.NoError(t, err)

	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(reader)
	require.NoError(t, err)

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

type errParserLoader struct {
	err error
}

func (epl *errParserLoader) Get(context.Context) (*config.Map, error) {
	return nil, epl.err
}

func (epl *errParserLoader) Close(context.Context) error {
	return nil
}

func TestCollector_reloadService(t *testing.T) {
	factories, err := testcomponents.DefaultComponents()
	require.NoError(t, err)
	ctx := context.Background()
	sentinelError := errors.New("sentinel error")

	tests := []struct {
		name           string
		parserProvider parserprovider.MapProvider
		service        *service
	}{
		{
			name:           "first_load_err",
			parserProvider: &errParserLoader{err: sentinelError},
		},
		{
			name:           "retire_service_ok_load_err",
			parserProvider: &errParserLoader{err: sentinelError},
			service: &service{
				telemetry:       componenttest.NewNopTelemetrySettings(),
				builtExporters:  builder.Exporters{},
				builtPipelines:  builder.BuiltPipelines{},
				builtReceivers:  builder.Receivers{},
				builtExtensions: builder.Extensions{},
			},
		},
		{
			name:           "retire_service_ok_load_ok",
			parserProvider: parserprovider.NewInMemoryMapProvider(strings.NewReader(configStr)),
			service: &service{
				telemetry:       componenttest.NewNopTelemetrySettings(),
				builtExporters:  builder.Exporters{},
				builtPipelines:  builder.BuiltPipelines{},
				builtReceivers:  builder.Receivers{},
				builtExtensions: builder.Extensions{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := Collector{
				set: CollectorSettings{
					ConfigMapProvider: tt.parserProvider,
					ConfigUnmarshaler: configunmarshaler.NewDefault(),
					Factories:         factories,
				},
				logger:         zap.NewNop(),
				tracerProvider: trace.NewNoopTracerProvider(),
				service:        tt.service,
			}

			if err = col.reloadService(ctx); err != nil {
				assert.ErrorIs(t, err, sentinelError)
				return
			}

			// If successful need to shutdown active service.
			assert.NoError(t, col.service.Shutdown(ctx))
		})
	}
}
