// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelconf "go.opentelemetry.io/contrib/otelconf/v0.3.0"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
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
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/pipeline/xpipeline"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
	"go.opentelemetry.io/collector/service/telemetry/telemetrytest"
)

const (
	otelCommand = "otelcoltest"
)

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
	observerCore, observedLogs := observer.New(zapcore.WarnLevel)
	zapLogger := zap.New(observerCore)

	set := newNopSettings()
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetrytest.WithLogger(zapLogger, nil),
	)

	cfg := newNopConfig()
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	}()

	require.NotNil(t, srv.telemetrySettings.Logger)
	assert.Equal(t, srv.telemetrySettings.Logger, srv.Logger())
	assert.Equal(t, zapcore.WarnLevel, srv.telemetrySettings.Logger.Level())
	srv.telemetrySettings.Logger.Warn("warn_message")
	srv.telemetrySettings.Logger.Info("info_message")

	entries := observedLogs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "warn_message", entries[0].Message)
}

func TestServiceTelemetryLogging_Settings(t *testing.T) {
	observerCore, observedLogs := observer.New(zapcore.WarnLevel)
	zapConfig := zap.Config{Encoding: "foo"}

	set := newNopSettings()
	set.BuildZapLogger = func(cfg zap.Config, opts ...zap.Option) (*zap.Logger, error) {
		require.Equal(t, zapConfig, cfg)
		return zap.New(observerCore, opts...), nil
	}
	set.LoggingOptions = []zap.Option{zap.Fields(zap.String("extra.field", "value"))}
	set.BuildInfo = component.BuildInfo{Version: "test version", Command: otelCommand}
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetry.WithCreateLogger(
			func(_ context.Context, set telemetry.LoggerSettings, _ component.Config) (
				*zap.Logger, component.ShutdownFunc, error,
			) {
				require.NotNil(t, set.BuildZapLogger)
				require.Empty(t, set.ZapOptions)
				logger, err := set.BuildZapLogger(zapConfig)
				return logger, nil, err
			},
		),
	)

	cfg := newNopConfig()
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	}()

	require.NotNil(t, srv.telemetrySettings.Logger)
	assert.Equal(t, srv.telemetrySettings.Logger, srv.Logger())
	assert.Equal(t, zapcore.WarnLevel, srv.telemetrySettings.Logger.Level())
	srv.telemetrySettings.Logger.Warn("warn_message")

	entries := observedLogs.All()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].ContextMap(), "extra.field")
}

func TestServiceTelemetryMetrics(t *testing.T) {
	// Start a service and check that metrics are produced as expected.
	// We do this twice to ensure that the server is stopped cleanly.
	for range 2 {
		reader := metric.NewManualReader()
		set := newNopSettings()
		set.TelemetryFactory = telemetry.NewFactory(
			func() component.Config { return nil },
			telemetrytest.WithMeterProvider(
				metric.NewMeterProvider(
					metric.WithReader(reader),
				),
			),
		)

		srv, err := New(context.Background(), set, newNopConfig())
		require.NoError(t, err)
		require.NoError(t, srv.Start(context.Background()))

		var rm metricdata.ResourceMetrics
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)

		assertMetrics(t, rm)
		require.NoError(t, srv.Shutdown(context.Background()))
	}
}

func assertMetrics(t *testing.T, rm metricdata.ResourceMetrics) {
	require.Len(t, rm.ScopeMetrics, 1)
	assert.Equal(t, "go.opentelemetry.io/collector/service", rm.ScopeMetrics[0].Scope.Name)

	actualNames := make([]string, len(rm.ScopeMetrics[0].Metrics))
	for i, m := range rm.ScopeMetrics[0].Metrics {
		actualNames[i] = m.Name
	}
	assert.ElementsMatch(t, []string{
		"otelcol_process_cpu_seconds",
		"otelcol_process_memory_rss",
		"otelcol_process_runtime_heap_alloc_bytes",
		"otelcol_process_runtime_total_alloc_bytes",
		"otelcol_process_runtime_total_sys_memory_bytes",
		"otelcol_process_uptime",
	}, actualNames)
}

func TestServiceTelemetryDefaultViews(t *testing.T) {
	var views []otelconf.View
	set := newNopSettings()
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetry.WithCreateMeterProvider(
			func(_ context.Context, set telemetry.MeterSettings, _ component.Config) (telemetry.MeterProvider, error) {
				views = set.DefaultViews(configtelemetry.LevelBasic)
				return telemetrytest.ShutdownMeterProvider{
					MeterProvider: noopmetric.NewMeterProvider(),
				}, nil
			},
		),
	)

	srv, err := New(context.Background(), set, newNopConfig())
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))
	defer func() {
		assert.NoError(t, srv.Shutdown(context.Background()))
	}()
	require.NotEmpty(t, views)
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

	// The zpages extension will register/unregister a span processor with
	// the tracer provider if it implements the RegisterSpanProcessor and
	// UnregisterSpanProcessor methods of the opentelemetry-go SDK implementation.
	// Hence we use sdktrace below, rather than the noop tracer provider.
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetrytest.WithTracerProvider(sdktrace.NewTracerProvider()),
	)

	// Start a service and check that zpages is healthy.
	// We do this twice to ensure that the server is stopped cleanly.
	for range 2 {
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

// TestServiceTelemetryRestart tests that the service starts and shuts down telemetry as expected.
func TestServiceTelemetryRestart(t *testing.T) {
	telemetryCreated := make(chan struct{}, 1)
	telemetryShutdown := make(chan struct{}, 1)

	set := newNopSettings()
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetry.WithCreateTracerProvider(
			func(context.Context, telemetry.TracerSettings, component.Config) (telemetry.TracerProvider, error) {
				telemetryCreated <- struct{}{}
				return telemetrytest.ShutdownTracerProvider{
					TracerProvider: nooptrace.NewTracerProvider(),
					ShutdownFunc: func(context.Context) error {
						telemetryShutdown <- struct{}{}
						return nil
					},
				}, nil
			},
		),
	)

	for range 2 {
		// Create and start a service, telemetry should be created.
		srv, err := New(context.Background(), set, newNopConfig())
		require.NoError(t, err)
		require.NoError(t, srv.Start(context.Background()))
		<-telemetryCreated

		// Shutdown the service, telemetry should be shutdown.
		require.NoError(t, srv.Shutdown(context.Background()))
		<-telemetryShutdown
	}
}

func TestServiceTelemetryShutdownError(t *testing.T) {
	set := newNopSettings()
	set.TelemetryFactory = telemetry.NewFactory(
		func() component.Config { return nil },
		telemetry.WithCreateLogger(
			func(context.Context, telemetry.LoggerSettings, component.Config) (*zap.Logger, component.ShutdownFunc, error) {
				return zap.NewNop(), func(context.Context) error {
					return errors.New("an exception occurred")
				}, nil
			},
		),
		telemetry.WithCreateMeterProvider(
			func(context.Context, telemetry.MeterSettings, component.Config) (telemetry.MeterProvider, error) {
				return telemetrytest.ShutdownMeterProvider{
					MeterProvider: noopmetric.NewMeterProvider(),
					ShutdownFunc: func(context.Context) error {
						return errors.New("an exception occurred")
					},
				}, nil
			},
		),
		telemetry.WithCreateTracerProvider(
			func(context.Context, telemetry.TracerSettings, component.Config) (telemetry.TracerProvider, error) {
				return telemetrytest.ShutdownTracerProvider{
					TracerProvider: nooptrace.NewTracerProvider(),
					ShutdownFunc: func(context.Context) error {
						return errors.New("an exception occurred")
					},
				}, nil
			},
		),
	)

	// Create and start a service
	cfg := newNopConfig()
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)
	require.NoError(t, srv.Start(context.Background()))

	// Shutdown the service
	err = srv.Shutdown(context.Background())
	assert.EqualError(t, err, ""+
		"failed to shutdown tracer provider: an exception occurred; "+
		"failed to shutdown meter provider: an exception occurred; "+
		"failed to shutdown logger: an exception occurred",
	)
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

func TestServiceTelemetryCreateProvidersError(t *testing.T) {
	loggerOpt := telemetry.WithCreateLogger(
		func(context.Context, telemetry.LoggerSettings, component.Config) (*zap.Logger, component.ShutdownFunc, error) {
			return nil, nil, errors.New("something went wrong")
		},
	)
	meterOpt := telemetry.WithCreateMeterProvider(
		func(context.Context, telemetry.MeterSettings, component.Config) (telemetry.MeterProvider, error) {
			return nil, errors.New("something went wrong")
		},
	)
	tracerOpt := telemetry.WithCreateTracerProvider(
		func(context.Context, telemetry.TracerSettings, component.Config) (telemetry.TracerProvider, error) {
			return nil, errors.New("something went wrong")
		},
	)
	resourceOpt := telemetry.WithCreateResource(
		func(context.Context, telemetry.Settings, component.Config) (pcommon.Resource, error) {
			return pcommon.Resource{}, errors.New("something went wrong")
		},
	)

	type testcase struct {
		opts        []telemetry.FactoryOption
		expectedErr string
	}
	for name, tc := range map[string]testcase{
		"CreateLogger": {
			opts:        []telemetry.FactoryOption{loggerOpt, meterOpt, tracerOpt},
			expectedErr: "failed to create logger: something went wrong",
		},
		"CreateMeterProvider": {
			opts:        []telemetry.FactoryOption{meterOpt, tracerOpt},
			expectedErr: "failed to create meter provider: something went wrong",
		},
		"CreateTracerProvider": {
			opts:        []telemetry.FactoryOption{tracerOpt},
			expectedErr: "failed to create tracer provider: something went wrong",
		},
		"CreateResource": {
			opts:        []telemetry.FactoryOption{resourceOpt},
			expectedErr: "failed to create resource: something went wrong",
		},
	} {
		t.Run(name, func(t *testing.T) {
			set := newNopSettings()
			set.TelemetryFactory = telemetry.NewFactory(func() component.Config { return nil }, tc.opts...)
			_, err := New(context.Background(), set, newNopConfig())
			require.EqualError(t, err, tc.expectedErr)
		})
	}
}

func TestNew_NilTelemetryProvider(t *testing.T) {
	set := newNopSettings()
	set.TelemetryFactory = nil
	_, err := New(context.Background(), set, newNopConfig())
	require.EqualError(t, err, "telemetry factory not provided")
}

func TestServiceTelemetryConsistentInstanceID(t *testing.T) {
	var loggerResource, meterResource, tracerResource *pcommon.Resource

	createLoggerCalled := false
	createMeterCalled := false
	createTracerCalled := false

	baseFactory := otelconftelemetry.NewFactory()

	set := newNopSettings()
	set.TelemetryFactory = telemetry.NewFactory(
		baseFactory.CreateDefaultConfig,
		telemetry.WithCreateResource(baseFactory.CreateResource),
		telemetry.WithCreateLogger(func(ctx context.Context, settings telemetry.LoggerSettings, cfg component.Config) (*zap.Logger, component.ShutdownFunc, error) {
			createLoggerCalled = true
			loggerResource = settings.Resource
			return baseFactory.CreateLogger(ctx, settings, cfg)
		}),
		telemetry.WithCreateMeterProvider(func(ctx context.Context, settings telemetry.MeterSettings, cfg component.Config) (telemetry.MeterProvider, error) {
			createMeterCalled = true
			meterResource = settings.Resource
			return baseFactory.CreateMeterProvider(ctx, settings, cfg)
		}),
		telemetry.WithCreateTracerProvider(func(ctx context.Context, settings telemetry.TracerSettings, cfg component.Config) (telemetry.TracerProvider, error) {
			createTracerCalled = true
			tracerResource = settings.Resource
			return baseFactory.CreateTracerProvider(ctx, settings, cfg)
		}),
	)

	cfg := newNopConfig()
	srv, err := New(context.Background(), set, cfg)
	require.NoError(t, err)

	require.True(t, createLoggerCalled, "logger should have been created")
	require.True(t, createMeterCalled, "meter provider should have been created")
	require.True(t, createTracerCalled, "tracer provider should have been created")

	var serviceInstanceID string
	if sid, ok := srv.telemetrySettings.Resource.Attributes().Get("service.instance.id"); ok {
		serviceInstanceID = sid.AsString()
	}
	require.NotEmpty(t, serviceInstanceID, "service.instance.id not found in service resource")

	require.NotNil(t, loggerResource, "logger should have received a resource")
	require.NotNil(t, meterResource, "meter provider should have received a resource")
	require.NotNil(t, tracerResource, "tracer provider should have received a resource")

	var loggerInstanceID, meterInstanceID, tracerInstanceID string
	if sid, ok := loggerResource.Attributes().Get("service.instance.id"); ok {
		loggerInstanceID = sid.AsString()
	}
	if sid, ok := meterResource.Attributes().Get("service.instance.id"); ok {
		meterInstanceID = sid.AsString()
	}
	if sid, ok := tracerResource.Attributes().Get("service.instance.id"); ok {
		tracerInstanceID = sid.AsString()
	}

	require.NotEmpty(t, loggerInstanceID, "logger resource should have service.instance.id")
	require.NotEmpty(t, meterInstanceID, "meter resource should have service.instance.id")
	require.NotEmpty(t, tracerInstanceID, "tracer resource should have service.instance.id")

	assert.Equal(t, serviceInstanceID, loggerInstanceID,
		"logger should use the same service.instance.id as the service resource")
	assert.Equal(t, serviceInstanceID, meterInstanceID,
		"meter provider should use the same service.instance.id as the service resource")
	assert.Equal(t, serviceInstanceID, tracerInstanceID,
		"tracer provider should use the same service.instance.id as the service resource")

	t.Logf("service.instance.id = %s (shared by logger, meter, and tracer)", serviceInstanceID)

	require.NoError(t, srv.Shutdown(context.Background()))
}

func newNopSettings() Settings {
	receiversConfigs, receiversFactories := builders.NewNopReceiverConfigsAndFactories()
	processorsConfigs, processorsFactories := builders.NewNopProcessorConfigsAndFactories()
	connectorsConfigs, connectorsFactories := builders.NewNopConnectorConfigsAndFactories()
	exportersConfigs, exportersFactories := builders.NewNopExporterConfigsAndFactories()
	extensionsConfigs, extensionsFactories := builders.NewNopExtensionConfigsAndFactories()
	telemetryFactory := telemetry.NewFactory(func() component.Config { return nil })

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
		TelemetryFactory:    telemetryFactory,
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
		Telemetry: &otelconftelemetry.Config{
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
