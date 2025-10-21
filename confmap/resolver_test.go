// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	yaml "go.yaml.in/yaml/v3"

	"go.opentelemetry.io/collector/confmap/internal"
	"go.opentelemetry.io/collector/featuregate"
)

type mockProvider struct {
	scheme    string
	retM      any
	errR      error
	errS      error
	errW      error
	closeFunc func(ctx context.Context) error
}

func (m *mockProvider) Retrieve(_ context.Context, _ string, watcher WatcherFunc) (*Retrieved, error) {
	if m.errR != nil {
		return nil, m.errR
	}
	if m.retM == nil {
		return NewRetrieved(nil)
	}

	watcher(&ChangeEvent{Error: m.errW})
	return NewRetrieved(m.retM, WithRetrievedClose(m.closeFunc))
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

func newMockProvider(m *mockProvider) ProviderFactory {
	return NewProviderFactory(func(_ ProviderSettings) Provider {
		return m
	})
}

type fakeProvider struct {
	scheme string
	ret    func(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error)
	logger *zap.Logger
}

func newFileProvider(tb testing.TB) ProviderFactory {
	return newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(newConfFromFile(tb, uri[5:]))
	})
}

func newFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error)) ProviderFactory {
	return NewProviderFactory(func(ps ProviderSettings) Provider {
		return &fakeProvider{
			scheme: scheme,
			ret:    ret,
			logger: ps.Logger,
		}
	})
}

func newObservableFileProvider(tb testing.TB) (ProviderFactory, *fakeProvider) {
	return newObservableProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(newConfFromFile(tb, uri[5:]))
	})
}

func newObservableProvider(scheme string, ret func(ctx context.Context, uri string, watcher WatcherFunc) (*Retrieved, error)) (ProviderFactory, *fakeProvider) {
	fp := &fakeProvider{
		scheme: scheme,
		ret:    ret,
	}
	return NewProviderFactory(func(ps ProviderSettings) Provider {
		fp.logger = ps.Logger
		return fp
	}), fp
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

func TestNewResolverInvalidSchemeInURI(t *testing.T) {
	_, err := NewResolver(ResolverSettings{URIs: []string{"s_3:has invalid char"}, ProviderFactories: []ProviderFactory{newMockProvider(&mockProvider{scheme: "s3"})}})
	assert.EqualError(t, err, `invalid uri: "s_3:has invalid char"`)
}

func TestNewResolverDuplicateScheme(t *testing.T) {
	_, err := NewResolver(ResolverSettings{URIs: []string{"mock:something"}, ProviderFactories: []ProviderFactory{newMockProvider(&mockProvider{scheme: "mock"}), newMockProvider(&mockProvider{scheme: "mock"})}})
	assert.EqualError(t, err, `duplicate 'confmap.Provider' scheme "mock"`)
}

func TestResolverErrors(t *testing.T) {
	tests := []struct {
		name              string
		locations         []string
		providers         []Provider
		converters        []Converter
		defaultScheme     string
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
			name:      "default scheme not found",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", errR: errors.New("retrieve_err")},
			},
			defaultScheme:  "missing",
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
			name:      "retrieve location not convertible to Conf",
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
				&mockProvider{scheme: "err", retM: map[string]any{}, errW: errors.New("watch_err")},
			},
			expectWatchErr: true,
		},
		{
			name:      "close error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: map[string]any{}, closeFunc: func(context.Context) error { return errors.New("close_err") }},
			},
			expectCloseErr: true,
		},
		{
			name:      "shutdown error",
			locations: []string{"mock:", "err:"},
			providers: []Provider{
				&mockProvider{},
				&mockProvider{scheme: "err", retM: map[string]any{}, errS: errors.New("close_err")},
			},
			expectShutdownErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockProviderFuncs := make([]ProviderFactory, len(tt.providers))
			for i, provider := range tt.providers {
				p := provider
				mockProviderFuncs[i] = NewProviderFactory(func(_ ProviderSettings) Provider { return p })
			}
			converterFuncs := make([]ConverterFactory, len(tt.converters))
			for i, converter := range tt.converters {
				c := converter
				converterFuncs[i] = NewConverterFactory(func(_ ConverterSettings) Converter { return c })
			}
			resolver, err := NewResolver(ResolverSettings{URIs: tt.locations, ProviderFactories: mockProviderFuncs, DefaultScheme: tt.defaultScheme, ConverterFactories: converterFuncs})
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
			location:   `c:\test`,
			errMessage: `file:c:\test`,
		},
		{
			name:       "windows_z",
			location:   `z:\test`,
			errMessage: `file:z:\test`,
		},
		{
			name:       "file_windows",
			location:   `file:c:\test`,
			errMessage: `file:c:\test`,
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
				ProviderFactories: []ProviderFactory{
					newFakeProvider("file", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
						return nil, errors.New(uri)
					}),
				},
				ConverterFactories: nil,
			})
			if tt.expectBuildErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			_, err = resolver.Resolve(context.Background())
			assert.ErrorContains(t, err, tt.errMessage, tt.name)
		})
	}
}

func TestResolver(t *testing.T) {
	numCalls := atomic.Int32{}
	resolver, err := NewResolver(ResolverSettings{
		URIs: []string{"mock:"},
		ProviderFactories: []ProviderFactory{
			newMockProvider(&mockProvider{retM: map[string]any{}, closeFunc: func(context.Context) error {
				numCalls.Add(1)
				return nil
			}}),
		},
		ConverterFactories: nil,
	})
	require.NoError(t, err)
	_, errN := resolver.Resolve(context.Background())
	require.NoError(t, errN)
	assert.Equal(t, int32(0), numCalls.Load())

	errW := <-resolver.Watch()
	require.NoError(t, errW)

	// Repeat Resolve/Watch.

	_, errN = resolver.Resolve(context.Background())
	require.NoError(t, errN)
	assert.Equal(t, int32(1), numCalls.Load())

	errW = <-resolver.Watch()
	require.NoError(t, errW)

	_, errN = resolver.Resolve(context.Background())
	require.NoError(t, errN)
	assert.Equal(t, int32(2), numCalls.Load())

	errC := resolver.Shutdown(context.Background())
	require.NoError(t, errC)
	assert.Equal(t, int32(3), numCalls.Load())
}

func TestResolverNewLinesInOpaqueValue(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:               []string{"mock:receivers:\n nop:\n"},
		ProviderFactories:  []ProviderFactory{newMockProvider(&mockProvider{retM: map[string]any{}})},
		ConverterFactories: nil,
	})
	assert.NoError(t, err)
}

func TestResolverNoLocations(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:               []string{},
		ProviderFactories:  []ProviderFactory{newMockProvider(&mockProvider{})},
		ConverterFactories: nil,
	})
	assert.Error(t, err)
}

func TestResolverNoProviders(t *testing.T) {
	_, err := NewResolver(ResolverSettings{
		URIs:               []string{filepath.Join("testdata", "config.yaml")},
		ProviderFactories:  nil,
		ConverterFactories: nil,
	})
	assert.Error(t, err)
}

func TestResolverShutdownClosesWatch(t *testing.T) {
	resolver, err := NewResolver(ResolverSettings{
		URIs:               []string{filepath.Join("testdata", "config.yaml")},
		ProviderFactories:  []ProviderFactory{newFileProvider(t)},
		ConverterFactories: nil,
	})
	require.NoError(t, err)
	_, errN := resolver.Resolve(context.Background())
	require.NoError(t, errN)

	var watcherWG sync.WaitGroup
	watcherWG.Add(1)
	go func() {
		errW, ok := <-resolver.Watch()
		// Channel is closed, no exception
		assert.NoError(t, errW)
		assert.False(t, ok)
		watcherWG.Done()
	}()

	require.NoError(t, resolver.Shutdown(context.Background()))
	watcherWG.Wait()
}

func TestProvidesDefaultLogger(t *testing.T) {
	factory, provider := newObservableFileProvider(t)
	_, err := NewResolver(ResolverSettings{
		URIs:              []string{filepath.Join("testdata", "config.yaml")},
		ProviderFactories: []ProviderFactory{factory},
		ConverterFactories: []ConverterFactory{NewConverterFactory(func(set ConverterSettings) Converter {
			assert.NotNil(t, set.Logger)
			return &mockConverter{}
		})},
	})
	require.NoError(t, err)
	require.NotNil(t, provider.logger)
}

func TestResolverDefaultProviderSet(t *testing.T) {
	envProvider := newEnvProvider()
	fileProvider := newFileProvider(t)

	r, err := NewResolver(ResolverSettings{
		URIs:              []string{"env:"},
		ProviderFactories: []ProviderFactory{fileProvider, envProvider},
		DefaultScheme:     "env",
	})
	require.NoError(t, err)
	assert.NotNil(t, r.defaultScheme)
	_, ok := r.providers["env"]
	assert.True(t, ok)
}

type mergeTest struct {
	Name        string           `yaml:"name"`
	AppendPaths []string         `yaml:"append_paths"`
	Configs     []map[string]any `yaml:"configs"`
	Expected    map[string]any   `yaml:"expected"`
}

func TestMergeFunctionality(t *testing.T) {
	tests := []struct {
		name         string
		scenarioFile string
		flagEnabled  bool
	}{
		{
			name:         "feature-flag-enabled",
			scenarioFile: "testdata/merge-append-scenarios.yaml",
			flagEnabled:  true,
		},
		{
			name:         "feature-flag-disabled",
			scenarioFile: "testdata/merge-append-scenarios-featuregate-disabled.yaml",
			flagEnabled:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.flagEnabled {
				require.NoError(t, featuregate.GlobalRegistry().Set(internal.EnableMergeAppendOption.ID(), true))
				defer func() {
					// Restore previous value.
					require.NoError(t, featuregate.GlobalRegistry().Set(internal.EnableMergeAppendOption.ID(), false))
				}()
			}
			runScenario(t, tt.scenarioFile)
		})
	}
}

func runScenario(t *testing.T, path string) {
	yamlData, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	var testcases []*mergeTest
	err = yaml.Unmarshal(yamlData, &testcases)
	require.NoError(t, err)
	for _, tt := range testcases {
		t.Run(tt.Name, func(t *testing.T) {
			configFiles := make([]string, 0)
			for _, c := range tt.Configs {
				// store configs into a temp file. This makes it easier for us to test feature gate functionality
				file, err := os.CreateTemp(t.TempDir(), "*.yaml")
				defer func() { require.NoError(t, file.Close()) }()
				require.NoError(t, err)
				b, err := json.Marshal(c)
				require.NoError(t, err)
				n, err := file.Write(b)
				require.NoError(t, err)
				require.Positive(t, n)
				configFiles = append(configFiles, file.Name())
			}

			resolver, err := NewResolver(ResolverSettings{
				URIs:              configFiles,
				ProviderFactories: []ProviderFactory{newFileProvider(t)},
				DefaultScheme:     "file",
			})
			require.NoError(t, err)
			conf, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			mergedConf := conf.ToStringMap()
			require.Truef(t, reflect.DeepEqual(mergedConf, tt.Expected), "Exp: %s\nGot: %s", tt.Expected, mergedConf)
		})
	}
}

// newConfFromFile creates a new Conf by reading the given file.
func newConfFromFile(tb testing.TB, fileName string) map[string]any {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(tb, err, "unable to read the file %v", fileName)

	var data map[string]any
	require.NoError(tb, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return NewFromStringMap(data).ToStringMap()
}

type provider struct {
	wg sync.WaitGroup
}

func newRaceDetectorProvider() ProviderFactory {
	return NewProviderFactory(func(_ ProviderSettings) Provider {
		return &provider{}
	})
}

func (p *provider) Retrieve(_ context.Context, _ string, watcher WatcherFunc) (*Retrieved, error) {
	p.wg.Add(1)
	go func() {
		// mock a config change event and wait for goroutine to return.
		defer p.wg.Done()
		watcher(&ChangeEvent{})
	}()
	return NewRetrieved(map[string]any{})
}

func (p *provider) Scheme() string {
	return "race"
}

func (p *provider) Shutdown(context.Context) error {
	p.wg.Wait()
	return nil
}

func TestProviderRaceCondition(t *testing.T) {
	resolver, err := NewResolver(ResolverSettings{
		URIs: []string{"race:"},
		ProviderFactories: []ProviderFactory{
			newRaceDetectorProvider(),
		},
		ConverterFactories: nil,
	})
	require.NoError(t, err)
	c, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	require.NotNil(t, c)
	require.NoError(t, resolver.Shutdown(context.Background()))
}
