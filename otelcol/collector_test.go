// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package collector handles the command-line, configuration, and runs the OC collector.
package otelcol

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/service/telemetry"
	"go.opentelemetry.io/collector/service/telemetry/telemetrytest"
)

func TestStateString(t *testing.T) {
	assert.Equal(t, "Starting", StateStarting.String())
	assert.Equal(t, "Running", StateRunning.String())
	assert.Equal(t, "Closing", StateClosing.String())
	assert.Equal(t, "Closed", StateClosed.String())
	assert.Equal(t, "UNKNOWN", State(13).String())
}

func TestCollectorStartAsGoRoutine(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	}
	col, err := NewCollector(set)
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()
	col.Shutdown()
	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorCancelContext(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	}
	col, err := NewCollector(set)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	wg := startCollector(ctx, t, col)

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	cancel()
	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorStateAfterConfigChange(t *testing.T) {
	var watcher confmap.WatcherFunc
	fileProvider := newFakeProvider("file", func(_ context.Context, uri string, w confmap.WatcherFunc) (*confmap.Retrieved, error) {
		watcher = w
		conf := newConfFromFile(t, uri[5:])
		return confmap.NewRetrieved(conf)
	})

	shutdownRequests := make(chan chan struct{})
	shutdown := func(ctx context.Context) error {
		unblock := make(chan struct{})
		select {
		case <-ctx.Done():
		case shutdownRequests <- unblock:
			select {
			case <-unblock:
			case <-ctx.Done():
			}
		}
		return nil
	}
	factories, err := nopFactories()
	require.NoError(t, err)
	factories.Telemetry = telemetry.NewFactory(
		func() component.Config { return fakeTelemetryConfig{} },
		telemetrytest.WithLogger(zap.NewNop(), shutdown),
	)

	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{filepath.Join("testdata", "otelcol-nop.yaml")},
			ProviderFactories: []confmap.ProviderFactory{
				fileProvider,
			},
		},
	}
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: set,
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)
	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 10*time.Second, 10*time.Millisecond)

	// On config change, the collector will internally close
	// and recreate the service. The metrics reader will try to
	// push to the OTLP endpoint. We block the request to check
	// the state of the collector during the config change event.
	watcher(&confmap.ChangeEvent{})
	unblock := <-shutdownRequests
	assert.Equal(t, StateClosing, col.GetState())
	close(unblock)
	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 10*time.Second, 10*time.Millisecond)

	// Do it again, but this time call Shutdown during the
	// config change to make sure the internal service shutdown
	// does not influence collector shutdown.
	watcher(&confmap.ChangeEvent{})
	unblock = <-shutdownRequests
	assert.Equal(t, StateClosing, col.GetState())
	col.Shutdown()
	close(unblock)

	// After the config reload, the final shutdown should occur.
	close(<-shutdownRequests)
	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorReportError(t *testing.T) {
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.asyncErrorChannel <- errors.New("err2")

	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

// NewStatusWatcherExtensionFactory returns a component.ExtensionFactory to construct a status watcher extension.
func NewStatusWatcherExtensionFactory(
	onStatusChanged func(source *componentstatus.InstanceID, event *componentstatus.Event),
) extension.Factory {
	return extension.NewFactory(
		component.MustNewType("statuswatcher"),
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return &statusWatcherExtension{onStatusChanged: onStatusChanged}, nil
		},
		component.StabilityLevelStable)
}

// statusWatcherExtension receives status events reported via component status reporting for testing
// purposes.
type statusWatcherExtension struct {
	component.StartFunc
	component.ShutdownFunc
	onStatusChanged func(source *componentstatus.InstanceID, event *componentstatus.Event)
}

func (e statusWatcherExtension) ComponentStatusChanged(source *componentstatus.InstanceID, event *componentstatus.Event) {
	e.onStatusChanged(source, event)
}

func TestComponentStatusWatcher(t *testing.T) {
	factories, err := nopFactories()
	require.NoError(t, err)

	// Use a processor factory that creates "unhealthy" processor: one that
	// always reports StatusRecoverableError after successful Start.
	unhealthyProcessorFactory := processortest.NewUnhealthyProcessorFactory()
	factories.Processors[unhealthyProcessorFactory.Type()] = unhealthyProcessorFactory

	// Keep track of all status changes in a map.
	changedComponents := map[*componentstatus.InstanceID][]componentstatus.Status{}
	var mux sync.Mutex
	onStatusChanged := func(source *componentstatus.InstanceID, event *componentstatus.Event) {
		if source.ComponentID().Type() != unhealthyProcessorFactory.Type() {
			return
		}
		mux.Lock()
		defer mux.Unlock()
		changedComponents[source] = append(changedComponents[source], event.Status())
	}

	// Add a "statuswatcher" extension that will receive notifications when processor
	// status changes.
	factory := NewStatusWatcherExtensionFactory(onStatusChanged)
	factories.Extensions[factory.Type()] = factory

	// Create a collector
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-statuswatcher.yaml")}),
	})
	require.NoError(t, err)

	// Start the newly created collector.
	wg := startCollector(context.Background(), t, col)

	// An unhealthy processor asynchronously reports a recoverable error. Depending on the Go
	// Scheduler the statuses reported at startup will be one of the two valid sequences below.
	startupStatuses1 := []componentstatus.Status{
		componentstatus.StatusStarting,
		componentstatus.StatusOK,
		componentstatus.StatusRecoverableError,
	}
	startupStatuses2 := []componentstatus.Status{
		componentstatus.StatusStarting,
		componentstatus.StatusRecoverableError,
	}
	// the modulus of the actual statuses will match the modulus of the startup statuses
	startupStatuses := func(actualStatuses []componentstatus.Status) []componentstatus.Status {
		if len(actualStatuses)%2 == 1 {
			return startupStatuses1
		}
		return startupStatuses2
	}

	// The "unhealthy" processors will now begin to asynchronously report StatusRecoverableError.
	// We expect to see these reports.
	assert.Eventually(t, func() bool {
		mux.Lock()
		defer mux.Unlock()

		for k, v := range changedComponents {
			// All processors must report a status change with the same ID
			assert.Equal(t, component.NewID(unhealthyProcessorFactory.Type()), k.ComponentID())
			// And all must have a valid startup sequence
			assert.Equal(t, startupStatuses(v), v)
		}
		// We have 3 processors with exactly the same ID in otelcol-statuswatcher.yaml
		// We must have exactly 3 items in our map. This ensures that the "source" argument
		// passed to status change func is unique per instance of source component despite
		// components having the same IDs (having same ID for different component instances
		// is a normal situation for processors).
		return len(changedComponents) == 3
	}, 2*time.Second, time.Millisecond*100)

	col.Shutdown()
	wg.Wait()

	// Check for additional statuses after Shutdown.
	for _, v := range changedComponents {
		expectedStatuses := append([]componentstatus.Status{}, startupStatuses(v)...)
		expectedStatuses = append(expectedStatuses, componentstatus.StatusStopping, componentstatus.StatusStopped)
		assert.Equal(t, expectedStatuses, v)
	}

	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorSendSignal(t *testing.T) {
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	})
	require.NoError(t, err)

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.signalsChannel <- syscall.SIGHUP

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.signalsChannel <- syscall.SIGTERM

	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorFailedShutdown(t *testing.T) {
	t.Skip("This test was using telemetry shutdown failure, switch to use a component that errors on shutdown.")

	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	})
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.EqualError(t, col.Run(context.Background()), "failed to shutdown collector telemetry: err1")
	}()

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()

	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorStartInvalidConfig(t *testing.T) {
	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid.yaml")}),
	})
	require.NoError(t, err)
	assert.EqualError(t, col.Run(context.Background()), "invalid configuration: service::pipelines::traces: references processor \"invalid\" which is not configured")
}

func TestNewCollectorInvalidConfigProviderSettings(t *testing.T) {
	_, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: ConfigProviderSettings{},
	})
	require.Error(t, err)
}

func TestNewCollectorUseConfig(t *testing.T) {
	set := newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")})

	col, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: set,
	})
	require.NoError(t, err)
	require.NotNil(t, col.configProvider)
}

func TestNewCollectorValidatesResolverSettings(t *testing.T) {
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{filepath.Join("testdata", "otelcol-nop.yaml")},
		},
	}

	_, err := NewCollector(CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: set,
	})
	require.Error(t, err)
}

func TestCollectorRun(t *testing.T) {
	tests := map[string]struct {
		factories  func() (Factories, error)
		configFile string
	}{
		"nop": {
			factories:  nopFactories,
			configFile: "otelcol-nop.yaml",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			set := CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              test.factories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", test.configFile)}),
			}
			col, err := NewCollector(set)
			require.NoError(t, err)

			wg := startCollector(context.Background(), t, col)

			col.Shutdown()
			wg.Wait()
			assert.Equal(t, StateClosed, col.GetState())
		})
	}
}

func TestCollectorRun_AfterShutdown(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	}
	col, err := NewCollector(set)
	require.NoError(t, err)

	// Calling shutdown before collector is running should cause it to return quickly
	require.NotPanics(t, func() { col.Shutdown() })

	wg := startCollector(context.Background(), t, col)

	col.Shutdown()
	wg.Wait()
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorRun_Errors(t *testing.T) {
	tests := map[string]struct {
		settings    CollectorSettings
		expectedErr string
	}{
		"factories_error": {
			settings: CollectorSettings{
				Factories: func() (Factories, error) {
					return Factories{}, errors.New("no factories for you")
				},
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
			},
			expectedErr: "failed to initialize factories: no factories for you",
		},
		"invalid_processor": {
			settings: CollectorSettings{
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid.yaml")}),
			},
			expectedErr: `invalid configuration: service::pipelines::traces: references processor "invalid" which is not configured`,
		},
		"invalid_telemetry_config": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid-telemetry.yaml")}),
			},
			expectedErr: "failed to get config: cannot unmarshal the configuration: decoding failed due to the following error(s):\n\n'service.telemetry' has invalid keys: unknown",
		},
		"missing_telemetry_factory": {
			settings: CollectorSettings{
				BuildInfo: component.NewDefaultBuildInfo(),
				Factories: func() (Factories, error) {
					factories, _ := nopFactories()
					factories.Telemetry = nil
					return factories, nil
				},
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-otelconftelemetry.yaml")}),
			},
			expectedErr: "failed to get config: cannot unmarshal the configuration: otelcol.Factories.Telemetry must not be nil. For example, you can use otelconftelemetry.NewFactory to build a telemetry factory",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			col, err := NewCollector(test.settings)
			require.NoError(t, err)

			// Expect run to error
			err = col.Run(context.Background())
			require.EqualError(t, err, test.expectedErr)

			// Expect state to be closed
			assert.Equal(t, StateClosed, col.GetState())
		})
	}
}

func TestCollectorDryRun(t *testing.T) {
	tests := map[string]struct {
		settings    CollectorSettings
		expectedErr string
	}{
		"factories_error": {
			settings: CollectorSettings{
				Factories: func() (Factories, error) {
					return Factories{}, errors.New("no factories for you")
				},
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
			},
			expectedErr: "failed to initialize factories: no factories for you",
		},
		"invalid_processor": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid.yaml")}),
			},
			expectedErr: `service::pipelines::traces: references processor "invalid" which is not configured`,
		},
		"invalid_connector_use_unused_exp": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid-connector-unused-exp.yaml")}),
			},
			expectedErr: `failed to build pipelines: connector "nop/connector1" used as receiver in [logs/in2] pipeline but not used in any supported exporter pipeline`,
		},
		"invalid_connector_use_unused_rec": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid-connector-unused-rec.yaml")}),
			},
			expectedErr: `failed to build pipelines: connector "nop/connector1" used as exporter in [logs/in2] pipeline but not used in any supported receiver pipeline`,
		},
		"cyclic_connector": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-cyclic-connector.yaml")}),
			},
			expectedErr: `failed to build pipelines: cycle detected: connector "nop/forward" (traces to traces) -> connector "nop/forward" (traces to traces)`,
		},
		"invalid_telemetry_config": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid-telemetry.yaml")}),
			},
			expectedErr: "failed to get config: cannot unmarshal the configuration: decoding failed due to the following error(s):\n\n'service.telemetry' has invalid keys: unknown",
		},
		"missing_telemetry_factory": {
			settings: CollectorSettings{
				BuildInfo: component.NewDefaultBuildInfo(),
				Factories: func() (Factories, error) {
					factories, _ := nopFactories()
					factories.Telemetry = nil
					return factories, nil
				},
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-otelconftelemetry.yaml")}),
			},
			expectedErr: "failed to get config: cannot unmarshal the configuration: otelcol.Factories.Telemetry must not be nil. For example, you can use otelconftelemetry.NewFactory to build a telemetry factory",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			col, err := NewCollector(test.settings)
			require.NoError(t, err)

			err = col.DryRun(context.Background())
			if test.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, test.expectedErr)
			}
		})
	}
}

func startCollector(ctx context.Context, t *testing.T, col *Collector) *sync.WaitGroup {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		assert.NoError(t, col.Run(ctx))
	}()
	return wg
}

type failureProvider struct{}

func newFailureProvider(_ confmap.ProviderSettings) confmap.Provider {
	return &failureProvider{}
}

func (fmp *failureProvider) Retrieve(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return nil, errors.New("a failure occurred during configuration retrieval")
}

func (*failureProvider) Scheme() string {
	return "file"
}

func (*failureProvider) Shutdown(context.Context) error {
	return nil
}

type fakeProvider struct {
	scheme string
	ret    func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)
	logger *zap.Logger
}

func (f *fakeProvider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return f.ret(ctx, uri, watcher)
}

func (f *fakeProvider) Scheme() string {
	return f.scheme
}

func (f *fakeProvider) Shutdown(context.Context) error {
	return nil
}

func newFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)) confmap.ProviderFactory {
	return confmap.NewProviderFactory(func(ps confmap.ProviderSettings) confmap.Provider {
		return &fakeProvider{
			scheme: scheme,
			ret:    ret,
			logger: ps.Logger,
		}
	})
}

func newEnvProvider() confmap.ProviderFactory {
	return newFakeProvider("env", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		// When using `env` as the default scheme for tests, the uri will not include `env:`.
		// Instead of duplicating the switch cases, the scheme is added instead.
		if uri[0:4] != "env:" {
			uri = "env:" + uri
		}
		switch uri {
		case "env:COMPLEX_VALUE":
			return confmap.NewRetrieved([]any{"localhost:3042"})
		case "env:HOST":
			return confmap.NewRetrieved("localhost")
		case "env:OS":
			return confmap.NewRetrieved("ubuntu")
		case "env:PR":
			return confmap.NewRetrieved("amd")
		case "env:PORT":
			return confmap.NewRetrieved(3044)
		case "env:INT":
			return confmap.NewRetrieved(1)
		case "env:INT32":
			return confmap.NewRetrieved(32)
		case "env:INT64":
			return confmap.NewRetrieved(64)
		case "env:FLOAT32":
			return confmap.NewRetrieved(float32(3.25))
		case "env:FLOAT64":
			return confmap.NewRetrieved(float64(6.4))
		case "env:BOOL":
			return confmap.NewRetrieved(true)
		}
		return nil, errors.New("impossible")
	})
}

func newDefaultConfigProviderSettings(tb testing.TB, uris []string) ConfigProviderSettings {
	fileProvider := newFakeProvider("file", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(tb, uri[5:]))
	})
	return ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: uris,
			ProviderFactories: []confmap.ProviderFactory{
				fileProvider,
				newEnvProvider(),
			},
		},
	}
}

// newConfFromFile creates a new Conf by reading the given file.
func newConfFromFile(tb testing.TB, fileName string) map[string]any {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(tb, err, "unable to read the file %v", fileName)

	var data map[string]any
	require.NoError(tb, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return confmap.NewFromStringMap(data).ToStringMap()
}

func TestProviderAndConverterModules(t *testing.T) {
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
		ProviderModules: map[string]string{
			"nop": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
		},
		ConverterModules: []string{
			"go.opentelemetry.io/collector/converter/testconverter v1.2.3",
		},
	}
	col, err := NewCollector(set)
	require.NoError(t, err)
	wg := startCollector(context.Background(), t, col)
	require.NoError(t, err)
	providerModules := map[string]string{
		"nop": "go.opentelemetry.io/collector/confmap/provider/testprovider v1.2.3",
	}
	converterModules := []string{
		"go.opentelemetry.io/collector/converter/testconverter v1.2.3",
	}
	assert.Equal(t, providerModules, col.set.ProviderModules)
	assert.Equal(t, converterModules, col.set.ConverterModules)
	col.Shutdown()
	wg.Wait()
}

func TestCollectorLoggingOptions(t *testing.T) {
	// Use zap observer to verify that LoggingOptions are applied
	observerCore, observedLogs := observer.New(zapcore.InfoLevel)

	factories, err := nopFactories()
	require.NoError(t, err)

	// Create a custom telemetry factory that uses BuildZapLogger
	// This ensures BuildZapLogger (which includes LoggingOptions) is used
	factories.Telemetry = telemetry.NewFactory(
		func() component.Config { return fakeTelemetryConfig{} },
		telemetry.WithCreateLogger(
			func(_ context.Context, set telemetry.LoggerSettings, _ component.Config) (
				*zap.Logger, component.ShutdownFunc, error,
			) {
				require.Empty(t, set.ZapOptions) // injected through BuidlZapLogger
				logger, buildErr := set.BuildZapLogger(zap.NewDevelopmentConfig())
				return logger, nil, buildErr
			},
		),
	)

	set := CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: func() (Factories, error) { return factories, nil },
		ConfigProviderSettings: newDefaultConfigProviderSettings(t,
			[]string{filepath.Join("testdata", "otelcol-nop.yaml")},
		),
		LoggingOptions: []zap.Option{
			zap.WrapCore(func(zapcore.Core) zapcore.Core {
				return observerCore
			}),
		},
	}

	col, err := NewCollector(set)
	require.NoError(t, err)

	// Start and stop the collector.
	wg := startCollector(context.Background(), t, col)
	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState() && col.service != nil
	}, 2*time.Second, 200*time.Millisecond)
	col.Shutdown()
	wg.Wait()

	// Check that logs have been redirected to our observer core,
	// which proves that LoggingOptions were applied.
	entries := observedLogs.All()
	require.NotEmpty(t, entries, "Logger should have logged messages")
}
