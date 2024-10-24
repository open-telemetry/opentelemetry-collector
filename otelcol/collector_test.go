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
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor/processortest"
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

type mockCfgProvider struct {
	ConfigProvider
	watcher chan error
}

func (p mockCfgProvider) Watch() <-chan error {
	return p.watcher
}

func TestCollectorStateAfterConfigChange(t *testing.T) {
	watcher := make(chan error, 1)
	col, err := NewCollector(CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: nopFactories,
		// this will be overwritten, but we need something to get past validation
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}),
	})
	require.NoError(t, err)
	provider, err := NewConfigProvider(newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	col.configProvider = &mockCfgProvider{ConfigProvider: provider, watcher: watcher}

	wg := startCollector(context.Background(), t, col)

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	watcher <- nil

	assert.Eventually(t, func() bool {
		return StateRunning == col.GetState()
	}, 2*time.Second, 200*time.Millisecond)

	col.Shutdown()

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
		func(context.Context, extension.Settings, component.Config) (component.Component, error) {
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
			assert.EqualValues(t, component.NewID(unhealthyProcessorFactory.Type()), k.ComponentID())
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
	assert.Error(t, col.Run(context.Background()))
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
			set := CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", tt.file)}),
			}

			col, err := NewCollector(set)
			require.NoError(t, err)

			if tt.errExpected {
				require.Error(t, col.Run(context.Background()))
				assert.Equal(t, StateClosed, col.GetState())
			} else {
				wg := startCollector(context.Background(), t, col)
				col.Shutdown()
				wg.Wait()
				assert.Equal(t, StateClosed, col.GetState())
			}
		})
	}
}

func TestCollectorRun(t *testing.T) {
	tests := []struct {
		file string
	}{
		{file: "otelcol-noreaders.yaml"},
		{file: "otelcol-emptyreaders.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			set := CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", tt.file)}),
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

func TestCollectorShutdownBeforeRun(t *testing.T) {
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

func TestCollectorClosedStateOnStartUpError(t *testing.T) {
	// Load a bad config causing startup to fail
	set := CollectorSettings{
		BuildInfo:              component.NewDefaultBuildInfo(),
		Factories:              nopFactories,
		ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid.yaml")}),
	}
	col, err := NewCollector(set)
	require.NoError(t, err)

	// Expect run to error
	require.Error(t, col.Run(context.Background()))

	// Expect state to be closed
	assert.Equal(t, StateClosed, col.GetState())
}

func TestCollectorDryRun(t *testing.T) {
	tests := map[string]struct {
		settings    CollectorSettings
		expectedErr string
	}{
		"invalid_processor": {
			settings: CollectorSettings{
				BuildInfo:              component.NewDefaultBuildInfo(),
				Factories:              nopFactories,
				ConfigProviderSettings: newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "otelcol-invalid.yaml")}),
			},
			expectedErr: `service::pipelines::traces: references processor "invalid" which is not configured`,
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

func newDefaultConfigProviderSettings(t testing.TB, uris []string) ConfigProviderSettings {
	fileProvider := newFakeProvider("file", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, uri[5:]))
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
func newConfFromFile(t testing.TB, fileName string) map[string]any {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(t, err, "unable to read the file %v", fileName)

	var data map[string]any
	require.NoError(t, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return confmap.NewFromStringMap(data).ToStringMap()
}
