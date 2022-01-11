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
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

type mockProvider struct {
	ret  *fakeRetrieved
	errR error
	errS error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher configmapprovider.WatcherFunc) (configmapprovider.Retrieved, error) {
	if m.errR != nil {
		return nil, m.errR
	}
	if m.ret == nil {
		return &fakeRetrieved{}, nil
	}
	m.ret.watcher = watcher
	return m.ret, nil
}

func (m *mockProvider) Scheme() string {
	return "mock"
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

type fakeRetrieved struct {
	configmapprovider.Retrieved
	retM    *config.Map
	errG    error
	errW    error
	errC    error
	watcher configmapprovider.WatcherFunc
}

func (er *fakeRetrieved) Get(context.Context) (*config.Map, error) {
	if er.watcher != nil {
		er.watcher(&configmapprovider.ChangeEvent{Error: er.errW})
	}
	if er.errG != nil {
		return nil, er.errG
	}
	if er.retM == nil {
		return config.NewMap(), nil
	}
	return er.retM, nil
}

func (er *fakeRetrieved) Close(context.Context) error {
	return er.errC
}

func TestConfigProvider_Errors(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)

	tests := []struct {
		name              string
		locations         []string
		parserProvider    map[string]configmapprovider.Provider
		cfgMapConverters  []config.MapConverter
		configUnmarshaler configunmarshaler.ConfigUnmarshaler
		expectNewErr      bool
		expectWatchErr    bool
		expectShutdownErr bool
	}{
		{
			name:      "retrieve_err",
			locations: []string{"mock:", "not_supported:"},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "retrieve_err",
			locations: []string{"mock:", "err:"},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
				"err":  &mockProvider{errR: errors.New("retrieve_err")},
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "get_err",
			locations: []string{"mock:", "err:"},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
				"err":  &mockProvider{ret: &fakeRetrieved{errG: errors.New("retrieve_err")}},
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "converter_err",
			locations: []string{"mock:", path.Join("testdata", "otelcol-nop.yaml")},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
				"file": configmapprovider.NewFile(),
			},
			cfgMapConverters: []config.MapConverter{
				config.NewMapConverter(func(context.Context, *config.Map) error { return errors.New("converter_err") }),
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "unmarshal_err",
			locations: []string{"mock:", path.Join("testdata", "otelcol-nop.yaml")},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
				"file": configmapprovider.NewFile(),
			},
			configUnmarshaler: &errConfigUnmarshaler{err: errors.New("unmarshal_err")},
			expectNewErr:      true,
		},
		{
			name:      "validation_err",
			locations: []string{"mock:", path.Join("testdata", "otelcol-invalid.yaml")},
			parserProvider: map[string]configmapprovider.Provider{
				"mock": &mockProvider{},
				"file": configmapprovider.NewFile(),
			},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:      "watch_err",
			locations: []string{"mock:", "err:"},
			parserProvider: func() map[string]configmapprovider.Provider {
				cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return map[string]configmapprovider.Provider{
					"mock": &mockProvider{},
					"err":  &mockProvider{ret: &fakeRetrieved{retM: cfgMap, errW: errors.New("watch_err")}},
				}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectWatchErr:    true,
		},
		{
			name:      "close_err",
			locations: []string{"mock:", "err:"},
			parserProvider: func() map[string]configmapprovider.Provider {
				cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return map[string]configmapprovider.Provider{
					"mock": &mockProvider{},
					"err":  &mockProvider{ret: &fakeRetrieved{retM: cfgMap, errC: errors.New("close_err")}},
				}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectShutdownErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgW := NewConfigProvider(tt.locations, tt.parserProvider, tt.cfgMapConverters, tt.configUnmarshaler)
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
	configMapProvider := func() configmapprovider.Provider {
		// Use fakeRetrieved with nil errors to have Watchable interface implemented.
		cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{ret: &fakeRetrieved{retM: cfgMap}}
	}()

	cfgW := NewConfigProvider(
		[]string{"watcher:"},
		map[string]configmapprovider.Provider{"watcher": configMapProvider},
		nil,
		configunmarshaler.NewDefault())
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
	cfgW := NewConfigProvider(
		[]string{path.Join("testdata", "otelcol-nop.yaml")},
		map[string]configmapprovider.Provider{"file": configmapprovider.NewFile()},
		nil,
		configunmarshaler.NewDefault())
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
	configMapProvider := func() configmapprovider.Provider {
		// Use fakeRetrieved with nil errors to have Watchable interface implemented.
		cfgMap, err := configtest.LoadConfigMap(path.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{ret: &fakeRetrieved{retM: cfgMap, errW: configsource.ErrSessionClosed}}
	}()

	watcherWG := sync.WaitGroup{}
	cfgW := NewConfigProvider(
		[]string{"watcher:"},
		map[string]configmapprovider.Provider{"watcher": configMapProvider},
		nil,
		configunmarshaler.NewDefault())
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
