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
	retM   interface{}
	errR   error
	errS   error
	errW   error
	errC   error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher WatcherFunc) (*Retrieved, error) {
	if m.errR != nil {
		return nil, m.errR
	}
	if m.retM == nil {
		return NewRetrieved(nil)
	}

	watcher(&ChangeEvent{Error: m.errW})
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
	ret    func(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error)
}

func newFileProvider(t testing.TB) Provider {
	return newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(newConfFromFile(t, uri[5:]))
	})
}

func newFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error)) Provider {
	return &fakeProvider{
		scheme: scheme,
		ret:    ret,
	}
}

func (f *fakeProvider) Retrieve(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error) {
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

func TestNewResolverInvalidScheme(t *testing.T) {
	_, err := NewResolver(ResolverSettings{URIs: []string{"s_3:has invalid char"}, Providers: makeMapProvidersMap(&mockProvider{scheme: "s_3"})})
	assert.EqualError(t, err, `invalid uri: "s_3:has invalid char"`)
}

func TestResolverErrors(t *testing.T) {
	tests := []struct {
		name              string
		locations         []string
		providers         []Provider
		converters        []Converter
		expectBuildErr    bool
		expectResolveErr  bool
		expectWatchErr    bool
		expectCloseErr    bool
		expectShutdownErr bool
	}{
		{
			name:           "unsupported location scheme",
			locations:      []string{"mock:", "notsupported:"},
			providers:      []Provider{&mockProvider{}},
			expectBuildErr: true,
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
			name:      "retrieve location not convertable to Conf",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: "invalid value"},
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
			if tt.expectBuildErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

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
		name           string
		location       string
		errMessage     string
		expectBuildErr bool
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
			name:           "invalid_scheme",
			location:       `LL:\test`,
			expectBuildErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(ResolverSettings{
				URIs: []string{tt.location},
				Providers: makeMapProvidersMap(newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
					return nil, errors.New(uri)
				})),
				Converters: nil})
			if tt.expectBuildErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			_, err = resolver.Resolve(context.Background())
			assert.Contains(t, err.Error(), tt.errMessage, tt.name)
		})
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

func TestResolverNewLinesInOpaqueValue(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:       []string{"mock:receivers:\n nop:\n"},
		Providers:  makeMapProvidersMap(&mockProvider{retM: newConfFromFile(t, filepath.Join("testdata", "config.yaml"))}),
		Converters: nil})
	assert.NoError(t, err)
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

func TestResolverExpandEnvVars(t *testing.T) {
	var testCases = []struct {
		name string // test case name (also file name containing config yaml)
	}{
		{name: "expand-with-no-env.yaml"},
		{name: "expand-with-partial-env.yaml"},
		{name: "expand-with-all-env.yaml"},
	}

	envs := map[string]string{
		"EXTRA":                  "some string",
		"EXTRA_MAP_VALUE_1":      "some map value_1",
		"EXTRA_MAP_VALUE_2":      "some map value_2",
		"EXTRA_LIST_MAP_VALUE_1": "some list map value_1",
		"EXTRA_LIST_MAP_VALUE_2": "some list map value_2",
		"EXTRA_LIST_VALUE_1":     "some list value_1",
		"EXTRA_LIST_VALUE_2":     "some list value_2",
	}

	expectedCfgMap := newConfFromFile(t, filepath.Join("testdata", "expand-with-no-env.yaml"))
	fileProvider := newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(newConfFromFile(t, uri[5:]))
	})
	envProvider := newFakeProvider("env", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(envs[uri[4:]])
	})

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := NewResolver(ResolverSettings{URIs: []string{filepath.Join("testdata", test.name)}, Providers: makeMapProvidersMap(fileProvider, envProvider), Converters: nil})
			require.NoError(t, err)

			// Test that expanded configs are the same with the simple config with no env vars.
			cfgMap, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			assert.Equal(t, expectedCfgMap, cfgMap.ToStringMap())
		})
	}
}

func TestResolverDoneNotExpandOldEnvVars(t *testing.T) {
	expectedCfgMap := map[string]interface{}{"test.1": "${EXTRA}", "test.2": "$EXTRA"}
	fileProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(expectedCfgMap)
	})
	envProvider := newFakeProvider("env", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved("some string")
	})
	emptySchemeProvider := newFakeProvider("", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved("some string")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"test:"}, Providers: makeMapProvidersMap(fileProvider, envProvider, emptySchemeProvider), Converters: nil})
	require.NoError(t, err)

	// Test that expanded configs are the same with the simple config with no env vars.
	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedCfgMap, cfgMap.ToStringMap())
}

func TestResolverExpandMapAndSliceValues(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{
			"test_map":   map[string]interface{}{"recv": "${test:MAP_VALUE}"},
			"test_slice": []interface{}{"${test:MAP_VALUE}"}})
	})

	const receiverExtraMapValue = "some map value"
	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverExtraMapValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	expectedMap := map[string]interface{}{
		"test_map":   map[string]interface{}{"recv": receiverExtraMapValue},
		"test_slice": []interface{}{receiverExtraMapValue}}
	assert.Equal(t, expectedMap, cfgMap.ToStringMap())
}

func TestResolverInfiniteExpand(t *testing.T) {
	const receiverValue = "${test:VALUE}"
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{"test": receiverValue})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.ErrorIs(t, err, errTooManyRecursiveExpansions)
}

func TestResolverExpandSliceValueError(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{"test": []interface{}{"${test:VALUE}"}})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(errors.New("invalid value"))
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, "unsupported type=*errors.errorString for retrieved config")
}

func TestResolverExpandMapValueError(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{"test": []interface{}{map[string]interface{}{"test": "${test:VALUE}"}}})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(errors.New("invalid value"))
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, "unsupported type=*errors.errorString for retrieved config")
}

func TestResolverExpandInvalidScheme(t *testing.T) {
	const receiverValue = "${g_c_s:VALUE}"
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{"test": receiverValue})
	})

	testProvider := newFakeProvider("g_c_s", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `invalid uri: "g_c_s:VALUE"`)
}

func TestResolverExpandInvalidOpaqueValue(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]interface{}{"test": []interface{}{map[string]interface{}{"test": "${test:$VALUE}"}}})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(errors.New("invalid value"))
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `the uri "test:$VALUE" contains unsupported characters ('$')`)
}

func makeMapProvidersMap(providers ...Provider) map[string]Provider {
	ret := make(map[string]Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}
