// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

// Test_EscapedEnvVars tests that the resolver supports escaped env vars working together with expand converter.
func Test_EscapedEnvVars_NoDefaultScheme(t *testing.T) {
	const expandedValue = "some expanded value"
	t.Setenv("ENV_VALUE", expandedValue)

	expectedMap := map[string]any{
		"test_map": map[string]any{
			"key1":  "$ENV_VALUE",
			"key2":  "$$ENV_VALUE",
			"key3":  "$${ENV_VALUE}",
			"key4":  "some${ENV_VALUE}text",
			"key5":  "some${ENV_VALUE}text",
			"key6":  "${ONE}${TWO}",
			"key7":  "text$",
			"key8":  "$",
			"key9":  "${1}${env:2}",
			"key10": "some${env:ENV_VALUE}text",
			"key11": "${env:${ENV_VALUE}}",
			"key12": "${env:${ENV_VALUE}}",
			"key13": "env:MAP_VALUE_2}${ENV_VALUE}{",
			"key14": "$" + expandedValue,
			"key15": "$ENV_VALUE",
		},
	}

	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs:              []string{filepath.Join("testdata", "expand-escaped-env.yaml")},
		ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
		DefaultScheme:     "",
	})
	require.NoError(t, err)

	// Test that expanded configs are the same with the simple config with no env vars.
	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	m := cfgMap.ToStringMap()
	assert.Equal(t, expectedMap, m)
}

// Test_EscapedEnvVars tests that the resolver supports escaped env vars working together with expand converter.
func Test_EscapedEnvVars_DefaultScheme(t *testing.T) {
	const expandedValue = "some expanded value"
	t.Setenv("ENV_VALUE", expandedValue)

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
			"key15": "$ENV_VALUE",
		},
	}

	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs:              []string{filepath.Join("testdata", "expand-escaped-env.yaml")},
		ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory(), envprovider.NewFactory()},
		DefaultScheme:     "env",
	})
	require.NoError(t, err)

	// Test that expanded configs are the same with the simple config with no env vars.
	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	m := cfgMap.ToStringMap()
	assert.Equal(t, expectedMap, m)
}
