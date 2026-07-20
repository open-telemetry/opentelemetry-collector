// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensioncapabilities"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/internal/status"
)

func TestBuildExtensions(t *testing.T) {
	nopExtensionFactory := extensiontest.NewNopFactory()
	nopExtensionConfig := nopExtensionFactory.CreateDefaultConfig()
	errExtensionFactory := newCreateErrorExtensionFactory()
	errExtensionConfig := errExtensionFactory.CreateDefaultConfig()
	badExtensionFactory := newBadExtensionFactory()
	badExtensionCfg := badExtensionFactory.CreateDefaultConfig()

	tests := []struct {
		name              string
		factories         map[component.Type]extension.Factory
		extensionsConfigs map[component.ID]component.Config
		config            Config
		wantErrMsg        string
	}{
		{
			name: "extension_not_configured",
			config: Config{
				component.MustNewID("myextension"),
			},
			wantErrMsg: "failed to create extension \"myextension\": extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("unknown"): nopExtensionConfig,
			},
			config: Config{
				component.MustNewID("unknown"),
			},
			wantErrMsg: "failed to create extension \"unknown\": extension factory not available for: \"unknown\"",
		},
		{
			name: "error_on_create_extension",
			factories: map[component.Type]extension.Factory{
				errExtensionFactory.Type(): errExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID(errExtensionFactory.Type()): errExtensionConfig,
			},
			config: Config{
				component.NewID(errExtensionFactory.Type()),
			},
			wantErrMsg: "failed to create extension \"err\": cannot create \"err\" extension type",
		},
		{
			name: "bad_factory",
			factories: map[component.Type]extension.Factory{
				badExtensionFactory.Type(): badExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID(badExtensionFactory.Type()): badExtensionCfg,
			},
			config: Config{
				component.NewID(badExtensionFactory.Type()),
			},
			wantErrMsg: "factory for \"bf\" produced a nil extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(context.Background(), Settings{
				Telemetry:  componenttest.NewNopTelemetrySettings(),
				BuildInfo:  component.NewDefaultBuildInfo(),
				Extensions: builders.NewExtension(tt.extensionsConfigs, tt.factories),
			}, tt.config)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErrMsg)
		})
	}
}

type testOrderExt struct {
	name string
	deps []string
}

type testOrderCase struct {
	testName   string
	extensions []testOrderExt
	order      []string
	err        string
}

func TestOrdering(t *testing.T) {
	tests := []testOrderCase{
		{
			testName:   "no_deps",
			extensions: []testOrderExt{{name: ""}, {name: "foo"}, {name: "bar"}},
			order:      nil, // no predictable order
		},
		{
			testName: "deps",
			extensions: []testOrderExt{
				{name: "foo", deps: []string{"bar"}}, // foo -> bar
				{name: "baz", deps: []string{"foo"}}, // baz -> foo
				{name: "bar"},
			},
			// baz -> foo -> bar
			order: []string{"recording/bar", "recording/foo", "recording/baz"},
		},
		{
			testName: "deps_double",
			extensions: []testOrderExt{
				{name: "foo", deps: []string{"bar"}},        // foo -> bar
				{name: "baz", deps: []string{"foo", "bar"}}, // baz -> {foo, bar}
				{name: "bar"},
			},
			// baz -> foo -> bar
			order: []string{"recording/bar", "recording/foo", "recording/baz"},
		},
		{
			testName: "unknown_dep",
			extensions: []testOrderExt{
				{name: "foo", deps: []string{"BAZ"}},
				{name: "bar"},
			},
			err: "unable to find extension",
		},
		{
			testName: "circular",
			extensions: []testOrderExt{
				{name: "foo", deps: []string{"bar"}},
				{name: "bar", deps: []string{"foo"}},
			},
			err: "unable to order extensions",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.testName, testCase.testOrdering)
	}
}

func (tc testOrderCase) testOrdering(t *testing.T) {
	var startOrder []string
	var shutdownOrder []string

	recordingExtensionFactory := newRecordingExtensionFactory(func(set extension.Settings, _ component.Host) error {
		startOrder = append(startOrder, set.ID.String())
		return nil
	}, func(set extension.Settings) error {
		shutdownOrder = append(shutdownOrder, set.ID.String())
		return nil
	})

	extCfgs := make(map[component.ID]component.Config)
	extIDs := make([]component.ID, len(tc.extensions))
	for i, ext := range tc.extensions {
		extID := component.NewIDWithName(recordingExtensionFactory.Type(), ext.name)
		extIDs[i] = extID
		extCfgs[extID] = recordingExtensionConfig{dependencies: ext.deps}
	}

	exts, err := New(context.Background(), Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: builders.NewExtension(
			extCfgs,
			map[component.Type]extension.Factory{
				recordingExtensionFactory.Type(): recordingExtensionFactory,
			}),
	}, Config(extIDs))
	if tc.err != "" {
		require.ErrorContains(t, err, tc.err)
		return
	}
	require.NoError(t, err)

	err = exts.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	err = exts.Shutdown(context.Background())
	require.NoError(t, err)

	if len(tc.order) > 0 {
		require.Equal(t, tc.order, startOrder)
		slices.Reverse(shutdownOrder)
		require.Equal(t, tc.order, shutdownOrder)
	}
}

func TestNotifyConfig(t *testing.T) {
	notificationError := errors.New("Error processing config")
	nopExtensionFactory := extensiontest.NewNopFactory()
	nopExtensionConfig := nopExtensionFactory.CreateDefaultConfig()
	n1ExtensionFactory := newConfigWatcherExtensionFactory(component.MustNewType("notifiable1"), func() error { return nil })
	n1ExtensionConfig := n1ExtensionFactory.CreateDefaultConfig()
	n2ExtensionFactory := newConfigWatcherExtensionFactory(component.MustNewType("notifiable2"), func() error { return nil })
	n2ExtensionConfig := n1ExtensionFactory.CreateDefaultConfig()
	nErrExtensionFactory := newConfigWatcherExtensionFactory(component.MustNewType("notifiableErr"), func() error { return notificationError })
	nErrExtensionConfig := nErrExtensionFactory.CreateDefaultConfig()

	tests := []struct {
		name              string
		factories         map[component.Type]extension.Factory
		extensionsConfigs map[component.ID]component.Config
		serviceExtensions []component.ID
		wantErrMsg        string
		want              error
	}{
		{
			name: "No notifiable extensions",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("nop"): nopExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.MustNewID("nop"),
			},
		},
		{
			name: "One notifiable extension",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("notifiable1"): n1ExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("notifiable1"): n1ExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.MustNewID("notifiable1"),
			},
		},
		{
			name: "Multiple notifiable extensions",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("notifiable1"): n1ExtensionFactory,
				component.MustNewType("notifiable2"): n2ExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("notifiable1"): n1ExtensionConfig,
				component.MustNewID("notifiable2"): n2ExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.MustNewID("notifiable1"),
				component.MustNewID("notifiable2"),
			},
		},
		{
			name: "Errors in extension notification",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("notifiableErr"): nErrExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("notifiableErr"): nErrExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.MustNewID("notifiableErr"),
			},
			want: notificationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extensions, err := New(context.Background(), Settings{
				Telemetry:  componenttest.NewNopTelemetrySettings(),
				BuildInfo:  component.NewDefaultBuildInfo(),
				Extensions: builders.NewExtension(tt.extensionsConfigs, tt.factories),
			}, tt.serviceExtensions)
			require.NoError(t, err)
			errs := extensions.NotifyConfig(context.Background(), confmap.NewFromStringMap(map[string]any{}))
			assert.Equal(t, tt.want, errs)
		})
	}
}

func TestNotifyConfigWithNilConfig(t *testing.T) {
	called := false
	extensionFactory := newConfigWatcherExtensionFactory(component.MustNewType("notifiable"), func() error {
		called = true
		return errors.New("unexpected notification")
	})

	extensions, err := New(context.Background(), Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: builders.NewExtension(
			map[component.ID]component.Config{component.MustNewID("notifiable"): extensionFactory.CreateDefaultConfig()},
			map[component.Type]extension.Factory{component.MustNewType("notifiable"): extensionFactory},
		),
	}, []component.ID{component.MustNewID("notifiable")})
	require.NoError(t, err)

	require.NoError(t, extensions.NotifyConfig(context.Background(), nil))
	assert.False(t, called)
}

type configWatcherExtension struct {
	fn func() error
}

func (comp *configWatcherExtension) Start(context.Context, component.Host) error {
	return comp.fn()
}

func (comp *configWatcherExtension) Shutdown(context.Context) error {
	return comp.fn()
}

func (comp *configWatcherExtension) NotifyConfig(context.Context, *confmap.Conf) error {
	return comp.fn()
}

func newConfigWatcherExtension(fn func() error) *configWatcherExtension {
	comp := &configWatcherExtension{
		fn: fn,
	}

	return comp
}

func newConfigWatcherExtensionFactory(name component.Type, fn func() error) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return newConfigWatcherExtension(fn), nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newBadExtensionFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType("bf"),
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return nil, nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newCreateErrorExtensionFactory() extension.Factory {
	return extension.NewFactory(
		component.MustNewType("err"),
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return nil, errors.New("cannot create \"err\" extension type")
		},
		component.StabilityLevelDevelopment,
	)
}

func TestStatusReportedOnStartupShutdown(t *testing.T) {
	// compare two slices of status events ignoring timestamp
	assertEqualStatuses := func(t *testing.T, evts1, evts2 []*componentstatus.Event) {
		assert.Len(t, evts2, len(evts1))
		for i := range evts1 {
			ev1 := evts1[i]
			ev2 := evts2[i]
			assert.Equal(t, ev1.Status(), ev2.Status())
			assert.Equal(t, ev1.Err(), ev2.Err())
		}
	}

	for _, tt := range []struct {
		name             string
		expectedStatuses []*componentstatus.Event
		startErr         error
		shutdownErr      error
	}{
		{
			name: "successful startup/shutdown",
			expectedStatuses: []*componentstatus.Event{
				componentstatus.NewEvent(componentstatus.StatusStarting),
				componentstatus.NewEvent(componentstatus.StatusOK),
				componentstatus.NewEvent(componentstatus.StatusStopping),
				componentstatus.NewEvent(componentstatus.StatusStopped),
			},
			startErr:    nil,
			shutdownErr: nil,
		},
		{
			name: "start error",
			expectedStatuses: []*componentstatus.Event{
				componentstatus.NewEvent(componentstatus.StatusStarting),
				componentstatus.NewPermanentErrorEvent(assert.AnError),
			},
			startErr:    assert.AnError,
			shutdownErr: nil,
		},
		{
			name: "shutdown error",
			expectedStatuses: []*componentstatus.Event{
				componentstatus.NewEvent(componentstatus.StatusStarting),
				componentstatus.NewEvent(componentstatus.StatusOK),
				componentstatus.NewEvent(componentstatus.StatusStopping),
				componentstatus.NewPermanentErrorEvent(assert.AnError),
			},
			startErr:    nil,
			shutdownErr: assert.AnError,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			statusType := component.MustNewType("statustest")
			compID := component.NewID(statusType)
			factory := newStatusTestExtensionFactory(statusType, tt.startErr, tt.shutdownErr)
			config := factory.CreateDefaultConfig()
			extensionsConfigs := map[component.ID]component.Config{
				compID: config,
			}
			factories := map[component.Type]extension.Factory{
				statusType: factory,
			}

			var actualStatuses []*componentstatus.Event
			rep := status.NewReporter(func(_ *componentstatus.InstanceID, ev *componentstatus.Event) {
				actualStatuses = append(actualStatuses, ev)
			}, func(err error) {
				require.NoError(t, err)
			})

			extensions, err := New(
				context.Background(),
				Settings{
					Telemetry:  componenttest.NewNopTelemetrySettings(),
					BuildInfo:  component.NewDefaultBuildInfo(),
					Extensions: builders.NewExtension(extensionsConfigs, factories),
				},
				[]component.ID{compID},
				WithReporter(rep),
			)
			require.NoError(t, err)

			assert.Equal(t, tt.startErr, extensions.Start(context.Background(), componenttest.NewNopHost()))
			if tt.startErr == nil {
				assert.Equal(t, tt.shutdownErr, extensions.Shutdown(context.Background()))
			}
			assertEqualStatuses(t, tt.expectedStatuses, actualStatuses)
		})
	}
}

func TestExtensionReportsOwnStatus(t *testing.T) {
	statusType := component.MustNewType("selfreporting")
	compID := component.NewID(statusType)

	reportedEvent := componentstatus.NewRecoverableErrorEvent(assert.AnError)
	factory := newSelfReportingExtensionFactory(statusType, reportedEvent)
	extensionsConfigs := map[component.ID]component.Config{
		compID: factory.CreateDefaultConfig(),
	}
	factories := map[component.Type]extension.Factory{
		statusType: factory,
	}

	var reportedSources []*componentstatus.InstanceID
	var reportedEvents []*componentstatus.Event
	rep := status.NewReporter(func(source *componentstatus.InstanceID, ev *componentstatus.Event) {
		reportedSources = append(reportedSources, source)
		reportedEvents = append(reportedEvents, ev)
	}, func(err error) {
		require.NoError(t, err)
	})

	extensions, err := New(
		context.Background(),
		Settings{
			Telemetry:  componenttest.NewNopTelemetrySettings(),
			BuildInfo:  component.NewDefaultBuildInfo(),
			Extensions: builders.NewExtension(extensionsConfigs, factories),
		},
		[]component.ID{compID},
		WithReporter(rep),
	)
	require.NoError(t, err)

	// The host must implement status.Reporter so that events reported by an
	// extension about itself are routed through the reporter.
	host := &statusReporterHost{Host: componenttest.NewNopHost(), reporter: rep}
	require.NoError(t, extensions.Start(context.Background(), host))
	require.NoError(t, extensions.Shutdown(context.Background()))

	// Find the self-reported RecoverableError event and verify it was attributed
	// to the extension that reported it.
	var found bool
	for i, ev := range reportedEvents {
		if ev != reportedEvent {
			continue
		}

		found = true
		require.NotNil(t, reportedSources[i])
		assert.Equal(t, compID, reportedSources[i].ComponentID())
		assert.Equal(t, component.KindExtension, reportedSources[i].Kind())
		assert.Equal(t, assert.AnError, ev.Err())
	}
	assert.True(t, found, "expected extension to report a status event about itself")
}

// statusReporterHost is a component.Host that also implements status.Reporter,
// mirroring the host provided to extensions by the running service.
type statusReporterHost struct {
	component.Host
	reporter status.Reporter
}

func (h *statusReporterHost) ReportStatus(id *componentstatus.InstanceID, ev *componentstatus.Event) {
	h.reporter.ReportStatus(id, ev)
}

func (h *statusReporterHost) ReportOKIfStarting(id *componentstatus.InstanceID) {
	h.reporter.ReportOKIfStarting(id)
}

type selfReportingExtension struct {
	event *componentstatus.Event
}

func (ext *selfReportingExtension) Start(_ context.Context, host component.Host) error {
	componentstatus.ReportStatus(host, ext.event)
	return nil
}

func (ext *selfReportingExtension) Shutdown(context.Context) error {
	return nil
}

func newSelfReportingExtensionFactory(name component.Type, event *componentstatus.Event) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return &selfReportingExtension{event: event}, nil
		},
		component.StabilityLevelDevelopment,
	)
}

type statusTestExtension struct {
	startErr    error
	shutdownErr error
}

func (ext *statusTestExtension) Start(context.Context, component.Host) error {
	return ext.startErr
}

func (ext *statusTestExtension) Shutdown(context.Context) error {
	return ext.shutdownErr
}

func newStatusTestExtension(startErr, shutdownErr error) *statusTestExtension {
	return &statusTestExtension{
		startErr:    startErr,
		shutdownErr: shutdownErr,
	}
}

func newStatusTestExtensionFactory(name component.Type, startErr, shutdownErr error) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config {
			return &struct{}{}
		},
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return newStatusTestExtension(startErr, shutdownErr), nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newRecordingExtensionFactory(startCallback func(set extension.Settings, host component.Host) error, shutdownCallback func(set extension.Settings) error) extension.Factory {
	return extension.NewFactory(
		component.MustNewType("recording"),
		func() component.Config {
			return &recordingExtensionConfig{}
		},
		func(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
			return &recordingExtension{
				config:           cfg.(recordingExtensionConfig),
				createSettings:   set,
				startCallback:    startCallback,
				shutdownCallback: shutdownCallback,
			}, nil
		},
		component.StabilityLevelDevelopment,
	)
}

type recordingExtensionConfig struct {
	dependencies []string // names of dependencies of the same extension type
}

type recordingExtension struct {
	config           recordingExtensionConfig
	startCallback    func(set extension.Settings, host component.Host) error
	shutdownCallback func(set extension.Settings) error
	createSettings   extension.Settings
}

var _ extensioncapabilities.Dependent = (*recordingExtension)(nil)

func (ext *recordingExtension) Dependencies() []component.ID {
	if len(ext.config.dependencies) == 0 {
		return nil
	}
	deps := make([]component.ID, len(ext.config.dependencies))
	for i, dep := range ext.config.dependencies {
		deps[i] = component.MustNewIDWithName("recording", dep)
	}
	return deps
}

func (ext *recordingExtension) Start(_ context.Context, host component.Host) error {
	return ext.startCallback(ext.createSettings, host)
}

func (ext *recordingExtension) Shutdown(context.Context) error {
	return ext.shutdownCallback(ext.createSettings)
}

func TestNotifyConfigSnapshot(t *testing.T) {
	notificationError := errors.New("Error processing config snapshot")
	nopExtensionFactory := extensiontest.NewNopFactory()
	nopExtensionConfig := nopExtensionFactory.CreateDefaultConfig()
	n1ExtensionFactory := newConfigSnapshotWatcherExtensionFactory(component.MustNewType("snapshotnotifiable1"), func(extensioncapabilities.ConfigSnapshot) error { return nil })
	n1ExtensionConfig := n1ExtensionFactory.CreateDefaultConfig()
	nErrExtensionFactory := newConfigSnapshotWatcherExtensionFactory(component.MustNewType("snapshotnotifiableErr"), func(extensioncapabilities.ConfigSnapshot) error { return notificationError })
	nErrExtensionConfig := nErrExtensionFactory.CreateDefaultConfig()

	tests := []struct {
		name              string
		factories         map[component.Type]extension.Factory
		extensionsConfigs map[component.ID]component.Config
		serviceExtensions []component.ID
		want              error
	}{
		{
			name: "No config-snapshot-watcher extensions",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("nop"): nopExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("nop"): nopExtensionConfig,
			},
			serviceExtensions: []component.ID{component.MustNewID("nop")},
		},
		{
			name: "One config-snapshot-watcher extension",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("snapshotnotifiable1"): n1ExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("snapshotnotifiable1"): n1ExtensionConfig,
			},
			serviceExtensions: []component.ID{component.MustNewID("snapshotnotifiable1")},
		},
		{
			name: "Errors in config-snapshot notification",
			factories: map[component.Type]extension.Factory{
				component.MustNewType("snapshotnotifiableErr"): nErrExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.MustNewID("snapshotnotifiableErr"): nErrExtensionConfig,
			},
			serviceExtensions: []component.ID{component.MustNewID("snapshotnotifiableErr")},
			want:              notificationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extensions, err := New(context.Background(), Settings{
				Telemetry:  componenttest.NewNopTelemetrySettings(),
				BuildInfo:  component.NewDefaultBuildInfo(),
				Extensions: builders.NewExtension(tt.extensionsConfigs, tt.factories),
			}, tt.serviceExtensions)
			require.NoError(t, err)
			errs := extensions.NotifyConfigSnapshot(
				context.Background(),
				extensioncapabilities.NewConfigSnapshot(confmap.NewFromStringMap(map[string]any{}), nil),
			)
			assert.Equal(t, tt.want, errs)
		})
	}
}

func TestNotifyConfigSnapshotWithNilSnapshot(t *testing.T) {
	called := false
	extensionFactory := newConfigSnapshotWatcherExtensionFactory(component.MustNewType("snapshotnotifiable"), func(extensioncapabilities.ConfigSnapshot) error {
		called = true
		return errors.New("unexpected notification")
	})

	exts, err := New(context.Background(), Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: builders.NewExtension(
			map[component.ID]component.Config{component.MustNewID("snapshotnotifiable"): extensionFactory.CreateDefaultConfig()},
			map[component.Type]extension.Factory{component.MustNewType("snapshotnotifiable"): extensionFactory},
		),
	}, []component.ID{component.MustNewID("snapshotnotifiable")})
	require.NoError(t, err)

	require.NoError(t, exts.NotifyConfigSnapshot(context.Background(), nil))
	assert.False(t, called)
}

func TestNotifyConfigSnapshotWithNilEffectiveConfig(t *testing.T) {
	called := false
	extensionFactory := newConfigWatcherExtensionFactory(component.MustNewType("notifiable"), func() error {
		called = true
		return errors.New("unexpected notification")
	})

	exts, err := New(context.Background(), Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: builders.NewExtension(
			map[component.ID]component.Config{component.MustNewID("notifiable"): extensionFactory.CreateDefaultConfig()},
			map[component.Type]extension.Factory{component.MustNewType("notifiable"): extensionFactory},
		),
	}, []component.ID{component.MustNewID("notifiable")})
	require.NoError(t, err)

	err = exts.NotifyConfigSnapshot(
		context.Background(),
		extensioncapabilities.NewConfigSnapshot(nil, confmap.NewFromStringMap(map[string]any{"unexpanded": "${env:VALUE}"})),
	)
	require.NoError(t, err)
	assert.False(t, called)
}

func TestNotifyConfigSnapshotPrefersSnapshotWatcher(t *testing.T) {
	var configSnapshotNotifications, configNotifications int
	extensionFactory := newConfigSnapshotAndConfigWatcherExtensionFactory(
		component.MustNewType("bothwatchers"),
		func() { configSnapshotNotifications++ },
		func() { configNotifications++ },
	)

	exts, err := New(context.Background(), Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: builders.NewExtension(
			map[component.ID]component.Config{component.MustNewID("bothwatchers"): extensionFactory.CreateDefaultConfig()},
			map[component.Type]extension.Factory{component.MustNewType("bothwatchers"): extensionFactory},
		),
	}, []component.ID{component.MustNewID("bothwatchers")})
	require.NoError(t, err)

	err = exts.NotifyConfigSnapshot(
		context.Background(),
		extensioncapabilities.NewConfigSnapshot(confmap.NewFromStringMap(map[string]any{}), nil),
	)
	require.NoError(t, err)
	assert.Equal(t, 1, configSnapshotNotifications)
	assert.Zero(t, configNotifications)
}

type configSnapshotWatcherExtension struct {
	fn func(extensioncapabilities.ConfigSnapshot) error
}

func (comp *configSnapshotWatcherExtension) Start(context.Context, component.Host) error {
	return nil
}

func (comp *configSnapshotWatcherExtension) Shutdown(context.Context) error {
	return nil
}

func (comp *configSnapshotWatcherExtension) NotifyConfigSnapshot(_ context.Context, configSnapshot extensioncapabilities.ConfigSnapshot) error {
	return comp.fn(configSnapshot)
}

func newConfigSnapshotWatcherExtensionFactory(name component.Type, fn func(extensioncapabilities.ConfigSnapshot) error) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config { return &struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return &configSnapshotWatcherExtension{fn: fn}, nil
		},
		component.StabilityLevelDevelopment,
	)
}

type configSnapshotAndConfigWatcherExtension struct {
	notifyConfigSnapshot func()
	notifyConfig         func()
}

func (comp *configSnapshotAndConfigWatcherExtension) Start(context.Context, component.Host) error {
	return nil
}

func (comp *configSnapshotAndConfigWatcherExtension) Shutdown(context.Context) error {
	return nil
}

func (comp *configSnapshotAndConfigWatcherExtension) NotifyConfigSnapshot(context.Context, extensioncapabilities.ConfigSnapshot) error {
	comp.notifyConfigSnapshot()
	return nil
}

func (comp *configSnapshotAndConfigWatcherExtension) NotifyConfig(context.Context, *confmap.Conf) error {
	comp.notifyConfig()
	return nil
}

func newConfigSnapshotAndConfigWatcherExtensionFactory(name component.Type, notifyConfigSnapshot, notifyConfig func()) extension.Factory {
	return extension.NewFactory(
		name,
		func() component.Config { return &struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return &configSnapshotAndConfigWatcherExtension{
				notifyConfigSnapshot: notifyConfigSnapshot,
				notifyConfig:         notifyConfig,
			}, nil
		},
		component.StabilityLevelDevelopment,
	)
}
