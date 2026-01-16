// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/internal"
	"go.opentelemetry.io/collector/featuregate"
)

type testHeadersConfig struct {
	Headers configopaque.MapList `mapstructure:"headers"`
}

// TestMapListWithExpandedValue tests that MapList can handle ExpandedValue
// from environment variable expansion
func TestMapListWithExpandedValue(t *testing.T) {
	// Simulate what happens when ${env:TOKEN} is expanded
	// The confmap will contain an ExpandedValue instead of a plain string
	data := map[string]any{
		"headers": []any{
			map[string]any{
				"name": "Authorization",
				"value": internal.ExpandedValue{
					Value:    "Bearer secret-token",
					Original: "Bearer secret-token",
				},
			},
		},
	}

	conf := confmap.NewFromStringMap(data)
	var tc testHeadersConfig
	err := conf.Unmarshal(&tc)
	require.NoError(t, err)

	val, ok := tc.Headers.Get("Authorization")
	require.True(t, ok)
	require.Equal(t, configopaque.String("Bearer secret-token"), val)
}

// TestMapListWithExpandedValueIntValue tests an ExpandedValue with an integer Value
func TestMapListWithExpandedValueIntValue(t *testing.T) {
	// Simulate what happens when expanding a value that parses as an int
	data := map[string]any{
		"headers": []any{
			map[string]any{
				"name": "X-Port",
				"value": internal.ExpandedValue{
					Value:    8080,   // Value is parsed as int
					Original: "8080", // Original is string
				},
			},
		},
	}

	originalState := internal.NewExpandedValueSanitizer.IsEnabled()
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(internal.NewExpandedValueSanitizer.ID(), originalState))
	}()

	require.NoError(t, featuregate.GlobalRegistry().Set(internal.NewExpandedValueSanitizer.ID(), true))

	conf := confmap.NewFromStringMap(data)
	var tc testHeadersConfig
	err := conf.Unmarshal(&tc)
	require.NoError(t, err)

	val, ok := tc.Headers.Get("X-Port")
	require.True(t, ok)
	require.Equal(t, configopaque.String("8080"), val)

	require.NoError(t, featuregate.GlobalRegistry().Set(internal.NewExpandedValueSanitizer.ID(), false))

	// This will fail because when reverting to old behavior, ExpandedValues get decoded at collection time and doesn't
	// take struct collections into account.
	err = conf.Unmarshal(&tc)
	require.Error(t, err)
}

// TestDirectConfigopaqueStringWithExpandedValueIntValue tests that direct unmarshaling works
func TestDirectConfigopaqueStringWithExpandedValueIntValue(t *testing.T) {
	type testConfig struct {
		Value configopaque.String `mapstructure:"value"`
	}

	// Direct configopaque.String field (not in a map/slice structure)
	data := map[string]any{
		"value": internal.ExpandedValue{
			Value:    8080,
			Original: "8080",
		},
	}

	conf := confmap.NewFromStringMap(data)
	var tc testConfig
	err := conf.Unmarshal(&tc)
	// This should work because useExpandValue detects the target is a string
	require.NoError(t, err)
	require.Equal(t, configopaque.String("8080"), tc.Value)
}

// TestStringyStructureWithExpandedValue tests the isStringyStructure path in useExpandValue
func TestStringyStructureWithExpandedValue(t *testing.T) {
	type testConfig struct {
		Tags []string `mapstructure:"tags"`
	}

	data := map[string]any{
		"tags": []any{
			internal.ExpandedValue{
				Value:    8080,
				Original: "8080",
			},
		},
	}

	originalState := internal.NewExpandedValueSanitizer.IsEnabled()
	defer func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(internal.NewExpandedValueSanitizer.ID(), originalState))
	}()

	// With feature gate disabled, useExpandValue should detect []string as stringy
	require.NoError(t, featuregate.GlobalRegistry().Set(internal.NewExpandedValueSanitizer.ID(), false))

	conf := confmap.NewFromStringMap(data)
	var tc testConfig
	err := conf.Unmarshal(&tc)
	require.NoError(t, err)
	require.Equal(t, []string{"8080"}, tc.Tags)
}
