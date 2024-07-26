// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/globalgates"
)

func newConfig(yamlBytes []byte, factories Factories) (*Config, error) {
	var stringMap = map[string]interface{}{}
	err := yaml.Unmarshal(yamlBytes, stringMap)

	if err != nil {
		return nil, err
	}

	conf := confmap.NewFromStringMap(stringMap)

	cfg, err := Unmarshal(conf, factories)
	if err != nil {
		return nil, err
	}

	return &Config{
		Receivers:  cfg.Receivers.Configs(),
		Processors: cfg.Processors.Configs(),
		Exporters:  cfg.Exporters.Configs(),
		Connectors: cfg.Connectors.Configs(),
		Extensions: cfg.Extensions.Configs(),
		Service:    cfg.Service,
	}, nil
}

func TestConfigProviderYaml(t *testing.T) {
	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	uriLocation := "yaml:" + string(yamlBytes)

	yamlProvider := newFakeProvider("yaml", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		var rawConf any
		if err = yaml.Unmarshal(yamlBytes, &rawConf); err != nil {
			return nil, err
		}
		return confmap.NewRetrieved(rawConf)
	})

	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{uriLocation},
			ProviderFactories: []confmap.ProviderFactory{yamlProvider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := nopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)

	configNop, err := newConfig(yamlBytes, factories)
	require.NoError(t, err)

	assert.EqualValues(t, configNop, cfg)
}

func TestConfigProviderFile(t *testing.T) {
	uriLocation := "file:" + filepath.Join("testdata", "otelcol-nop.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, uriLocation[5:]))
	})
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{uriLocation},
			ProviderFactories: []confmap.ProviderFactory{fileProvider},
		},
	}

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	factories, err := nopFactories()
	require.NoError(t, err)

	cfg, err := cp.Get(context.Background(), factories)
	require.NoError(t, err)

	yamlBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	configNop, err := newConfig(yamlBytes, factories)
	require.NoError(t, err)

	assert.EqualValues(t, configNop, cfg)
}

func TestGetConfmap(t *testing.T) {
	uriLocation := "file:" + filepath.Join("testdata", "otelcol-nop.yaml")
	fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
		return confmap.NewRetrieved(newConfFromFile(t, uriLocation[5:]))
	})
	set := ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:              []string{uriLocation},
			ProviderFactories: []confmap.ProviderFactory{fileProvider},
		},
	}

	configBytes, err := os.ReadFile(filepath.Join("testdata", "otelcol-nop.yaml"))
	require.NoError(t, err)

	yamlMap := map[string]any{}
	err = yaml.Unmarshal(configBytes, yamlMap)
	require.NoError(t, err)

	cp, err := NewConfigProvider(set)
	require.NoError(t, err)

	cmp, ok := cp.(ConfmapProvider)
	require.True(t, ok)

	cmap, err := cmp.GetConfmap(context.Background())
	require.NoError(t, err)

	assert.EqualValues(t, yamlMap, cmap.ToStringMap())
}

func TestStrictlyTypedCoda(t *testing.T) {
	tests := []struct {
		basename string
		// isErrFromStrictTypes indicates whether the test should expect an error when the feature gate is
		// disabled. If so, we check that it errs both with and without the feature gate and that the coda is never
		// present.
		isErrFromStrictTypes bool
	}{
		{basename: "weak-implicit-bool-to-string.yaml"},
		{basename: "weak-implicit-int-to-string.yaml"},
		{basename: "weak-implicit-bool-to-int.yaml"},
		{basename: "weak-implicit-string-to-int.yaml"},
		{basename: "weak-implicit-int-to-bool.yaml"},
		{basename: "weak-implicit-string-to-bool.yaml"},
		{basename: "weak-empty-map-to-empty-array.yaml"},
		{basename: "weak-slice-of-maps-to-map.yaml"},
		{basename: "weak-single-element-to-slice.yaml"},
		{
			basename:             "otelcol-invalid-components.yaml",
			isErrFromStrictTypes: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.basename, func(t *testing.T) {
			filename := filepath.Join("testdata", tt.basename)
			uriLocation := "file:" + filename
			fileProvider := newFakeProvider("file", func(_ context.Context, _ string, _ confmap.WatcherFunc) (*confmap.Retrieved, error) {
				return confmap.NewRetrieved(newConfFromFile(t, filename))
			})
			cp, err := NewConfigProvider(ConfigProviderSettings{
				ResolverSettings: confmap.ResolverSettings{
					URIs:              []string{uriLocation},
					ProviderFactories: []confmap.ProviderFactory{fileProvider},
				},
			})
			require.NoError(t, err)
			factories, err := nopFactories()
			require.NoError(t, err)

			// Save the previous value of the feature gate and restore it after the test.
			prev := globalgates.StrictlyTypedInputGate.IsEnabled()
			defer func() {
				require.NoError(t, featuregate.GlobalRegistry().Set(globalgates.StrictlyTypedInputID, prev))
			}()

			// Ensure the error does not appear with the feature gate disabled.
			require.NoError(t, featuregate.GlobalRegistry().Set(globalgates.StrictlyTypedInputID, false))
			_, errWeakTypes := cp.Get(context.Background(), factories)
			if tt.isErrFromStrictTypes {
				require.Error(t, errWeakTypes)
				// Ensure coda is **NOT** present.
				assert.NotContains(t, errWeakTypes.Error(), strictlyTypedMessageCoda)
			} else {
				require.NoError(t, errWeakTypes)
			}

			// Test with the feature gate enabled.
			require.NoError(t, featuregate.GlobalRegistry().Set(globalgates.StrictlyTypedInputID, true))
			_, errStrictTypes := cp.Get(context.Background(), factories)
			require.Error(t, errStrictTypes)
			if tt.isErrFromStrictTypes {
				// Ensure coda is **NOT** present.
				assert.NotContains(t, errStrictTypes.Error(), strictlyTypedMessageCoda)
			} else {
				// Ensure coda is present.
				assert.ErrorContains(t, errStrictTypes, strictlyTypedMessageCoda)
			}
		})
	}
}
