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

	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

type mockProvider struct {
	scheme string
	retM   *confmap.Conf
	errR   error
	errS   error
	errW   error
	errC   error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher confmap.WatcherFunc) (confmap.Retrieved, error) {
	if m.errR != nil {
		return confmap.Retrieved{}, m.errR
	}
	if m.retM == nil {
		return confmap.NewRetrievedFromMap(confmap.New()), nil
	}
	if watcher != nil {
		watcher(&confmap.ChangeEvent{Error: m.errW})
	}
	return confmap.NewRetrievedFromMap(
		m.retM,
		confmap.WithRetrievedClose(func(ctx context.Context) error { return m.errC })), nil
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

type mockConverter struct {
	err error
}

func (m *mockConverter) Convert(context.Context, *confmap.Conf) error {
	return errors.New("converter_err")
}

func TestMapResolver_Errors(t *testing.T) {
	tests := []struct {
		name              string
		locations         []string
		providers         []confmap.Provider
		converters        []confmap.Converter
		expectResolveErr  bool
		expectWatchErr    bool
		expectCloseErr    bool
		expectShutdownErr bool
	}{
		{
			name:             "unsupported location scheme",
			locations:        []string{"mock:", "not_supported:"},
			providers:        []confmap.Provider{&mockProvider{}},
			expectResolveErr: true,
		},
		{
			name:      "retrieve location config error",
			locations: []string{"mock:", "err:"},
			providers: []confmap.Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", errR: errors.New("retrieve_err")},
			},
			expectResolveErr: true,
		},
		{
			name:             "converter error",
			locations:        []string{"mock:", filepath.Join("testdata", "otelcol-nop.yaml")},
			providers:        []confmap.Provider{&mockProvider{}, fileprovider.New()},
			converters:       []confmap.Converter{&mockConverter{err: errors.New("converter_err")}},
			expectResolveErr: true,
		},
		{
			name:      "watch error",
			locations: []string{"mock:", "err:"},
			providers: func() []confmap.Provider {
				conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return []confmap.Provider{&mockProvider{}, &mockProvider{scheme: "err", retM: conf, errW: errors.New("watch_err")}}
			}(),
			expectWatchErr: true,
		},
		{
			name:      "close error",
			locations: []string{"mock:", "err:"},
			providers: func() []confmap.Provider {
				conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return []confmap.Provider{
					&mockProvider{},
					&mockProvider{scheme: "err", retM: conf, errC: errors.New("close_err")},
				}
			}(),
			expectCloseErr: true,
		},
		{
			name:      "shutdown error",
			locations: []string{"mock:", "err:"},
			providers: func() []confmap.Provider {
				conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
				require.NoError(t, err)
				return []confmap.Provider{
					&mockProvider{},
					&mockProvider{scheme: "err", retM: conf, errS: errors.New("close_err")},
				}
			}(),
			expectShutdownErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := newMapResolver(tt.locations, makeMapProvidersMap(tt.providers...), tt.converters)
			assert.NoError(t, err)

			_, errN := resolver.Resolve(context.Background())
			if tt.expectResolveErr {
				assert.Error(t, errN)
				return
			}
			assert.NoError(t, errN)

			errW := <-resolver.Watch()
			if tt.expectWatchErr {
				assert.Error(t, errW)
				return
			}
			assert.NoError(t, errW)

			_, errC := resolver.Resolve(context.Background())
			if tt.expectCloseErr {
				assert.Error(t, errC)
				return
			}
			assert.NoError(t, errN)

			errS := resolver.Shutdown(context.Background())
			if tt.expectShutdownErr {
				assert.Error(t, errS)
				return
			}
			assert.NoError(t, errC)
		})
	}
}

func TestBackwardsCompatibilityForFilePath(t *testing.T) {
	tests := []struct {
		name       string
		location   string
		errMessage string
	}{
		{
			name:       "unix",
			location:   `/test`,
			errMessage: `unable to read the file file:/test`,
		},
		{
			name:       "file_unix",
			location:   `file:/test`,
			errMessage: `unable to read the file file:/test`,
		},
		{
			name:       "windows_C",
			location:   `C:\test`,
			errMessage: `unable to read the file file:C:\test`,
		},
		{
			name:       "windows_z",
			location:   `z:\test`,
			errMessage: `unable to read the file file:z:\test`,
		},
		{
			name:       "file_windows",
			location:   `file:C:\test`,
			errMessage: `unable to read the file file:C:\test`,
		},
		{
			name:       "invalid_scheme",
			location:   `LL:\test`,
			errMessage: `scheme "LL" is not supported for uri "LL:\\test"`,
		},
	}
	for _, tt := range tests {
		resolver, err := newMapResolver([]string{tt.location}, makeMapProvidersMap(fileprovider.New()), nil)
		assert.NoError(t, err)
		_, err = resolver.Resolve(context.Background())
		assert.Contains(t, err.Error(), tt.errMessage, tt.name)
	}
}

func TestMapResolver(t *testing.T) {
	provider := func() confmap.Provider {
		conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{retM: conf}
	}()

	resolver, err := newMapResolver([]string{"mock:"}, makeMapProvidersMap(provider), nil)
	require.NoError(t, err)
	_, errN := resolver.Resolve(context.Background())
	assert.NoError(t, errN)

	errW := <-resolver.Watch()
	assert.NoError(t, errW)

	// Repeat Resolve/Watch.

	_, errN = resolver.Resolve(context.Background())
	assert.NoError(t, errN)

	errW = <-resolver.Watch()
	assert.NoError(t, errW)

	errC := resolver.Shutdown(context.Background())
	assert.NoError(t, errC)
}

func TestMapResolverNoLocations(t *testing.T) {
	_, err := newMapResolver([]string{}, makeMapProvidersMap(fileprovider.New()), nil)
	assert.Error(t, err)
}

func TestMapResolverMapProviders(t *testing.T) {
	_, err := newMapResolver([]string{filepath.Join("testdata", "otelcol-nop.yaml")}, nil, nil)
	assert.Error(t, err)
}

func TestMapResolverNoWatcher(t *testing.T) {
	watcherWG := sync.WaitGroup{}
	resolver, err := newMapResolver(
		[]string{filepath.Join("testdata", "otelcol-nop.yaml")},
		makeMapProvidersMap(fileprovider.New()), nil)
	require.NoError(t, err)
	_, errN := resolver.Resolve(context.Background())
	assert.NoError(t, errN)

	watcherWG.Add(1)
	go func() {
		errW, ok := <-resolver.Watch()
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	assert.NoError(t, resolver.Shutdown(context.Background()))
	watcherWG.Wait()
}

func TestMapResolverShutdownClosesWatch(t *testing.T) {
	provider := func() confmap.Provider {
		// Use fakeRetrieved with nil errors to have Watchable interface implemented.
		conf, err := confmaptest.LoadConf(filepath.Join("testdata", "otelcol-nop.yaml"))
		require.NoError(t, err)
		return &mockProvider{retM: conf, errW: configsource.ErrSessionClosed}
	}()

	resolver, err := newMapResolver([]string{"mock:"}, makeMapProvidersMap(provider), nil)
	_, errN := resolver.Resolve(context.Background())
	require.NoError(t, err)

	assert.NoError(t, errN)

	watcherWG := sync.WaitGroup{}
	watcherWG.Add(1)
	go func() {
		errW, ok := <-resolver.Watch()
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	assert.NoError(t, resolver.Shutdown(context.Background()))
	watcherWG.Wait()
}
