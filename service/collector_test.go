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
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/testutil"
)

func TestStateString(t *testing.T) {
	assert.Equal(t, "Starting", Starting.String())
	assert.Equal(t, "Running", Running.String())
	assert.Equal(t, "Closing", Closing.String())
	assert.Equal(t, "Closed", Closed.String())
	assert.Equal(t, "UNKNOWN", State(13).String())
}

func TestCollectorStartAsGoRoutine(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
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
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
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

type mockCfgProvider struct {
	ConfigProvider
	watcher chan error
}

func (p mockCfgProvider) Watch() <-chan error {
	return p.watcher
}

func TestCollectorStateAfterConfigChange(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	provider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)

	watcher := make(chan error, 1)
	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: &mockCfgProvider{ConfigProvider: provider, watcher: watcher},
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	watcher <- nil

	assert.Eventually(t, func() bool {
		return Running == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorReportError(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
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

func TestCollectorSendSignal(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
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

	col.signalsChannel <- syscall.SIGTERM

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorFailedShutdown(t *testing.T) {
	t.Skip("This test was using telemetry shutdown failure, switch to use a component that errors on shutdown.")
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.NewDefaultBuildInfo(),
		Factories:      factories,
		ConfigProvider: cfgProvider,
		telemetry:      newColTelemetry(featuregate.NewRegistry()),
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

// mapConverter applies extraMap of config settings. Useful for overriding the config
// for testing purposes. Keys must use "::" delimiter between levels.
type mapConverter struct {
	extraMap map[string]interface{}
}

func (m mapConverter) Convert(ctx context.Context, conf *confmap.Conf) error {
	return conf.Merge(confmap.NewFromStringMap(m.extraMap))
}

type labelState int

const (
	labelNotPresent labelState = iota
	labelSpecificValue
	labelAnyValue
)

type labelValue struct {
	label string
	state labelState
}

type ownMetricsTestCase struct {
	name                string
	userDefinedResource map[string]*string
	expectedLabels      map[string]labelValue
}

var testResourceAttrValue = "resource_attr_test_value"
var testInstanceID = "test_instance_id"
var testServiceVersion = "2022-05-20"

func ownMetricsTestCases(version string) []ownMetricsTestCase {
	return []ownMetricsTestCase{{
		name:                "no resource",
		userDefinedResource: nil,
		// All labels added to all collector metrics by default are listed below.
		// These labels are hard coded here in order to avoid inadvertent changes:
		// at this point changing labels should be treated as a breaking changing
		// and requires a good justification. The reason is that changes to metric
		// names or labels can break alerting, dashboards, etc that are used to
		// monitor the Collector in production deployments.
		expectedLabels: map[string]labelValue{
			"service_instance_id": {state: labelAnyValue},
			"service_version":     {label: version, state: labelSpecificValue},
		},
	},
		{
			name: "resource with custom attr",
			userDefinedResource: map[string]*string{
				"custom_resource_attr": &testResourceAttrValue,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id":  {state: labelAnyValue},
				"service_version":      {label: version, state: labelSpecificValue},
				"custom_resource_attr": {label: "resource_attr_test_value", state: labelSpecificValue},
			},
		},
		{
			name: "override service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": &testInstanceID,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {label: "test_instance_id", state: labelSpecificValue},
				"service_version":     {label: version, state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": nil, // nil value in config is used to suppress attributes.
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelNotPresent},
				"service_version":     {label: version, state: labelSpecificValue},
			},
		},
		{
			name: "override service.version",
			userDefinedResource: map[string]*string{
				"service.version": &testServiceVersion,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_version":     {label: "2022-05-20", state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.version",
			userDefinedResource: map[string]*string{
				"service.version": nil, // nil value in config is used to suppress attributes.
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_version":     {state: labelNotPresent},
			},
		}}
}

func testCollectorStartHelper(t *testing.T, telemetry *telemetryInitializer, tc ownMetricsTestCase) {
	factories, err := componenttest.NopFactories()
	zpagesExt := zpagesextension.NewFactory()
	factories.Extensions[zpagesExt.Type()] = zpagesExt
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
	// Prepare config properties to be merged with the main config.
	extraCfgAsProps := map[string]interface{}{
		// Setup the zpages extension.
		"extensions::zpages":  nil,
		"service::extensions": "nop, zpages",
		// Set the metrics address to expose own metrics on.
		"service::telemetry::metrics::address": metricsAddr,
	}
	// Also include resource attributes under the service::telemetry::resource key.
	for k, v := range tc.userDefinedResource {
		extraCfgAsProps["service::telemetry::resource::"+k] = v
	}

	cfgSet := newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")})
	cfgSet.ResolverSettings.Converters = append([]confmap.Converter{
		mapConverter{extraCfgAsProps}},
		cfgSet.ResolverSettings.Converters...,
	)
	cfgProvider, err := NewConfigProvider(cfgSet)
	require.NoError(t, err)

	col, err := New(CollectorSettings{
		BuildInfo:      component.BuildInfo{Version: "test version"},
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
	assert.True(t, loggingHookCalled)

	assertMetrics(t, metricsAddr, tc.expectedLabels)

	assertZPages(t)

	col.Shutdown()

	wg.Wait()
	assert.Equal(t, Closed, col.GetState())
}

func TestCollectorStartWithOpenCensusMetrics(t *testing.T) {
	for _, tc := range ownMetricsTestCases("test version") {
		t.Run(tc.name, func(t *testing.T) {
			testCollectorStartHelper(t, newColTelemetry(featuregate.NewRegistry()), tc)
		})
	}
}

func TestCollectorStartWithOpenTelemetryMetrics(t *testing.T) {
	for _, tc := range ownMetricsTestCases("test version") {
		t.Run(tc.name, func(t *testing.T) {
			registry := featuregate.NewRegistry()
			registerInternalMetricFeatureGate(registry)
			colTel := newColTelemetry(registry)
			require.NoError(t, colTel.registry.Apply(map[string]bool{useOtelForInternalMetricsfeatureGateID: true}))
			testCollectorStartHelper(t, colTel, tc)
		})
	}
}

func TestCollectorStartWithTraceContextPropagation(t *testing.T) {
	tests := []struct {
		file        string
		errExpected bool
	}{
		{file: "otelcol-invalidprop.yaml", errExpected: true},
		{file: "otelcol-nop.yaml", errExpected: false},
		{file: "otelcol-validprop.yaml", errExpected: false},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			factories, err := componenttest.NopFactories()
			require.NoError(t, err)

			cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", tt.file)}))
			require.NoError(t, err)

			set := CollectorSettings{
				BuildInfo:      component.NewDefaultBuildInfo(),
				Factories:      factories,
				ConfigProvider: cfgProvider,
				telemetry:      newColTelemetry(featuregate.NewRegistry()),
			}

			col, err := New(set)
			require.NoError(t, err)

			if tt.errExpected {
				require.Error(t, col.Run(context.Background()))
				assert.Equal(t, Closed, col.GetState())
			} else {
				wg := startCollector(context.Background(), t, col)
				col.Shutdown()
				wg.Wait()
				assert.Equal(t, Closed, col.GetState())
			}
		})
	}
}

func TestCollectorRun(t *testing.T) {
	tests := []struct {
		file string
	}{
		{file: "otelcol-nometrics.yaml"},
		{file: "otelcol-noaddress.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			factories, err := componenttest.NopFactories()
			require.NoError(t, err)

			cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", tt.file)}))
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

			col.Shutdown()
			wg.Wait()
			assert.Equal(t, Closed, col.GetState())
		})
	}
}

func TestCollectorShutdownBeforeRun(t *testing.T) {
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
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
	factories, err := componenttest.NopFactories()
	require.NoError(t, err)

	cfgProvider, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-invalid.yaml")}))
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

func assertMetrics(t *testing.T, metricsAddr string, expectedLabels map[string]labelValue) {
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
			labelMap := map[string]string{}
			for _, labelPair := range metric.Label {
				labelMap[*labelPair.Name] = *labelPair.Value
			}

			for k, v := range expectedLabels {
				switch v.state {
				case labelNotPresent:
					_, present := labelMap[k]
					assert.Falsef(t, present, "label %q must not be present", k)
				case labelSpecificValue:
					require.Equalf(t, v.label, labelMap[k], "mandatory label %q value mismatch", k)
				case labelAnyValue:
					assert.NotEmptyf(t, labelMap[k], "mandatory label %q not present", k)
				}
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
