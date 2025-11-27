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
