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

func Test_EscapedEnvVars_NoDefaultScheme(t *testing.T) {
	const expandedValue = "some expanded value"
	t.Setenv("ENV_VALUE", expandedValue)
	t.Setenv("ENV_LIST", "['$$ESCAPE_ME','$${ESCAPE_ME}','$${env:ESCAPE_ME}']")
	t.Setenv("ENV_MAP", "{'key1':'$$ESCAPE_ME','key2':'$${ESCAPE_ME}','key3':'$${env:ESCAPE_ME}'}")
	t.Setenv("ENV_NESTED_DOLLARSIGN", "here is 1 $$")
	t.Setenv("ENV_NESTED_DOLLARSIGN_ESCAPED", "here are 2 $$$")
	t.Setenv("ENV_EXPAND_NESTED", "${env:ENV_VALUE} came from nested expansion")

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
			"key16": []any{"$ESCAPE_ME", "${ESCAPE_ME}", "${env:ESCAPE_ME}"},
			"key17": map[string]any{"key1": "$ESCAPE_ME", "key2": "${ESCAPE_ME}", "key3": "${env:ESCAPE_ME}"},
			"key18": "here is 1 $",
			"key19": "here are 2 $$",
			"key20": "some expanded value came from nested expansion",
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

func Test_EscapedEnvVars_DefaultScheme(t *testing.T) {
	const expandedValue = "some expanded value"
	t.Setenv("ENV_VALUE", expandedValue)
	t.Setenv("ENV_LIST", "['$$ESCAPE_ME','$${ESCAPE_ME}','$${env:ESCAPE_ME}']")
	t.Setenv("ENV_MAP", "{'key1':'$$ESCAPE_ME','key2':'$${ESCAPE_ME}','key3':'$${env:ESCAPE_ME}'}")
	t.Setenv("ENV_NESTED_DOLLARSIGN", "here is 1 $$")
	t.Setenv("ENV_NESTED_DOLLARSIGN_ESCAPED", "here are 2 $$$")
	t.Setenv("ENV_EXPAND_NESTED", "${env:ENV_VALUE} came from nested expansion")

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
			"key16": []any{"$ESCAPE_ME", "${ESCAPE_ME}", "${env:ESCAPE_ME}"},
			"key17": map[string]any{"key1": "$ESCAPE_ME", "key2": "${ESCAPE_ME}", "key3": "${env:ESCAPE_ME}"},
			"key18": "here is 1 $",
			"key19": "here are 2 $$",
			"key20": "some expanded value came from nested expansion",
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
