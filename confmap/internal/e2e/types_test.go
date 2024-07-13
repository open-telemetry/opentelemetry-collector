// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2etest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/internal/featuregates"
)

type TargetField string

const (
	TargetFieldInt          TargetField = "int_field"
	TargetFieldString       TargetField = "string_field"
	TargetFieldBool         TargetField = "bool_field"
	TargetFieldInlineString TargetField = "inline_string_field"
)

type Test struct {
	value        string
	targetField  TargetField
	expected     any
	resolveErr   string
	unmarshalErr string
}

type TargetConfig[T any] struct {
	Field T `mapstructure:"field"`
}

func AssertExpectedMatch[T any](t *testing.T, tt Test, conf *confmap.Conf, cfg *TargetConfig[T]) {
	err := conf.Unmarshal(cfg)
	if tt.unmarshalErr != "" {
		require.ErrorContains(t, err, tt.unmarshalErr)
		return
	}
	require.NoError(t, err)
	require.Equal(t, tt.expected, cfg.Field)
}

func TestTypeCasting(t *testing.T) {
	values := []Test{
		{
			value:       "123",
			targetField: TargetFieldInt,
			expected:    123,
		},
		{
			value:       "123",
			targetField: TargetFieldString,
			expected:    "123",
		},
		{
			value:       "123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 123 expansion",
		},
		{
			value:       "0123",
			targetField: TargetFieldInt,
			expected:    83,
		},
		{
			value:       "0123",
			targetField: TargetFieldString,
			expected:    "83",
		},
		{
			value:       "0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 83 expansion",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInt,
			expected:    3735928559,
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldString,
			expected:    "3735928559",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 3735928559 expansion",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldString,
			expected:    "0123",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldInt,
			expected:    83,
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldString,
			expected:    "0123",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:       "'!!str 0123'",
			targetField: TargetFieldString,
			expected:    "!!str 0123",
		},
		{
			value:       "\"!!str 0123\"",
			targetField: TargetFieldInlineString,
			expected:    "inline field with !!str 0123 expansion",
		},
		{
			value:       "t",
			targetField: TargetFieldBool,
			expected:    true,
		},
		{
			value:       "23",
			targetField: TargetFieldBool,
			expected:    true,
		},
		{
			value:       "foo\nbar",
			targetField: TargetFieldString,
			expected:    "foo bar",
		},
		{
			value:       "foo\nbar",
			targetField: TargetFieldInlineString,
			expected:    "inline field with foo bar expansion",
		},
		{
			value:       "\"1111:1111:1111:1111:1111::\"",
			targetField: TargetFieldString,
			expected:    "1111:1111:1111:1111:1111::",
		},
		{
			value:       "\"1111:1111:1111:1111:1111::\"",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 1111:1111:1111:1111:1111:: expansion",
		},
	}

	previousValue := featuregates.StrictlyTypedInputGate.IsEnabled()
	err := featuregate.GlobalRegistry().Set(featuregates.StrictlyTypedInputID, false)
	require.NoError(t, err)
	defer func() {
		err := featuregate.GlobalRegistry().Set(featuregates.StrictlyTypedInputID, previousValue)
		require.NoError(t, err)
	}()

	for _, tt := range values {
		t.Run(tt.value+"/"+string(tt.targetField), func(t *testing.T) {
			testFile := "types_expand.yaml"
			if tt.targetField == TargetFieldInlineString {
				testFile = "types_expand_inline.yaml"
			}

			resolver, err := confmap.NewResolver(confmap.ResolverSettings{
				URIs: []string{filepath.Join("testdata", testFile)},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					envprovider.NewFactory(),
				},
			})
			require.NoError(t, err)
			t.Setenv("ENV", tt.value)

			conf, err := resolver.Resolve(context.Background())
			require.NoError(t, err)

			switch tt.targetField {
			case TargetFieldInt:
				var cfg TargetConfig[int]
				AssertExpectedMatch(t, tt, conf, &cfg)
			case TargetFieldString, TargetFieldInlineString:
				var cfg TargetConfig[string]
				AssertExpectedMatch(t, tt, conf, &cfg)
			case TargetFieldBool:
				var cfg TargetConfig[bool]
				AssertExpectedMatch(t, tt, conf, &cfg)
			default:
				t.Fatalf("unexpected target field %q", tt.targetField)
			}

		})
	}
}

func TestStrictTypeCasting(t *testing.T) {
	values := []Test{
		{
			value:       "123",
			targetField: TargetFieldInt,
			expected:    123,
		},
		{
			value:        "123",
			targetField:  TargetFieldString,
			unmarshalErr: "'field' expected type 'string', got unconvertible type 'int', value: '123'",
		},
		{
			value:       "123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 123 expansion",
		},
		{
			value:       "0123",
			targetField: TargetFieldInt,
			expected:    83,
		},
		{
			value:        "0123",
			targetField:  TargetFieldString,
			unmarshalErr: "'field' expected type 'string', got unconvertible type 'int', value: '83'",
		},
		{
			value:       "0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInt,
			expected:    3735928559,
		},
		{
			value:        "0xdeadbeef",
			targetField:  TargetFieldString,
			unmarshalErr: "'field' expected type 'string', got unconvertible type 'int', value: '3735928559'",
		},
		{
			value:       "0xdeadbeef",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0xdeadbeef expansion",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldString,
			expected:    "0123",
		},
		{
			value:        "\"0123\"",
			targetField:  TargetFieldInt,
			unmarshalErr: "'field' expected type 'int', got unconvertible type 'string', value: '0123'",
		},
		{
			value:       "\"0123\"",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldString,
			expected:    "0123",
		},
		{
			value:       "!!str 0123",
			targetField: TargetFieldInlineString,
			expected:    "inline field with 0123 expansion",
		},
		{
			value:        "t",
			targetField:  TargetFieldBool,
			unmarshalErr: "'field' expected type 'bool', got unconvertible type 'string', value: 't'",
		},
		{
			value:        "23",
			targetField:  TargetFieldBool,
			unmarshalErr: "'field' expected type 'bool', got unconvertible type 'int', value: '23'",
		},
		{
			value:       "{\"field\": 123}",
			targetField: TargetFieldInlineString,
			resolveErr:  "retrieved value does not have unambiguous string representation",
		},
		{
			value:       "1111:1111:1111:1111:1111::",
			targetField: TargetFieldInlineString,
			resolveErr:  "retrieved value does not have unambiguous string representation",
		},
		{
			value:        "1111:1111:1111:1111:1111::",
			targetField:  TargetFieldString,
			unmarshalErr: "'field' expected type 'string', got unconvertible type 'map[string]interface {}', value: 'map[1111:1111:1111:1111:1111::<nil>]'",
		},
	}

	previousValue := featuregates.StrictlyTypedInputGate.IsEnabled()
	err := featuregate.GlobalRegistry().Set(featuregates.StrictlyTypedInputID, true)
	require.NoError(t, err)
	defer func() {
		err := featuregate.GlobalRegistry().Set(featuregates.StrictlyTypedInputID, previousValue)
		require.NoError(t, err)
	}()

	for _, tt := range values {
		t.Run(tt.value+"/"+string(tt.targetField), func(t *testing.T) {
			testFile := "types_expand.yaml"
			if tt.targetField == TargetFieldInlineString {
				testFile = "types_expand_inline.yaml"
			}

			resolver, err := confmap.NewResolver(confmap.ResolverSettings{
				URIs: []string{filepath.Join("testdata", testFile)},
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					envprovider.NewFactory(),
				},
			})
			require.NoError(t, err)
			t.Setenv("ENV", tt.value)

			conf, err := resolver.Resolve(context.Background())
			if tt.resolveErr != "" {
				require.ErrorContains(t, err, tt.resolveErr)
				return
			}
			require.NoError(t, err)

			switch tt.targetField {
			case TargetFieldInt:
				var cfg TargetConfig[int]
				AssertExpectedMatch(t, tt, conf, &cfg)
			case TargetFieldString, TargetFieldInlineString:
				var cfg TargetConfig[string]
				AssertExpectedMatch(t, tt, conf, &cfg)
			case TargetFieldBool:
				var cfg TargetConfig[bool]
				AssertExpectedMatch(t, tt, conf, &cfg)
			default:
				t.Fatalf("unexpected target field %q", tt.targetField)
			}

		})
	}
}
