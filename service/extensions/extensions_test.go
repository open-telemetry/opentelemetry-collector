// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package extensions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/service/internal/servicetelemetry"
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
				component.NewID("myextension"),
			},
			wantErrMsg: "failed to create extension \"myextension\": extension \"myextension\" is not configured",
		},
		{
			name: "missing_extension_factory",
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID("unknown"): nopExtensionConfig,
			},
			config: Config{
				component.NewID("unknown"),
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
				Telemetry:  servicetelemetry.NewNopTelemetrySettings(),
				BuildInfo:  component.NewDefaultBuildInfo(),
				Extensions: extension.NewBuilder(tt.extensionsConfigs, tt.factories),
			}, tt.config)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErrMsg)
		})
	}
}

func TestOrdering(t *testing.T) {
	var startOrder []string
	var shutdownOrder []string

	recordingExtensionFactory := newRecordingExtensionFactory(func(set extension.CreateSettings, host component.Host) error {
		id := set.ID.String()
		if id != "recording" {
			// we're only interested in the bar/baz order
			startOrder = append(startOrder, set.ID.String())
		}
		return nil
	}, func(set extension.CreateSettings) error {
		id := set.ID.String()
		if id != "recording" {
			// we're only interested in the bar/baz order
			shutdownOrder = append(shutdownOrder, set.ID.String())
		}
		return nil
	})

	exts, err := New(context.Background(), Settings{
		Telemetry: servicetelemetry.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		Extensions: extension.NewBuilder(
			map[component.ID]component.Config{
				component.NewID(recordingExtensionFactory.Type()): recordingExtensionConfig{},
				component.NewIDWithName(recordingExtensionFactory.Type(), "foo"): recordingExtensionConfig{
					// has a dependency on bar. In the real world this would be expressed by user via config.
					dependencies: []component.ID{
						component.NewIDWithName(recordingExtensionFactory.Type(), "bar"),
					},
				},
				component.NewIDWithName(recordingExtensionFactory.Type(), "bar"): recordingExtensionConfig{},
			},
			map[component.Type]extension.Factory{
				recordingExtensionFactory.Type(): recordingExtensionFactory,
			}),
	}, Config{
		component.NewID(recordingExtensionFactory.Type()),
		component.NewIDWithName(recordingExtensionFactory.Type(), "foo"),
		component.NewIDWithName(recordingExtensionFactory.Type(), "bar"),
	})
	require.NoError(t, err)
	err = exts.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	// The exact order of starting and stopping is not guaranteed, because the first extension is
	// not comparable with the other two, and can occur in any place.
	// What is guaranteed is "foo" starting before "bar".
	require.Equal(t, []string{"recording/foo", "recording/bar"}, startOrder)
	err = exts.Shutdown(context.Background())
	require.NoError(t, err)
	require.Equal(t, []string{"recording/bar", "recording/foo"}, shutdownOrder)
}

func TestNotifyConfig(t *testing.T) {
	notificationError := errors.New("Error processing config")
	nopExtensionFactory := extensiontest.NewNopFactory()
	nopExtensionConfig := nopExtensionFactory.CreateDefaultConfig()
	n1ExtensionFactory := newConfigWatcherExtensionFactory("notifiable1", func() error { return nil })
	n1ExtensionConfig := n1ExtensionFactory.CreateDefaultConfig()
	n2ExtensionFactory := newConfigWatcherExtensionFactory("notifiable2", func() error { return nil })
	n2ExtensionConfig := n1ExtensionFactory.CreateDefaultConfig()
	nErrExtensionFactory := newConfigWatcherExtensionFactory("notifiableErr", func() error { return notificationError })
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
				"nop": nopExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID("nop"): nopExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID("nop"),
			},
		},
		{
			name: "One notifiable extension",
			factories: map[component.Type]extension.Factory{
				"notifiable1": n1ExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID("notifiable1"): n1ExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID("notifiable1"),
			},
		},
		{
			name: "Multiple notifiable extensions",
			factories: map[component.Type]extension.Factory{
				"notifiable1": n1ExtensionFactory,
				"notifiable2": n2ExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID("notifiable1"): n1ExtensionConfig,
				component.NewID("notifiable2"): n2ExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID("notifiable1"),
				component.NewID("notifiable2"),
			},
		},
		{
			name: "Errors in extension notification",
			factories: map[component.Type]extension.Factory{
				"notifiableErr": nErrExtensionFactory,
			},
			extensionsConfigs: map[component.ID]component.Config{
				component.NewID("notifiableErr"): nErrExtensionConfig,
			},
			serviceExtensions: []component.ID{
				component.NewID("notifiableErr"),
			},
			want: notificationError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extensions, err := New(context.Background(), Settings{
				Telemetry:  servicetelemetry.NewNopTelemetrySettings(),
				BuildInfo:  component.NewDefaultBuildInfo(),
				Extensions: extension.NewBuilder(tt.extensionsConfigs, tt.factories),
			}, tt.serviceExtensions)
			assert.NoError(t, err)
			errs := extensions.NotifyConfig(context.Background(), confmap.NewFromStringMap(map[string]interface{}{}))
			assert.Equal(t, tt.want, errs)
		})
	}
}

type configWatcherExtension struct {
	fn func() error
}

func (comp *configWatcherExtension) Start(_ context.Context, _ component.Host) error {
	return comp.fn()
}

func (comp *configWatcherExtension) Shutdown(_ context.Context) error {
	return comp.fn()
}

func (comp *configWatcherExtension) NotifyConfig(_ context.Context, _ *confmap.Conf) error {
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
		func(ctx context.Context, set extension.CreateSettings, extension component.Config) (extension.Extension, error) {
			return newConfigWatcherExtension(fn), nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newBadExtensionFactory() extension.Factory {
	return extension.NewFactory(
		"bf",
		func() component.Config {
			return &struct{}{}
		},
		func(ctx context.Context, set extension.CreateSettings, extension component.Config) (extension.Extension, error) {
			return nil, nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newCreateErrorExtensionFactory() extension.Factory {
	return extension.NewFactory(
		"err",
		func() component.Config {
			return &struct{}{}
		},
		func(ctx context.Context, set extension.CreateSettings, extension component.Config) (extension.Extension, error) {
			return nil, errors.New("cannot create \"err\" extension type")
		},
		component.StabilityLevelDevelopment,
	)
}

func TestStatusReportedOnStartupShutdown(t *testing.T) {
	// compare two slices of status events ignoring timestamp
	assertEqualStatuses := func(t *testing.T, evts1, evts2 []*component.StatusEvent) {
		assert.Equal(t, len(evts1), len(evts2))
		for i := 0; i < len(evts1); i++ {
			ev1 := evts1[i]
			ev2 := evts2[i]
			assert.Equal(t, ev1.Status(), ev2.Status())
			assert.Equal(t, ev1.Err(), ev2.Err())
		}
	}

	for _, tc := range []struct {
		name             string
		expectedStatuses []*component.StatusEvent
		startErr         error
		shutdownErr      error
	}{
		{
			name: "successful startup/shutdown",
			expectedStatuses: []*component.StatusEvent{
				component.NewStatusEvent(component.StatusStarting),
				component.NewStatusEvent(component.StatusStopping),
				component.NewStatusEvent(component.StatusStopped),
			},
			startErr:    nil,
			shutdownErr: nil,
		},
		{
			name: "start error",
			expectedStatuses: []*component.StatusEvent{
				component.NewStatusEvent(component.StatusStarting),
				component.NewPermanentErrorEvent(assert.AnError),
			},
			startErr:    assert.AnError,
			shutdownErr: nil,
		},
		{
			name: "shutdown error",
			expectedStatuses: []*component.StatusEvent{
				component.NewStatusEvent(component.StatusStarting),
				component.NewStatusEvent(component.StatusStopping),
				component.NewPermanentErrorEvent(assert.AnError),
			},
			startErr:    nil,
			shutdownErr: assert.AnError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compID := component.NewID("statustest")
			factory := newStatusTestExtensionFactory("statustest", tc.startErr, tc.shutdownErr)
			config := factory.CreateDefaultConfig()
			extensionsConfigs := map[component.ID]component.Config{
				compID: config,
			}
			factories := map[component.Type]extension.Factory{
				"statustest": factory,
			}
			extensions, err := New(
				context.Background(),
				Settings{
					Telemetry:  servicetelemetry.NewNopTelemetrySettings(),
					BuildInfo:  component.NewDefaultBuildInfo(),
					Extensions: extension.NewBuilder(extensionsConfigs, factories),
				},
				[]component.ID{compID},
			)

			assert.NoError(t, err)

			var actualStatuses []*component.StatusEvent
			init, statusFunc := status.NewServiceStatusFunc(func(id *component.InstanceID, ev *component.StatusEvent) {
				actualStatuses = append(actualStatuses, ev)
			})
			extensions.telemetry.ReportComponentStatus = statusFunc
			init()

			assert.Equal(t, tc.startErr, extensions.Start(context.Background(), componenttest.NewNopHost()))
			if tc.startErr == nil {
				assert.Equal(t, tc.shutdownErr, extensions.Shutdown(context.Background()))
			}
			assertEqualStatuses(t, tc.expectedStatuses, actualStatuses)
		})
	}
}

type statusTestExtension struct {
	startErr    error
	shutdownErr error
}

func (ext *statusTestExtension) Start(_ context.Context, _ component.Host) error {
	return ext.startErr
}

func (ext *statusTestExtension) Shutdown(_ context.Context) error {
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
		func(ctx context.Context, set extension.CreateSettings, extension component.Config) (extension.Extension, error) {
			return newStatusTestExtension(startErr, shutdownErr), nil
		},
		component.StabilityLevelDevelopment,
	)
}

func newRecordingExtensionFactory(startCallback func(set extension.CreateSettings, host component.Host) error, shutdownCallback func(set extension.CreateSettings) error) extension.Factory {
	return extension.NewFactory(
		"recording",
		func() component.Config {
			return &recordingExtensionConfig{}
		},
		func(ctx context.Context, set extension.CreateSettings, cfg component.Config) (extension.Extension, error) {
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
	dependencies []component.ID
}

type recordingExtension struct {
	config           recordingExtensionConfig
	startCallback    func(set extension.CreateSettings, host component.Host) error
	shutdownCallback func(set extension.CreateSettings) error
	createSettings   extension.CreateSettings
}

var _ extension.DependentExtension = (*recordingExtension)(nil)

func (ext *recordingExtension) Dependencies() []component.ID {
	return ext.config.dependencies
}

func (ext *recordingExtension) Start(_ context.Context, host component.Host) error {
	return ext.startCallback(ext.createSettings, host)
}

func (ext *recordingExtension) Shutdown(_ context.Context) error {
	return ext.shutdownCallback(ext.createSettings)
}
