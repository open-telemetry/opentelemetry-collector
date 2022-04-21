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

package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/config/mapprovider/filemapprovider"
	"go.opentelemetry.io/collector/internal/configunmarshaler"
)

type mockProvider struct {
	scheme string
	retM   *config.Map
	errR   error
	errS   error
	errW   error
	errC   error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher config.WatcherFunc) (config.Retrieved, error) {
	if m.errR != nil {
		return config.Retrieved{}, m.errR
	}
	if m.retM == nil {
		return config.NewRetrievedFromMap(config.NewMap()), nil
	}
	if watcher != nil {
		watcher(&config.ChangeEvent{Error: m.errW})
	}
	return config.NewRetrievedFromMap(
		m.retM,
		config.WithRetrievedClose(func(ctx context.Context) error { return m.errC })), nil
}

func (m *mockProvider) Scheme() string {
	if m.scheme == "" {
		return "mock"
	}
	return m.scheme
}

func (m *mockProvider) Shutdown(context.Context) error {
	return m.errS
}

type errConfigUnmarshaler struct {
	err error
}

func (ecu *errConfigUnmarshaler) Unmarshal(*config.Map, component.Factories) (*config.Config, error) {
	return nil, ecu.err
}

func TestConfigProvider_Errors(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)

	tests := []struct {
		name              string
		locations         []string
		parserProvider    []config.MapProvider
		cfgMapConverters  []config.MapConverterFunc
		configUnmarshaler configunmarshaler.ConfigUnmarshaler
		expectNewErr      bool
		expectWatchErr    bool
		expectShutdownErr bool
	}{
		{
			name:              "retrieve_err",
			locations:         []string{"mock:", "not_supported:"},
			parserProvider:    []config.MapProvider{&mockProvider{}},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "retrieve_err",
			locations: []string{"mock:", "err:"},
			parserProvider: []config.MapProvider{
				&mockProvider{},
				&mockProvider{scheme: "err", errR: errors.New("retrieve_err")},
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:              "converter_err",
			locations:         []string{"mock:", filepath.Join("testdata", "otelcol-nop.yaml")},
			parserProvider:    []config.MapProvider{&mockProvider{}, filemapprovider.New()},
			cfgMapConverters:  []config.MapConverterFunc{func(context.Context, *config.Map) error { return errors.New("converter_err") }},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:              "unmarshal_err",
			locations:         []string{"mock:", filepath.Join("testdata", "otelcol-nop.yaml")},
			parserProvider:    []config.MapProvider{&mockProvider{}, filemapprovider.New()},
			configUnmarshaler: &errConfigUnmarshaler{err: errors.New("unmarshal_err")},
			expectNewErr:      true,
		},
		{
			name:              "validation_err",
			locations:         []string{"mock:", filepath.Join("testdata", "otelcol-invalid.yaml")},
			parserProvider:    []config.MapProvider{&mockProvider{}, filemapprovider.New()},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "watch_err",
			locations: []string{"mock:", "err:"},
			parserProvider: func() []config.MapProvider {
				cfgMap, err := configtest.LoadConfigMap(filepath.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return []config.MapProvider{&mockProvider{}, &mockProvider{scheme: "err", retM: cfgMap, errW: errors.New("watch_err")}}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectWatchErr:    true,
		},
		{
			name:      "close_err",
			locations: []string{"mock:", "err:"},
			parserProvider: func() []config.MapProvider {
				cfgMap, err := configtest.LoadConfigMap(filepath.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return []config.MapProvider{
					&mockProvider{},
					&mockProvider{scheme: "err", retM: cfgMap, errC: errors.New("close_err")},
				}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectShutdownErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := ConfigProviderSettings{
				Locations:     tt.locations,
				MapProviders:  makeConfigMapProviderMap(tt.parserProvider...),
				MapConverters: tt.cfgMapConverters,
				Unmarshaler:   tt.configUnmarshaler,
			}

			cfgW, err := NewConfigProvider(set)
			assert.NoError(t, err)

			_, errN := cfgW.Get(context.Background(), factories)
			if tt.expectNewErr {
				assert.Error(t, errN)
				return
			}
			assert.NoError(t, errN)

			errW := <-cfgW.Watch()
			if tt.expectWatchErr {
				assert.Error(t, errW)
				return
			}
			assert.NoError(t, errW)

			errC := cfgW.Shutdown(context.Background())
			if tt.expectShutdownErr {
				assert.Error(t, errC)
				return
			}
			assert.NoError(t, errC)
		})
	}
}

func TestConfigProvider(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)
	mapProvider := func() config.MapProvider {
		cfgMap, err := configtest.LoadConfigMap(filepath.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{retM: cfgMap}
	}()

	set := newDefaultConfigProviderSettings([]string{"mock:"})
	set.MapProviders[mapProvider.Scheme()] = mapProvider
	cfgW, err := NewConfigProvider(set)
	require.NoError(t, err)
	_, errN := cfgW.Get(context.Background(), factories)
	assert.NoError(t, errN)

	errW := <-cfgW.Watch()
	assert.NoError(t, errW)

	// Repeat Get/Watch.

	_, errN = cfgW.Get(context.Background(), factories)
	assert.NoError(t, errN)

	errW = <-cfgW.Watch()
	assert.NoError(t, errW)

	errC := cfgW.Shutdown(context.Background())
	assert.NoError(t, errC)
}

func TestConfigProviderNoWatcher(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)

	watcherWG := sync.WaitGroup{}
	cfgW, err := NewConfigProvider(newDefaultConfigProviderSettings([]string{filepath.Join("testdata", "otelcol-nop.yaml")}))
	require.NoError(t, err)
	_, errN := cfgW.Get(context.Background(), factories)
	assert.NoError(t, errN)

	watcherWG.Add(1)
	go func() {
		errW, ok := <-cfgW.Watch()
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	assert.NoError(t, cfgW.Shutdown(context.Background()))
	watcherWG.Wait()
}

func TestConfigProvider_ShutdownClosesWatch(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)
	mapProvider := func() config.MapProvider {
		// Use fakeRetrieved with nil errors to have Watchable interface implemented.
		cfgMap, err := configtest.LoadConfigMap(filepath.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{retM: cfgMap, errW: configsource.ErrSessionClosed}
	}()

	set := newDefaultConfigProviderSettings([]string{"mock:"})
	set.MapProviders[mapProvider.Scheme()] = mapProvider

	watcherWG := sync.WaitGroup{}
	cfgW, err := NewConfigProvider(set)
	_, errN := cfgW.Get(context.Background(), factories)
	require.NoError(t, err)

	assert.NoError(t, errN)

	watcherWG.Add(1)
	go func() {
		errW, ok := <-cfgW.Watch()
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	assert.NoError(t, cfgW.Shutdown(context.Background()))
	watcherWG.Wait()
}

func TestBackwardsCompatibilityForFilePath(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)

	tests := []struct {
		name       string
		location   string
		errMessage string
	}{
		{
			name:       "unix",
			location:   `/test`,
			errMessage: "unable to read the file file:/test",
		},
		{
			name:       "file_unix",
			location:   `file:/test`,
			errMessage: "unable to read the file file:/test",
		},
		{
			name:       "windows_C",
			location:   `C:\test`,
			errMessage: "unable to read the file file:C:\\test",
		},
		{
			name:       "windows_z",
			location:   `z:\test`,
			errMessage: "unable to read the file file:z:\\test",
		},
		{
			name:       "file_windows",
			location:   `file:C:\test`,
			errMessage: "unable to read the file file:C:\\test",
		},
		{
			name:       "invalid_scheme",
			location:   `LL:\test`,
			errMessage: "scheme LL is not supported for location",
		},
	}
	for _, tt := range tests {
		set := newDefaultConfigProviderSettings([]string{tt.location})
		provider, err := NewConfigProvider(set)
		assert.NoError(t, err)
		_, err = provider.Get(context.Background(), factories)
		assert.Contains(t, err.Error(), tt.errMessage, tt.name)
	}

}
