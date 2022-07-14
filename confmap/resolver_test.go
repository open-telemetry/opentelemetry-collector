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

package confmap

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	scheme string
	retM   map[string]interface{}
	errR   error
	errS   error
	errW   error
	errC   error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher WatcherFunc) (Retrieved, error) {
	if m.errR != nil {
		return Retrieved{}, m.errR
	}
	if m.retM == nil {
		return NewRetrieved(nil)
	}
	if watcher != nil {
		watcher(&ChangeEvent{Error: m.errW})
	}
	return NewRetrieved(m.retM, WithRetrievedClose(func(ctx context.Context) error { return m.errC }))
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

type fakeProvider struct {
	scheme string
	ret    func(ctx context.Context, uri string, watcher WatcherFunc) (Retrieved, error)
}

func newFileProvider(t testing.TB) Provider {
	return newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (Retrieved, error) {
		return NewRetrieved(newConfFromFile(t, uri[5:]))
	})
}

func newFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher WatcherFunc) (Retrieved, error)) Provider {
	return &fakeProvider{
		scheme: scheme,
		ret:    ret,
	}
}

func (f *fakeProvider) Retrieve(ctx context.Context, uri string, watcher WatcherFunc) (Retrieved, error) {
	return f.ret(ctx, uri, watcher)
}

func (f *fakeProvider) Scheme() string {
	return f.scheme
}

func (f *fakeProvider) Shutdown(context.Context) error {
	return nil
}

type mockConverter struct {
	err error
}

func (m *mockConverter) Convert(context.Context, *Conf) error {
	return errors.New("converter_err")
}

func TestResolverErrors(t *testing.T) {
	tests := []struct {
		name              string
		locations         []string
		providers         []Provider
		converters        []Converter
		expectResolveErr  bool
		expectWatchErr    bool
		expectCloseErr    bool
		expectShutdownErr bool
	}{
		{
			name:             "unsupported location scheme",
			locations:        []string{"mock:", "not_supported:"},
			providers:        []Provider{&mockProvider{}},
			expectResolveErr: true,
		},
		{
			name:      "retrieve location config error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", errR: errors.New("retrieve_err")},
			},
			expectResolveErr: true,
		},
		{
			name:             "converter error",
			locations:        []string{"mock:"},
			providers:        []Provider{&mockProvider{}},
			converters:       []Converter{&mockConverter{err: errors.New("converter_err")}},
			expectResolveErr: true,
		},
		{
			name:      "watch error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: map[string]interface{}{}, errW: errors.New("watch_err")},
			},
			expectWatchErr: true,
		},
		{
			name:      "close error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: map[string]interface{}{}, errC: errors.New("close_err")},
			},
			expectCloseErr: true,
		},
		{
			name:      "shutdown error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: map[string]interface{}{}, errS: errors.New("close_err")},
			},
			expectShutdownErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(ResolverSettings{URIs: tt.locations, Providers: makeMapProvidersMap(tt.providers...), Converters: tt.converters})
			assert.NoError(t, err)

			_, errN := resolver.Resolve(context.Background())
			if tt.expectResolveErr {
				assert.Error(t, errN)
				return
			}
			require.NoError(t, errN)

			errW := <-resolver.Watch()
			if tt.expectWatchErr {
				assert.Error(t, errW)
				return
			}
			require.NoError(t, errW)

			_, errC := resolver.Resolve(context.Background())
			if tt.expectCloseErr {
				assert.Error(t, errC)
				return
			}
			require.NoError(t, errN)

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
			errMessage: `file:/test`,
		},
		{
			name:       "file_unix",
			location:   `file:/test`,
			errMessage: `file:/test`,
		},
		{
			name:       "windows_C",
			location:   `C:\test`,
			errMessage: `file:C:\test`,
		},
		{
			name:       "windows_z",
			location:   `z:\test`,
			errMessage: `file:z:\test`,
		},
		{
			name:       "file_windows",
			location:   `file:C:\test`,
			errMessage: `file:C:\test`,
		},
		{
			name:       "invalid_scheme",
			location:   `LL:\test`,
			errMessage: `scheme "LL" is not supported for uri "LL:\\test"`,
		},
	}
	for _, tt := range tests {
		resolver, err := NewResolver(ResolverSettings{
			URIs: []string{tt.location},
			Providers: makeMapProvidersMap(newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (Retrieved, error) {
				return Retrieved{}, errors.New(uri)
			})),
			Converters: nil})
		assert.NoError(t, err)
		_, err = resolver.Resolve(context.Background())
		assert.Contains(t, err.Error(), tt.errMessage, tt.name)
	}
}

func TestResolver(t *testing.T) {
	resolver, err := NewResolver(ResolverSettings{
		URIs:       []string{"mock:"},
		Providers:  makeMapProvidersMap(&mockProvider{retM: newConfFromFile(t, filepath.Join("testdata", "config.yaml"))}),
		Converters: nil})
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

func TestResolverNoLocations(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:       []string{},
		Providers:  makeMapProvidersMap(&mockProvider{}),
		Converters: nil})
	assert.Error(t, err)
}

func TestResolverNoProviders(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:       []string{filepath.Join("testdata", "config.yaml")},
		Providers:  nil,
		Converters: nil})
	assert.Error(t, err)
}

func TestResolverShutdownClosesWatch(t *testing.T) {
	resolver, err := NewResolver(ResolverSettings{
		URIs:       []string{filepath.Join("testdata", "config.yaml")},
		Providers:  makeMapProvidersMap(newFileProvider(t)),
		Converters: nil})
	require.NoError(t, err)
	_, errN := resolver.Resolve(context.Background())
	assert.NoError(t, errN)

	var watcherWG sync.WaitGroup
	watcherWG.Add(1)
	go func() {
		errW, ok := <-resolver.Watch()
		// Channel is closed, no exception
		assert.Nil(t, errW)
		assert.False(t, ok)
		watcherWG.Done()
	}()

	assert.NoError(t, resolver.Shutdown(context.Background()))
	watcherWG.Wait()
}

func makeMapProvidersMap(providers ...Provider) map[string]Provider {
	ret := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
