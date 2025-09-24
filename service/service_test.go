// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/zpagesextension"
	"go.opentelemetry.io/collector/internal/testutil"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/plog/plogotlp"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/promtest"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
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
	assert.Nil(t, srv.host.GetFactory(component.Kind{}, nopType))
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

func TestServiceTelemetryLogging(t *testing.T) {
	// Create a server for receiving OTLP logs.
	var received []plog.Logs
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/logs", func(_ http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		assert.NoError(t, err)

		exportRequest := plogotlp.NewExportRequest()
		assert.NoError(t, exportRequest.UnmarshalProto(body))
		received = append(received, exportRequest.Logs())
	})
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()

	// We'll divert zap logs to an observer.
	observerCore, observedLogs := observer.New(zapcore.WarnLevel)

	set := newNopSettings()
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}
	set.LoggingOptions = []zap.Option{
		zap.WrapCore(func(zapcore.Core) zapcore.Core {
			return observerCore
		}),
	}

	cfg := newNopConfig()
	cfg.Telemetry.Logs.Sampling = &otelconftelemetry.LogsSamplingConfig{
		Enabled:    true,
		Tick:       time.Minute,
		Initial:    2,
		Thereafter: 0,
	}
	cfg.Telemetry.Logs.Processors = []config.LogRecordProcessor{{
		Simple: &config.SimpleLogRecordProcessor{
			Exporter: config.LogRecordExporter{
				OTLP: &config.OTLP{
					Endpoint: ptr(httpServer.URL),
					Protocol: ptr("http/protobuf"),
					Insecure: ptr(true),
				},
			},
		},
	}}

	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	}()

	// The level we configured on the initial Zap logger should have
	// propagated to the final one provided to components.
	require.NotNil(t, srv.telemetrySettings.Logger)
	assert.Equal(t, zapcore.WarnLevel, srv.telemetrySettings.Logger.Level())

	// Log 5 messages at different levels. Only the warning messages should
	// be accepted, and only 2 of those due to sampling.
	for i := 0; i < 5; i++ {
		srv.telemetrySettings.Logger.Warn("warn_message")
		srv.telemetrySettings.Logger.Info("info_message")
		srv.telemetrySettings.Logger.Debug("debug_message")
	}
	assert.Equal(t, 2, observedLogs.Len())
	assert.Equal(t, 2, observedLogs.FilterMessage("warn_message").Len())
	require.Len(t, received, 2)
	for _, logs := range received {
		assert.Equal(t,
			"warn_message",
			logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Body().AsString(),
		)
	}
}

func TestServiceTelemetryMetrics(t *testing.T) {
	for _, tc := range ownMetricsTestCases() {
		t.Run("ipv4_"+tc.name, func(t *testing.T) {
			testCollectorStartHelperWithReaders(t, tc, promtest.GetAvailableLocalAddressPrometheus(t))
		})
		t.Run("ipv6_"+tc.name, func(t *testing.T) {
			testCollectorStartHelperWithReaders(t, tc, promtest.GetAvailableLocalIPv6AddressPrometheus(t))
		})
	}
}

func testCollectorStartHelperWithReaders(t *testing.T, tc ownMetricsTestCase, metricsAddr *config.Prometheus) {
	set := newNopSettings()
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}

	cfg := newNopConfig()
	cfg.Telemetry.Metrics.Readers = []config.MetricReader{
		{
			Pull: &config.PullMetricReader{
				Exporter: config.PullMetricExporter{
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

	// Start a service and check that metrics are produced as expected.
	// We do this twice to ensure that the server is stopped cleanly.
	for i := 0; i < 2; i++ {
		srv, err := New(context.Background(), set, cfg)
		require.NoError(t, err)

		require.NoError(t, srv.Start(context.Background()))

		// Wait for the HTTP server to start.
		promHost := fmt.Sprintf("%s:%d", *metricsAddr.Host, *metricsAddr.Port)
		require.Eventually(t, func() bool {
			resp, err := http.Get("http://" + promHost + "/metrics")
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode == http.StatusOK
		}, 10*time.Second, 50*time.Millisecond)

		assertResourceLabels(t, srv.telemetrySettings.Resource, tc.expectedLabels)
		assertMetrics(t, promHost, tc.expectedLabels)
		require.NoError(t, srv.Shutdown(context.Background()))
	}
}

// TestServiceTelemetryZPages verifies that the zpages extension works correctly with servce telemetry.
func TestServiceTelemetryZPages(t *testing.T) {
	t.Run("ipv4", func(t *testing.T) {
		testZPages(t, testutil.GetAvailableLocalAddress(t))
	})
	t.Run("ipv6", func(t *testing.T) {
		testZPages(t, testutil.GetAvailableLocalIPv6Address(t))
	})
}

func testZPages(t *testing.T, zpagesAddr string) {
	set := newNopSettings()
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}
	set.ExtensionsConfigs = map[component.ID]component.Config{
		component.MustNewID("zpages"): &zpagesextension.Config{
			ServerConfig: confighttp.ServerConfig{Endpoint: zpagesAddr},
		},
	}
	set.ExtensionsFactories = map[component.Type]extension.Factory{
		component.MustNewType("zpages"): zpagesextension.NewFactory(),
	}

	cfg := newNopConfig()
	cfg.Extensions = []component.ID{component.MustNewID("zpages")}
	cfg.Telemetry.Logs.Level = zapcore.FatalLevel // disable logs

	// Start a service and check that zpages is healthy.
	// We do this twice to ensure that the server is stopped cleanly.
	for i := 0; i < 2; i++ {
		srv, err := New(context.Background(), set, cfg)
		require.NoError(t, err)
		require.NoError(t, srv.Start(context.Background()))

		assert.Eventually(t, func() bool {
			return zpagesHealthy(zpagesAddr)
		}, 10*time.Second, 100*time.Millisecond, "zpages endpoint is not healthy")

		require.NoError(t, srv.Shutdown(context.Background()))
	}
}

func zpagesHealthy(zpagesAddr string) bool {
	paths := []string{
		"/debug/tracez",
		"/debug/pipelinez",
		"/debug/servicez",
		"/debug/extensionz",
	}

	for _, path := range paths {
		resp, err := http.Get("http://" + zpagesAddr + path)
		if err != nil {
			return false
		}
		if resp.Body.Close() != nil {
			return false
		}
		if resp.StatusCode != http.StatusOK {
			return false
		}
	}
	return true
}

// TestServiceTelemetryRestart tests that the service correctly restarts the telemetry server.
func TestServiceTelemetryRestart(t *testing.T) {
	metricsAddr := promtest.GetAvailableLocalAddressPrometheus(t)
	cfg := newNopConfig()
	cfg.Telemetry.Metrics.Readers = []config.MetricReader{
		{
			Pull: &config.PullMetricReader{
				Exporter: config.PullMetricExporter{
					Prometheus: metricsAddr,
				},
			},
		},
	}
	// Create a service
	srvOne, err := New(context.Background(), newNopSettings(), cfg)
	require.NoError(t, err)

	// URL of the telemetry service metrics endpoint
	telemetryURL := fmt.Sprintf("http://%s:%d/metrics", *metricsAddr.Host, *metricsAddr.Port)

	// Start the service
	require.NoError(t, srvOne.Start(context.Background()))

	// check telemetry server to ensure we get a response
	var resp *http.Response

	//nolint:gosec
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
	srvTwo, err := New(context.Background(), newNopSettings(), cfg)
	require.NoError(t, err)

	// Start the new service
	require.NoError(t, srvTwo.Start(context.Background()))

	// check telemetry server to ensure we get a response
	require.Eventually(t,
		func() bool {
			//nolint:gosec
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

func TestServiceTelemetryShutdownError(t *testing.T) {
	cfg := newNopConfig()
	cfg.Telemetry.Logs.Level = zapcore.DebugLevel
	cfg.Telemetry.Logs.Processors = []config.LogRecordProcessor{{
		Batch: &config.BatchLogRecordProcessor{
			Exporter: config.LogRecordExporter{
				OTLP: &config.OTLP{
					Protocol: ptr("http/protobuf"),
					Endpoint: ptr("http://testing.invalid"),
				},
			},
		},
	}}
	cfg.Telemetry.Metrics.Level = configtelemetry.LevelDetailed
	cfg.Telemetry.Metrics.Readers = []config.MetricReader{{
		Periodic: &config.PeriodicMetricReader{
			Exporter: config.PushMetricExporter{
				OTLP: &config.OTLPMetric{
					Protocol: ptr("http/protobuf"),
					Endpoint: ptr("http://testing.invalid"),
				},
			},
		},
	}}

	// Create and start a service
	srv, err := New(context.Background(), newNopSettings(), cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))

	// Shutdown the service
	err = srv.Shutdown(context.Background())
	require.ErrorContains(t, err, `failed to shutdown logger`)
	require.ErrorContains(t, err, `failed to shutdown meter provider`)
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
		wantErr string
		cfg     otelconftelemetry.Config
	}{
		{
			name: "log config with processors and invalid config",
			cfg: func() otelconftelemetry.Config {
				cfg := otelconftelemetry.NewFactory().CreateDefaultConfig().(*otelconftelemetry.Config)
				cfg.Logs.Level = zapcore.FatalLevel
				cfg.Logs.Processors = []config.LogRecordProcessor{{
					Batch: &config.BatchLogRecordProcessor{
						Exporter: config.LogRecordExporter{
							OTLP: &config.OTLP{},
						},
					},
				}}
				return *cfg
			}(),
			wantErr: "failed to create logger: no valid log exporter",
		},
		{
			name: "invalid metric reader",
			cfg: func() otelconftelemetry.Config {
				cfg := otelconftelemetry.NewFactory().CreateDefaultConfig().(*otelconftelemetry.Config)
				cfg.Logs.Level = zapcore.FatalLevel
				cfg.Metrics.Level = configtelemetry.LevelDetailed
				cfg.Metrics.Readers = []config.MetricReader{{
					Periodic: &config.PeriodicMetricReader{
						Exporter: config.PushMetricExporter{
							OTLP: &config.OTLPMetric{},
						},
					},
				}}
				return *cfg
			}(),
			wantErr: "failed to create meter provider: no valid metric exporter",
		},
		{
			name: "invalid trace exporter",
			cfg: func() otelconftelemetry.Config {
				cfg := otelconftelemetry.NewFactory().CreateDefaultConfig().(*otelconftelemetry.Config)
				cfg.Logs.Level = zapcore.FatalLevel
				cfg.Traces.Level = configtelemetry.LevelDetailed
				cfg.Traces.Processors = []config.SpanProcessor{{
					Batch: &config.BatchSpanProcessor{
						Exporter: config.SpanExporter{
							OTLP: &config.OTLP{},
						},
					},
				}}
				return *cfg
			}(),
			wantErr: "failed to create tracer provider: no valid span exporter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := newNopSettings()
			set.AsyncErrorChannel = make(chan error)

			cfg := newNopConfig()
			cfg.Telemetry = tt.cfg
			_, err := New(context.Background(), set, cfg)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
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

	parser := expfmt.NewTextParser(model.UTF8Validation)
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
		"promhttp_metric_handler_errors_total":           false,
	}
	for metricName, metricFamily := range parsed {
		if _, ok := expectedMetrics[metricName]; !ok {
			require.True(t, ok, "unexpected metric: %s", metricName)
		}
		expectedMetrics[metricName] = true
		if metricName == "promhttp_metric_handler_errors_total" {
			continue
		}
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
		Telemetry: otelconftelemetry.Config{
			Logs: otelconftelemetry.LogsConfig{
				Level:       zapcore.InfoLevel,
				Development: false,
				Encoding:    "console",
				Sampling: &otelconftelemetry.LogsSamplingConfig{
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
			Metrics: otelconftelemetry.MetricsConfig{
				Level: configtelemetry.LevelBasic,
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

func TestValidateGraph(t *testing.T) {
	testCases := map[string]struct {
		connectorCfg  map[component.ID]component.Config
		receiverCfg   map[component.ID]component.Config
		exporterCfg   map[component.ID]component.Config
		pipelinesCfg  pipelines.Config
		expectedError string
	}{
		"Valid connector usage": {
			connectorCfg: map[component.ID]component.Config{
				component.NewIDWithName(nopType, "connector1"): &struct{}{},
			},
			receiverCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			exporterCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			pipelinesCfg: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewIDWithName(nopType, "connector1")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.NewIDWithName(nopType, "connector1")},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
			},
			expectedError: "",
		},
		"Valid without Connector": {
			receiverCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			exporterCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			pipelinesCfg: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
			},
			expectedError: "",
		},
		"Connector used as exporter but not as receiver": {
			connectorCfg: map[component.ID]component.Config{
				component.NewIDWithName(nopType, "connector1"): &struct{}{},
			},
			receiverCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			exporterCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			pipelinesCfg: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in1"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "in2"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewIDWithName(nopType, "connector1")},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
			},
			expectedError: `failed to build pipelines: connector "nop/connector1" used as exporter in [logs/in2] pipeline but not used in any supported receiver pipeline`,
		},
		"Connector used as receiver but not as exporter": {
			connectorCfg: map[component.ID]component.Config{
				component.NewIDWithName(nopType, "connector1"): &struct{}{},
			},
			receiverCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			exporterCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			pipelinesCfg: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalLogs, "in1"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "in2"): {
					Receivers:  []component.ID{component.NewIDWithName(nopType, "connector1")},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
				pipeline.NewIDWithName(pipeline.SignalLogs, "out"): {
					Receivers:  []component.ID{component.NewID(nopType)},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewID(nopType)},
				},
			},
			expectedError: `failed to build pipelines: connector "nop/connector1" used as receiver in [logs/in2] pipeline but not used in any supported exporter pipeline`,
		},
		"Connector creates direct cycle between pipelines": {
			connectorCfg: map[component.ID]component.Config{
				component.NewIDWithName(nopType, "forward"): &struct{}{},
			},
			receiverCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			exporterCfg: map[component.ID]component.Config{
				component.NewID(nopType): &struct{}{},
			},
			pipelinesCfg: pipelines.Config{
				pipeline.NewIDWithName(pipeline.SignalTraces, "in"): {
					Receivers:  []component.ID{component.NewIDWithName(nopType, "forward")},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewIDWithName(nopType, "forward")},
				},
				pipeline.NewIDWithName(pipeline.SignalTraces, "out"): {
					Receivers:  []component.ID{component.NewIDWithName(nopType, "forward")},
					Processors: []component.ID{},
					Exporters:  []component.ID{component.NewIDWithName(nopType, "forward")},
				},
			},
			expectedError: `failed to build pipelines: cycle detected: connector "nop/forward" (traces to traces) -> connector "nop/forward" (traces to traces)`,
		},
	}

	_, connectorsFactories := builders.NewNopConnectorConfigsAndFactories()
	_, receiversFactories := builders.NewNopReceiverConfigsAndFactories()
	_, exportersFactories := builders.NewNopExporterConfigsAndFactories()

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			settings := Settings{
				ConnectorsConfigs:   tc.connectorCfg,
				ConnectorsFactories: connectorsFactories,
				ReceiversConfigs:    tc.receiverCfg,
				ReceiversFactories:  receiversFactories,
				ExportersConfigs:    tc.exporterCfg,
				ExportersFactories:  exportersFactories,
			}
			cfg := Config{
				Pipelines: tc.pipelinesCfg,
			}

			err := Validate(context.Background(), settings, cfg)
			if tc.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Equal(t, tc.expectedError, err.Error())
			}
		})
	}
}
