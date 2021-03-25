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
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/processor/attributesprocessor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/receiver/jaegerreceiver"
	"go.opentelemetry.io/collector/service/defaultcomponents"
	"go.opentelemetry.io/collector/service/internal/builder"
	"go.opentelemetry.io/collector/testutil"
)

func TestApplication_Start(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	loggingHookCalled := false
	hook := func(entry zapcore.Entry) error {
		loggingHookCalled = true
		return nil
	}

	app, err := New(Parameters{Factories: factories, ApplicationStartInfo: component.DefaultApplicationStartInfo(), LoggingOptions: []zap.Option{zap.Hooks(hook)}})
	require.NoError(t, err)
	assert.Equal(t, app.rootCmd, app.Command())

	const testPrefix = "a_test"
	metricsPort := testutil.GetAvailablePort(t)
	healthCheckPortStr := strconv.FormatUint(uint64(testutil.GetAvailablePort(t)), 10)
	app.rootCmd.SetArgs([]string{
		"--config=testdata/otelcol-config.yaml",
		"--metrics-addr=localhost:" + strconv.FormatUint(uint64(metricsPort), 10),
		"--metrics-prefix=" + testPrefix,
		"--set=extensions.health_check.port=" + healthCheckPortStr,
	})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.NoError(t, app.Run())
	}()

	assert.Equal(t, Starting, <-app.GetStateChannel())
	assert.Equal(t, Running, <-app.GetStateChannel())
	require.True(t, isAppAvailable(t, "http://localhost:"+healthCheckPortStr))
	assert.Equal(t, app.logger, app.GetLogger())
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

	app.signalsChannel <- syscall.SIGTERM
	<-appDone
	assert.Equal(t, Closing, <-app.GetStateChannel())
	assert.Equal(t, Closed, <-app.GetStateChannel())
}

type mockAppTelemetry struct{}

func (tel *mockAppTelemetry) init(chan<- error, uint64, *zap.Logger) error {
	return nil
}

func (tel *mockAppTelemetry) shutdown() error {
	return errors.New("err1")
}

func TestApplication_ReportError(t *testing.T) {
	// use a mock AppTelemetry struct to return an error on shutdown
	preservedAppTelemetry := applicationTelemetry
	applicationTelemetry = &mockAppTelemetry{}
	defer func() { applicationTelemetry = preservedAppTelemetry }()

	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	app, err := New(Parameters{Factories: factories, ApplicationStartInfo: component.DefaultApplicationStartInfo()})
	require.NoError(t, err)

	app.rootCmd.SetArgs([]string{"--config=testdata/otelcol-config-minimal.yaml"})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		assert.EqualError(t, app.Run(), "failed to shutdown application telemetry: err1")
	}()

	assert.Equal(t, Starting, <-app.GetStateChannel())
	assert.Equal(t, Running, <-app.GetStateChannel())
	app.service.ReportFatalError(errors.New("err2"))
	<-appDone
	assert.Equal(t, Closing, <-app.GetStateChannel())
	assert.Equal(t, Closed, <-app.GetStateChannel())
}

func TestApplication_StartAsGoRoutine(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	params := Parameters{
		ApplicationStartInfo: component.DefaultApplicationStartInfo(),
		ConfigFactory: func(_ *cobra.Command, factories component.Factories) (*configmodels.Config, error) {
			return constructMimumalOpConfig(t, factories), nil
		},
		Factories: factories,
	}
	app, err := New(params)
	require.NoError(t, err)
	app.Command().SetArgs([]string{
		"--metrics-level=NONE",
	})

	appDone := make(chan struct{})
	go func() {
		defer close(appDone)
		appErr := app.Run()
		if appErr != nil {
			err = appErr
		}
	}()

	assert.Equal(t, Starting, <-app.GetStateChannel())
	assert.Equal(t, Running, <-app.GetStateChannel())

	app.Shutdown()
	app.Shutdown()
	<-appDone
	assert.Equal(t, Closing, <-app.GetStateChannel())
	assert.Equal(t, Closed, <-app.GetStateChannel())
}

// isAppAvailable checks if the healthcheck server at the given endpoint is
// returning `available`.
func isAppAvailable(t *testing.T, healthCheckEndPoint string) bool {
	client := &http.Client{}
	resp, err := client.Get(healthCheckEndPoint)
	require.NoError(t, err)

	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
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

func TestSetFlag(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)
	params := Parameters{
		Factories: factories,
	}
	t.Run("unknown_component", func(t *testing.T) {
		app, err := New(params)
		require.NoError(t, err)
		err = app.rootCmd.ParseFlags([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.doesnotexist.timeout=2s",
		})
		require.NoError(t, err)
		cfg, err := FileLoaderConfigFactory(app.rootCmd, factories)
		require.Error(t, err)
		require.Nil(t, cfg)

	})
	t.Run("component_not_added_to_pipeline", func(t *testing.T) {
		app, err := New(params)
		require.NoError(t, err)
		err = app.rootCmd.ParseFlags([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.batch/foo.timeout=2s",
		})
		require.NoError(t, err)
		cfg, err := FileLoaderConfigFactory(app.rootCmd, factories)
		require.NoError(t, err)
		assert.NotNil(t, cfg)
		err = cfg.Validate()
		require.NoError(t, err)

		var processors []string
		for k := range cfg.Processors {
			processors = append(processors, k)
		}
		sort.Strings(processors)
		// batch/foo is not added to the pipeline
		assert.Equal(t, []string{"attributes", "batch", "batch/foo"}, processors)
		assert.Equal(t, []string{"attributes", "batch"}, cfg.Service.Pipelines["traces"].Processors)
	})
	t.Run("ok", func(t *testing.T) {
		app, err := New(params)
		require.NoError(t, err)

		err = app.rootCmd.ParseFlags([]string{
			"--config=testdata/otelcol-config.yaml",
			"--set=processors.batch.timeout=2s",
			// Arrays are overridden and object arrays cannot be indexed
			// this creates actions array of size 1
			"--set=processors.attributes.actions.key=foo",
			"--set=processors.attributes.actions.value=bar",
			"--set=receivers.jaeger.protocols.grpc.endpoint=localhost:12345",
			"--set=extensions.health_check.port=8080",
		})
		require.NoError(t, err)
		cfg, err := FileLoaderConfigFactory(app.rootCmd, factories)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		err = cfg.Validate()
		require.NoError(t, err)

		assert.Equal(t, 2, len(cfg.Processors))
		batch := cfg.Processors["batch"].(*batchprocessor.Config)
		assert.Equal(t, time.Second*2, batch.Timeout)
		jaeger := cfg.Receivers["jaeger"].(*jaegerreceiver.Config)
		assert.Equal(t, "localhost:12345", jaeger.GRPC.NetAddr.Endpoint)
		attributes := cfg.Processors["attributes"].(*attributesprocessor.Config)
		require.Equal(t, 1, len(attributes.Actions))
		assert.Equal(t, "foo", attributes.Actions[0].Key)
		assert.Equal(t, "bar", attributes.Actions[0].Value)
	})
}

func TestSetFlag_component_does_not_exist(t *testing.T) {
	factories, err := defaultcomponents.Components()
	require.NoError(t, err)

	cmd := &cobra.Command{}
	addSetFlag(cmd.Flags())
	fs := &flag.FlagSet{}
	builder.Flags(fs)
	cmd.Flags().AddGoFlagSet(fs)
	cmd.ParseFlags([]string{
		"--config=testdata/otelcol-config.yaml",
		"--set=processors.batch.timeout=2s",
		// Arrays are overridden and object arrays cannot be indexed
		// this creates actions array of size 1
		"--set=processors.attributes.actions.key=foo",
		"--set=processors.attributes.actions.value=bar",
		"--set=receivers.jaeger.protocols.grpc.endpoint=localhost:12345",
	})
	cfg, err := FileLoaderConfigFactory(cmd, factories)
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func constructMimumalOpConfig(t *testing.T, factories component.Factories) *configmodels.Config {
	configStr := `
receivers:
  otlp:
    protocols:
      grpc:
exporters:
  logging:
processors:
  batch:

extensions:

service:
  extensions:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
`
	v := config.NewViper()
	v.SetConfigType("yaml")
	v.ReadConfig(strings.NewReader(configStr))
	cfg, err := configparser.Load(config.ParserFromViper(v), factories)
	assert.NoError(t, err)
	err = cfg.Validate()
	assert.NoError(t, err)
	return cfg
}
