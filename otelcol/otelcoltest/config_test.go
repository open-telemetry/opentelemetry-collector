// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcoltest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service/pipelines"
)

type fakeProvider struct {
	scheme string
	ret    func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)
	logger *zap.Logger
}

func (f *fakeProvider) Retrieve(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return f.ret(ctx, uri, watcher)
}

func (f *fakeProvider) Scheme() string {
	return f.scheme
}

func (f *fakeProvider) Shutdown(context.Context) error {
	return nil
}

func newFakeProvider(scheme string, ret func(ctx context.Context, uri string, watcher confmap.WatcherFunc) (*confmap.Retrieved, error)) confmap.ProviderFactory {
	return confmap.NewProviderFactory(func(ps confmap.ProviderSettings) confmap.Provider {
		return &fakeProvider{
			scheme: scheme,
			ret:    ret,
			logger: ps.Logger,
		}
	})
}

func newEnvProvider() confmap.ProviderFactory {
	return newFakeProvider("env", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		// When using `env` as the default scheme for tests, the uri will not include `env:`.
		// Instead of duplicating the switch cases, the scheme is added instead.
		if uri[0:4] != "env:" {
			uri = "env:" + uri
		}
		switch uri {
		case "env:COMPLEX_VALUE":
			return confmap.NewRetrieved([]any{"localhost:3042"})
		case "env:HOST":
			return confmap.NewRetrieved("localhost")
		case "env:OS":
			return confmap.NewRetrieved("ubuntu")
		case "env:PR":
			return confmap.NewRetrieved("amd")
		case "env:PORT":
			return confmap.NewRetrieved(3044)
		case "env:INT":
			return confmap.NewRetrieved(1)
		case "env:INT32":
			return confmap.NewRetrieved(32)
		case "env:INT64":
			return confmap.NewRetrieved(64)
		case "env:FLOAT32":
			return confmap.NewRetrieved(float32(3.25))
		case "env:FLOAT64":
			return confmap.NewRetrieved(float64(6.4))
		case "env:BOOL":
			return confmap.NewRetrieved(true)
		}
		return nil, errors.New("impossible")
	})
}

func newDefaultConfigProviderSettings(t testing.TB, uris []string) otelcol.ConfigProviderSettings {
	fileProvider := newFakeProvider("file", func(_ context.Context, uri string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, uri[5:]))
	})
	return otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: uris,
			ProviderFactories: []confmap.ProviderFactory{
				fileProvider,
				newEnvProvider(),
			},
		},
	}
}

// newConfFromFile creates a new Conf by reading the given file.
func newConfFromFile(t testing.TB, fileName string) map[string]any {
	content, err := os.ReadFile(filepath.Clean(fileName))
	require.NoErrorf(t, err, "unable to read the file %v", fileName)

	var data map[string]any
	require.NoError(t, yaml.Unmarshal(content, &data), "unable to parse yaml")

	return confmap.NewFromStringMap(data).ToStringMap()
}

func TestLoadConfig(t *testing.T) {
	factories, err := NopFactories()
	assert.NoError(t, err)

	set := newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "config.yaml")})

	cfg, err := LoadConfig(filepath.Join("testdata", "config.yaml"), factories, set)
	require.NoError(t, err)

	// Verify extensions.
	require.Len(t, cfg.Extensions, 2)
	assert.Contains(t, cfg.Extensions, component.MustNewID("nop"))
	assert.Contains(t, cfg.Extensions, component.MustNewIDWithName("nop", "myextension"))

	// Verify receivers
	require.Len(t, cfg.Receivers, 2)
	assert.Contains(t, cfg.Receivers, component.MustNewID("nop"))
	assert.Contains(t, cfg.Receivers, component.MustNewIDWithName("nop", "myreceiver"))

	// Verify exporters
	assert.Len(t, cfg.Exporters, 2)
	assert.Contains(t, cfg.Exporters, component.MustNewID("nop"))
	assert.Contains(t, cfg.Exporters, component.MustNewIDWithName("nop", "myexporter"))

	// Verify procs
	assert.Len(t, cfg.Processors, 2)
	assert.Contains(t, cfg.Processors, component.MustNewID("nop"))
	assert.Contains(t, cfg.Processors, component.MustNewIDWithName("nop", "myprocessor"))

	// Verify connectors
	assert.Len(t, cfg.Connectors, 1)
	assert.Contains(t, cfg.Connectors, component.MustNewIDWithName("nop", "myconnector"))

	// Verify service.
	require.Len(t, cfg.Service.Extensions, 1)
	assert.Contains(t, cfg.Service.Extensions, component.MustNewID("nop"))
	require.Len(t, cfg.Service.Pipelines, 1)
	assert.Equal(t,
		&pipelines.PipelineConfig{
			Receivers:  []component.ID{component.MustNewID("nop")},
			Processors: []component.ID{component.MustNewID("nop")},
			Exporters:  []component.ID{component.MustNewID("nop")},
		},
		cfg.Service.Pipelines[component.MustNewID("traces")],
		"Did not load pipeline config correctly")
}

func TestLoadConfigAndValidate(t *testing.T) {
	factories, err := NopFactories()
	assert.NoError(t, err)

	set := newDefaultConfigProviderSettings(t, []string{filepath.Join("testdata", "config.yaml")})

	cfgValidate, errValidate := LoadConfigAndValidate(filepath.Join("testdata", "config.yaml"), factories, set)
	require.NoError(t, errValidate)

	cfg, errLoad := LoadConfig(filepath.Join("testdata", "config.yaml"), factories, set)
	require.NoError(t, errLoad)

	assert.Equal(t, cfg, cfgValidate)
}
