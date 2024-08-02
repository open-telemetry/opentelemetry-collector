// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
)

// Test_EscapedEnvVars tests that the resolver supports escaped env vars working together with expand converter.
func Test_EscapedEnvVars(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
	}{
		{
			name:   "no_default_scheme",
			scheme: "",
		},
		{
			name:   "env",
			scheme: "env",
		},
	}

	const expandedValue = "some expanded value"
	t.Setenv("ENV_VALUE", expandedValue)

	expectedFailures := map[string]string{
		"$ENV_VALUE":   "variable substitution using $VAR has been deprecated in favor of ${VAR} and ${env:VAR}",
		"$$$ENV_VALUE": "variable substitution using $VAR has been deprecated in favor of ${VAR} and ${env:VAR}",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedMap := map[string]any{
				"test_map": map[string]any{
					"key1":  "$ENV_VALUE",
					"key2":  "$$ENV_VALUE",
					"key3":  "$" + expandedValue,
					"key4":  "some" + expandedValue + "text",
					"key5":  "some${ENV_VALUE}text",
					"key6":  "${ONE}${TWO}",
					"key7":  "text$",
					"key8":  "$",
					"key9":  "${1}${env:2}",
					"key10": "some${env:ENV_VALUE}text",
					"key11": "${env:" + expandedValue + "}",
					"key12": "${env:${ENV_VALUE}}",
					"key13": "env:MAP_VALUE_2}${ENV_VALUE}{",
					"key14": "$" + expandedValue,
				},
			}

			resolver, err := confmap.NewResolver(confmap.ResolverSettings{
				URIs:               []string{filepath.Join("testdata", "expand-escaped-env.yaml")},
				ProviderFactories:  []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
				ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
				DefaultScheme:      tt.scheme,
			})
			require.NoError(t, err)

			// Test that expanded configs are the same with the simple config with no env vars.
			cfgMap, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			m := cfgMap.ToStringMap()
			assert.Equal(t, expectedMap, m)

			for val, expectedErr := range expectedFailures {
				resolver, err = confmap.NewResolver(confmap.ResolverSettings{
					URIs:               []string{fmt.Sprintf("yaml: test: %s", val)},
					ProviderFactories:  []confmap.ProviderFactory{yamlprovider.NewFactory(), envprovider.NewFactory()},
					ConverterFactories: []confmap.ConverterFactory{expandconverter.NewFactory()},
					DefaultScheme:      tt.scheme,
				})
				require.NoError(t, err)
				_, err := resolver.Resolve(context.Background())
				require.ErrorContains(t, err, expectedErr)
			}
		})
	}
}
