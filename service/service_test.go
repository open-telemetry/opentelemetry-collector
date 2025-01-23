// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/promtest"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
)

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

var (
	testResourceAttrValue = "resource_attr_test_value"
	testInstanceID        = "test_instance_id"
	testServiceVersion    = "2022-05-20"
	testServiceName       = "test name"
)

// prometheusToOtelConv is used to check that the expected resource labels exist as
// part of the otel resource attributes.
var prometheusToOtelConv = map[string]string{
	"service_instance_id": "service.instance.id",
	"service_name":        "service.name",
	"service_version":     "service.version",
}

const (
	metricsVersion = "test version"
	otelCommand    = "otelcoltest"
)

func ownMetricsTestCases() []ownMetricsTestCase {
	return []ownMetricsTestCase{
		{
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
				"service_name":        {label: otelCommand, state: labelSpecificValue},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "resource with custom attr",
			userDefinedResource: map[string]*string{
				"custom_resource_attr": &testResourceAttrValue,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id":  {state: labelAnyValue},
				"service_name":         {label: otelCommand, state: labelSpecificValue},
				"service_version":      {label: metricsVersion, state: labelSpecificValue},
				"custom_resource_attr": {label: "resource_attr_test_value", state: labelSpecificValue},
			},
		},
		{
			name: "override service.name",
			userDefinedResource: map[string]*string{
				"service.name": &testServiceName,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_name":        {label: testServiceName, state: labelSpecificValue},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.name",
			userDefinedResource: map[string]*string{
				"service.name": nil,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_name":        {state: labelNotPresent},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "override service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": &testInstanceID,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {label: "test_instance_id", state: labelSpecificValue},
				"service_name":        {label: otelCommand, state: labelSpecificValue},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "suppress service.instance.id",
			userDefinedResource: map[string]*string{
				"service.instance.id": nil, // nil value in config is used to suppress attributes.
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelNotPresent},
				"service_name":        {label: otelCommand, state: labelSpecificValue},
				"service_version":     {label: metricsVersion, state: labelSpecificValue},
			},
		},
		{
			name: "override service.version",
			userDefinedResource: map[string]*string{
				"service.version": &testServiceVersion,
			},
			expectedLabels: map[string]labelValue{
				"service_instance_id": {state: labelAnyValue},
				"service_name":        {label: otelCommand, state: labelSpecificValue},
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
				"service_name":        {label: otelCommand, state: labelSpecificValue},
				"service_version":     {state: labelNotPresent},
			},
		},
	}
}

var (
	nopType   = component.MustNewType("nop")
	wrongType = component.MustNewType("wrong")
)

func TestServiceGetFactory(t *testing.T) {
	set := newNopSettings()
	srv, err := New(context.Background(), set, newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	assert.Nil(t, srv.host.GetFactory(component.KindReceiver, wrongType))
	assert.Equal(t, srv.host.Receivers.Factory(nopType), srv.host.GetFactory(component.KindReceiver, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindProcessor, wrongType))
	assert.Equal(t, srv.host.Processors.Factory(nopType), srv.host.GetFactory(component.KindProcessor, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindExporter, wrongType))
	assert.Equal(t, srv.host.Exporters.Factory(nopType), srv.host.GetFactory(component.KindExporter, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindConnector, wrongType))
	assert.Equal(t, srv.host.Connectors.Factory(nopType), srv.host.GetFactory(component.KindConnector, nopType))

	assert.Nil(t, srv.host.GetFactory(component.KindExtension, wrongType))
	assert.Equal(t, srv.host.Extensions.Factory(nopType), srv.host.GetFactory(component.KindExtension, nopType))

	// Try retrieve non existing component.Kind.
	assert.Nil(t, srv.host.GetFactory(42, nopType))
}

func TestServiceGetExtensions(t *testing.T) {
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	extMap := srv.host.GetExtensions()

	assert.Len(t, extMap, 1)
	assert.Contains(t, extMap, component.NewID(nopType))
}

func TestServiceGetExporters(t *testing.T) {
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	//nolint:staticcheck
	expMap := srv.host.GetExporters()

	v, ok := expMap[pipeline.SignalTraces]
	assert.True(t, ok)
	assert.NotNil(t, v)

	assert.Len(t, expMap, 4)
	assert.Len(t, expMap[pipeline.SignalTraces], 1)
	assert.Contains(t, expMap[pipeline.SignalTraces], component.NewID(nopType))
	assert.Len(t, expMap[pipeline.SignalMetrics], 1)
	assert.Contains(t, expMap[pipeline.SignalMetrics], component.NewID(nopType))
	assert.Len(t, expMap[pipeline.SignalLogs], 1)
	assert.Contains(t, expMap[pipeline.SignalLogs], component.NewID(nopType))
	assert.Len(t, expMap[xpipeline.SignalProfiles], 1)
	assert.Contains(t, expMap[xpipeline.SignalProfiles], component.NewID(nopType))
}

// TestServiceTelemetryCleanupOnError tests that if newService errors due to an invalid config telemetry is cleaned up
// and another service with a valid config can be started right after.
func TestServiceTelemetryCleanupOnError(t *testing.T) {
	invalidCfg := newNopConfig()
	invalidCfg.Pipelines[pipeline.NewID(pipeline.SignalTraces)].Processors[0] = component.MustNewID("invalid")
	// Create a service with an invalid config and expect an error
	_, err := New(context.Background(), newNopSettings(), invalidCfg)
	require.Error(t, err)

	// Create a service with a valid config and expect no error
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)
	assert.NoError(t, srv.Shutdown(context.Background()))
}

func TestServiceTelemetry(t *testing.T) {
	for _, tc := range ownMetricsTestCases() {
		t.Run("ipv4_"+tc.name, func(t *testing.T) {
			testCollectorStartHelperWithReaders(t, tc, "tcp4")
		})
		t.Run("ipv6_"+tc.name, func(t *testing.T) {
			testCollectorStartHelperWithReaders(t, tc, "tcp6")
		})
	}
}

func testCollectorStartHelperWithReaders(t *testing.T, tc ownMetricsTestCase, network string) {
	var once sync.Once
	loggingHookCalled := false
	hook := func(zapcore.Entry) error {
		once.Do(func() {
			loggingHookCalled = true
		})
		return nil
	}

	var (
		metricsAddr *config.Prometheus
		zpagesAddr  string
	)
	switch network {
	case "tcp", "tcp4":
		metricsAddr = promtest.GetAvailableLocalAddressPrometheus(t)
		zpagesAddr = testutil.GetAvailableLocalAddress(t)
	case "tcp6":
		metricsAddr = promtest.GetAvailableLocalIPv6AddressPrometheus(t)
		zpagesAddr = testutil.GetAvailableLocalIPv6Address(t)
	}
	require.NotZero(t, metricsAddr, "network must be either of tcp, tcp4 or tcp6")
	require.NotZero(t, zpagesAddr, "network must be either of tcp, tcp4 or tcp6")

	set := newNopSettings()
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}
	set.ExtensionsConfigs = map[component.ID]component.Config{
		component.MustNewID("zpages"): &zpagesextension.Config{
			ServerConfig: confighttp.ServerConfig{Endpoint: zpagesAddr},
		},
	}
	set.ExtensionsFactories = map[component.Type]extension.Factory{component.MustNewType("zpages"): zpagesextension.NewFactory()}
	set.LoggingOptions = []zap.Option{zap.Hooks(hook)}

	cfg := newNopConfig()
	cfg.Extensions = []component.ID{component.MustNewID("zpages")}
	cfg.Telemetry.Metrics.Readers = []config.MetricReader{
		{
			Pull: &config.PullMetricReader{
				Exporter: config.MetricExporter{
					Prometheus: metricsAddr,
				},
			},
		},
	}
	cfg.Telemetry.Resource = make(map[string]*string)
	// Include resource attributes under the service::telemetry::resource key.
	for k, v := range tc.userDefinedResource {
		cfg.Telemetry.Resource[k] = v
	}

	// Create a service, check for metrics, shutdown and repeat to ensure that telemetry can be started/shutdown and started again.
	for i := 0; i < 2; i++ {
		srv, err := New(context.Background(), set, cfg)
		require.NoError(t, err)

		require.NoError(t, srv.Start(context.Background()))
		// Sleep for 1 second to ensure the http server is started.
		time.Sleep(1 * time.Second)
		assert.True(t, loggingHookCalled)

		assertResourceLabels(t, srv.telemetrySettings.Resource, tc.expectedLabels)
		assertMetrics(t, fmt.Sprintf("%s:%d", *metricsAddr.Host, *metricsAddr.Port), tc.expectedLabels)
		assertZPages(t, zpagesAddr)
		require.NoError(t, srv.Shutdown(context.Background()))
	}
}

// TestServiceTelemetryRestart tests that the service correctly restarts the telemetry server.
func TestServiceTelemetryRestart(t *testing.T) {
	// Create a service
	srvOne, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	// URL of the telemetry service metrics endpoint
	telemetryURL := "http://localhost:8888/metrics"

	// Start the service
	require.NoError(t, srvOne.Start(context.Background()))

	// check telemetry server to ensure we get a response
	var resp *http.Response

	resp, err = http.Get(telemetryURL)
	assert.NoError(t, err)
	assert.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Response body must be closed now instead of defer as the test
	// restarts the server on the same port. Leaving response open
	// leaks a goroutine.
	resp.Body.Close()

	// Shutdown the service
	require.NoError(t, srvOne.Shutdown(context.Background()))

	// Create a new service with the same telemetry
	srvTwo, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	// Start the new service
	require.NoError(t, srvTwo.Start(context.Background()))

	// check telemetry server to ensure we get a response
	require.Eventually(t,
		func() bool {
			resp, err = http.Get(telemetryURL)
			assert.NoError(t, resp.Body.Close())
			return err == nil
		},
		500*time.Millisecond,
		100*time.Millisecond,
		"Must get a valid response from the service",
	)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Shutdown the new service
	assert.NoError(t, srvTwo.Shutdown(context.Background()))
}

func TestExtensionNotificationFailure(t *testing.T) {
	set := newNopSettings()
	cfg := newNopConfig()

	extName := component.MustNewType("configWatcher")
	configWatcherExtensionFactory := newConfigWatcherExtensionFactory(extName)
	set.ExtensionsConfigs = map[component.ID]component.Config{component.NewID(extName): configWatcherExtensionFactory.CreateDefaultConfig()}
	set.ExtensionsFactories = map[component.Type]extension.Factory{extName: configWatcherExtensionFactory}
	cfg.Extensions = []component.ID{component.NewID(extName)}

	// Create a service
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)

	// Start the service
	require.Error(t, srv.Start(context.Background()))

	// Shut down the service
	require.NoError(t, srv.Shutdown(context.Background()))
}

func TestNilCollectorEffectiveConfig(t *testing.T) {
	set := newNopSettings()
	set.CollectorConf = nil
	cfg := newNopConfig()

	extName := component.MustNewType("configWatcher")
	configWatcherExtensionFactory := newConfigWatcherExtensionFactory(extName)
	set.ExtensionsConfigs = map[component.ID]component.Config{component.NewID(extName): configWatcherExtensionFactory.CreateDefaultConfig()}
	set.ExtensionsFactories = map[component.Type]extension.Factory{extName: configWatcherExtensionFactory}
	cfg.Extensions = []component.ID{component.NewID(extName)}

	// Create a service
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)

	// Start the service
	require.NoError(t, srv.Start(context.Background()))

	// Shut down the service
	require.NoError(t, srv.Shutdown(context.Background()))
}

func TestServiceTelemetryLogger(t *testing.T) {
	srv, err := New(context.Background(), newNopSettings(), newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})
	assert.NotNil(t, srv.telemetrySettings.Logger)
}

func TestServiceFatalError(t *testing.T) {
	set := newNopSettings()
	set.AsyncErrorChannel = make(chan error)

	srv, err := New(context.Background(), set, newNopConfig())
	require.NoError(t, err)

	assert.NoError(t, srv.Start(context.Background()))
	t.Cleanup(func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	})

	go func() {
		ev := componentstatus.NewFatalErrorEvent(assert.AnError)
		srv.host.NotifyComponentStatusChange(&componentstatus.InstanceID{}, ev)
	}()

	err = <-srv.host.AsyncErrorChannel

	require.ErrorIs(t, err, assert.AnError)
}

func TestServiceInvalidTelemetryConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		wantErr error
		cfg     telemetry.Config
	}{
		{
			name: "log config with processors and invalid config",
			cfg: telemetry.Config{
				Logs: telemetry.LogsConfig{
					Encoding: "console",
					Processors: []config.LogRecordProcessor{
						{
							Batch: &config.BatchLogRecordProcessor{
								Exporter: config.LogRecordExporter{
									OTLP: &config.OTLP{},
								},
							},
						},
					},
				},
			},
			wantErr: errors.New("unsupported protocol \"\""),
		},
	}
	for _, tt := range tests {
		set := newNopSettings()
		set.AsyncErrorChannel = make(chan error)

		cfg := newNopConfig()
		cfg.Telemetry = tt.cfg
		_, err := New(context.Background(), set, cfg)
		if tt.wantErr != nil {
			require.ErrorContains(t, err, tt.wantErr.Error())
		} else {
			require.NoError(t, err)
		}
	}
}

func assertResourceLabels(t *testing.T, res pcommon.Resource, expectedLabels map[string]labelValue) {
	for key, labelValue := range expectedLabels {
		lookupKey, ok := prometheusToOtelConv[key]
		if !ok {
			lookupKey = key
		}
		value, ok := res.Attributes().Get(lookupKey)
		switch labelValue.state {
		case labelNotPresent:
			assert.False(t, ok)
		case labelAnyValue:
			assert.True(t, ok)
		default:
			assert.Equal(t, labelValue.label, value.AsString())
		}
	}
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
	expectedMetrics := map[string]bool{
		"target_info":                                    false,
		"otelcol_process_memory_rss":                     false,
		"otelcol_process_cpu_seconds":                    false,
		"otelcol_process_runtime_total_sys_memory_bytes": false,
		"otelcol_process_runtime_heap_alloc_bytes":       false,
		"otelcol_process_runtime_total_alloc_bytes":      false,
		"otelcol_process_uptime":                         false,
	}
	for metricName, metricFamily := range parsed {
		if _, ok := expectedMetrics[metricName]; !ok {
			require.True(t, ok, "unexpected metric: %s", metricName)
		}
		expectedMetrics[metricName] = true
		if metricName != "target_info" {
			// require is used here so test fails with a single message.
			require.True(
				t,
				strings.HasPrefix(metricName, prefix),
				"expected prefix %q but string starts with %q",
				prefix,
				metricName[:len(prefix)+1]+"...")
		}

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
	for k, val := range expectedMetrics {
		require.True(t, val, "missing metric: %s", k)
	}
}

func assertZPages(t *testing.T, zpagesAddr string) {
	paths := []string{
		"/debug/tracez",
		"/debug/pipelinez",
		"/debug/servicez",
		"/debug/extensionz",
	}

	testZPagePathFn := func(t *testing.T, path string) {
		client := &http.Client{}
		resp, err := client.Get("http://" + zpagesAddr + path)
		require.NoError(t, err, "error retrieving zpage at %q", path)
		assert.Equal(t, http.StatusOK, resp.StatusCode, "unsuccessful zpage %q GET", path)
		assert.NoError(t, resp.Body.Close())
	}

	for _, path := range paths {
		testZPagePathFn(t, path)
	}
}

func newNopSettings() Settings {
	receiversConfigs, receiversFactories := builders.NewNopReceiverConfigsAndFactories()
	processorsConfigs, processorsFactories := builders.NewNopProcessorConfigsAndFactories()
	connectorsConfigs, connectorsFactories := builders.NewNopConnectorConfigsAndFactories()
	exportersConfigs, exportersFactories := builders.NewNopExporterConfigsAndFactories()
	extensionsConfigs, extensionsFactories := builders.NewNopExtensionConfigsAndFactories()

	return Settings{
		BuildInfo:           component.NewDefaultBuildInfo(),
		CollectorConf:       confmap.New(),
		ReceiversConfigs:    receiversConfigs,
		ReceiversFactories:  receiversFactories,
		ProcessorsConfigs:   processorsConfigs,
		ProcessorsFactories: processorsFactories,
		ExportersConfigs:    exportersConfigs,
		ExportersFactories:  exportersFactories,
		ConnectorsConfigs:   connectorsConfigs,
		ConnectorsFactories: connectorsFactories,
		ExtensionsConfigs:   extensionsConfigs,
		ExtensionsFactories: extensionsFactories,
		AsyncErrorChannel:   make(chan error),
	}
}

func newNopConfig() Config {
	return newNopConfigPipelineConfigs(pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{component.NewID(nopType)},
			Processors: []component.ID{component.NewID(nopType)},
			Exporters:  []component.ID{component.NewID(nopType)},
		},
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers:  []component.ID{component.NewID(nopType)},
			Processors: []component.ID{component.NewID(nopType)},
			Exporters:  []component.ID{component.NewID(nopType)},
		},
		pipeline.NewID(pipeline.SignalLogs): {
			Receivers:  []component.ID{component.NewID(nopType)},
			Processors: []component.ID{component.NewID(nopType)},
			Exporters:  []component.ID{component.NewID(nopType)},
		},
		pipeline.NewID(xpipeline.SignalProfiles): {
			Receivers:  []component.ID{component.NewID(nopType)},
			Processors: []component.ID{component.NewID(nopType)},
			Exporters:  []component.ID{component.NewID(nopType)},
		},
	})
}

func newNopConfigPipelineConfigs(pipelineCfgs pipelines.Config) Config {
	return Config{
		Extensions: extensions.Config{component.NewID(nopType)},
		Pipelines:  pipelineCfgs,
		Telemetry: telemetry.Config{
			Logs: telemetry.LogsConfig{
				Level:       zapcore.InfoLevel,
				Development: false,
				Encoding:    "console",
				Sampling: &telemetry.LogsSamplingConfig{
					Enabled:    true,
					Tick:       10 * time.Second,
					Initial:    100,
					Thereafter: 100,
				},
				OutputPaths:       []string{"stderr"},
				ErrorOutputPaths:  []string{"stderr"},
				DisableCaller:     false,
				DisableStacktrace: false,
				InitialFields:     map[string]any(nil),
			},
			Metrics: telemetry.MetricsConfig{
				Level: configtelemetry.LevelBasic,
				Readers: []config.MetricReader{
					{
						Pull: &config.PullMetricReader{Exporter: config.MetricExporter{Prometheus: &config.Prometheus{
							Host: newPtr("localhost"),
							Port: newPtr(8888),
						}}},
					},
				},
			},
		},
	}
}

type configWatcherExtension struct{}

func (comp *configWatcherExtension) Start(context.Context, component.Host) error {
	return nil
}

func (comp *configWatcherExtension) Shutdown(context.Context) error {
	return nil
}

func (comp *configWatcherExtension) NotifyConfig(context.Context, *confmap.Conf) error {
	return errors.New("Failed to resolve config")
}

func newConfigWatcherExtensionFactory(name component.Type) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return &configWatcherExtension{}, nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newPtr[T int | string](str T) *T {
	return &str
}
