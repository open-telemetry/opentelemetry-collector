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
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				Configs:   tt.extensionsConfigs,
				Factories: tt.factories,
			}, tt.config)
			require.Error(t, err)
			assert.EqualError(t, err, tt.wantErrMsg)
		})
	}
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
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				Configs:   tt.extensionsConfigs,
				Factories: tt.factories,
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
