// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package confmap // import "go.opentelemetry.io/collector/confmap"

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolverExpandEnvVars(t *testing.T) {
	testCases := []struct {
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

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			resolver, err := NewResolver(ResolverSettings{URIs: []string{filepath.Join("testdata", tt.name)}, ProviderFactories: []ProviderFactory{fileProvider, envProvider}, ConverterFactories: nil})
			require.NoError(t, err)

			// Test that expanded configs are the same with the simple config with no env vars.
			cfgMap, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			assert.Equal(t, expectedCfgMap, cfgMap.ToStringMap())
		})
	}
}

func TestResolverDoneNotExpandOldEnvVars(t *testing.T) {
	expectedCfgMap := map[string]any{"test.1": "${EXTRA}", "test.2": "$EXTRA", "test.3": "${EXTRA}:${EXTRA}"}
	fileProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(expectedCfgMap)
	})
	envProvider := newFakeProvider("env", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved("some string")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"test:"}, ProviderFactories: []ProviderFactory{fileProvider, envProvider}, ConverterFactories: nil})
	require.NoError(t, err)

	// Test that expanded configs are the same with the simple config with no env vars.
	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expectedCfgMap, cfgMap.ToStringMap())
}

func TestResolverExpandMapAndSliceValues(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{
			"test_map":   map[string]any{"recv": "${test:MAP_VALUE}"},
			"test_slice": []any{"${test:MAP_VALUE}"},
		})
	})

	const receiverExtraMapValue = "some map value"
	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverExtraMapValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
	require.NoError(t, err)

	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	expectedMap := map[string]any{
		"test_map":   map[string]any{"recv": receiverExtraMapValue},
		"test_slice": []any{receiverExtraMapValue},
	}
	assert.Equal(t, expectedMap, cfgMap.ToStringMap())
}

func TestResolverExpandStringValues(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		output          any
		defaultProvider bool
	}{
		// Embedded.
		{
			name:   "NoMatchOldStyle",
			input:  "${HOST}:${PORT}",
			output: "${HOST}:${PORT}",
		},
		{
			name:            "NoMatchOldStyleDefaultProvider",
			input:           "${HOST}:${PORT}",
			output:          "localhost:3044",
			defaultProvider: true,
		},
		{
			name:   "NoMatchOldStyleNoBrackets",
			input:  "${HOST}:$PORT",
			output: "${HOST}:$PORT",
		},
		{
			name:            "NoMatchOldStyleNoBracketsDefaultProvider",
			input:           "${HOST}:$PORT",
			output:          "localhost:$PORT",
			defaultProvider: true,
		},
		{
			name:   "ComplexValue",
			input:  "${env:COMPLEX_VALUE}",
			output: []any{"localhost:3042"},
		},
		{
			name:   "Embedded",
			input:  "${env:HOST}:3043",
			output: "localhost:3043",
		},
		{
			name:   "EmbeddedMulti",
			input:  "${env:HOST}:${env:PORT}",
			output: "localhost:3044",
		},
		{
			name:   "EmbeddedConcat",
			input:  "https://${env:HOST}:3045",
			output: "https://localhost:3045",
		},
		{
			name:   "EmbeddedNewAndOldStyle",
			input:  "${env:HOST}:${PORT}",
			output: "localhost:${PORT}",
		},
		{
			name:            "EmbeddedNewAndOldStyleDefaultProvider",
			input:           "${env:HOST}:${PORT}",
			output:          "localhost:3044",
			defaultProvider: true,
		},
		{
			name:   "Int",
			input:  "test_${env:INT}",
			output: "test_1",
		},
		{
			name:   "Int32",
			input:  "test_${env:INT32}",
			output: "test_32",
		},
		{
			name:   "Int64",
			input:  "test_${env:INT64}",
			output: "test_64",
		},
		{
			name:   "Float32",
			input:  "test_${env:FLOAT32}",
			output: "test_3.25",
		},
		{
			name:   "Float64",
			input:  "test_${env:FLOAT64}",
			output: "test_6.4",
		},
		{
			name:   "Bool",
			input:  "test_${env:BOOL}",
			output: "test_true",
		},
		{
			name:   "Timestamp",
			input:  "test_${env:TIMESTAMP}",
			output: "test_2023-03-20T03:17:55.432328Z",
		},
		{
			name:   "MultipleSameMatches",
			input:  "test_${env:BOOL}_test_${env:BOOL}",
			output: "test_true_test_true",
		},

		// Nested.
		{
			name:   "Nested",
			input:  "${test:localhost:${env:PORT}}",
			output: "localhost:3044",
		},
		{
			name:            "NestedDefaultProvider",
			input:           "${test:localhost:${PORT}}",
			output:          "localhost:3044",
			defaultProvider: true,
		},
		{
			name:   "EmbeddedInNested",
			input:  "${test:${env:HOST}:${env:PORT}}",
			output: "localhost:3044",
		},
		{
			name:            "EmbeddedInNestedDefaultProvider",
			input:           "${test:${HOST}:${PORT}}",
			output:          "localhost:3044",
			defaultProvider: true,
		},
		{
			name:   "EmbeddedAndNested",
			input:  "${test:localhost:${env:PORT}}?os=${env:OS}",
			output: "localhost:3044?os=ubuntu",
		},
		{
			name:   "NestedMultiple",
			input:  "${test:1${test:2${test:3${test:4${test:5${test:6}}}}}}",
			output: "123456",
		},
		// No expand.
		{
			name:   "NoMatchMissingOpeningBracket",
			input:  "env:HOST}",
			output: "env:HOST}",
		},
		{
			name:            "NoMatchMissingOpeningBracketDefaultProvider",
			input:           "env:HOST}",
			output:          "env:HOST}",
			defaultProvider: true,
		},
		{
			name:   "NoMatchMissingClosingBracket",
			input:  "${HOST",
			output: "${HOST",
		},
		{
			name:            "NoMatchMissingClosingBracketDefaultProvider",
			input:           "${HOST",
			output:          "${HOST",
			defaultProvider: true,
		},
		{
			name:   "NoMatchBracketsWithout$",
			input:  "HO{ST}",
			output: "HO{ST}",
		},
		{
			name:            "NoMatchBracketsWithout$DefaultProvider",
			input:           "HO{ST}",
			output:          "HO{ST}",
			defaultProvider: true,
		},
		{
			name:   "NoMatchOnlyMissingClosingBracket",
			input:  "${env:HOST${env:PORT?os=${env:OS",
			output: "${env:HOST${env:PORT?os=${env:OS",
		},
		{
			name:            "NoMatchOnlyMissingClosingBracketDefaultProvider",
			input:           "${env:HOST${env:PORT?os=${env:OS",
			output:          "${env:HOST${env:PORT?os=${env:OS",
			defaultProvider: true,
		},
		{
			name:   "NoMatchOnlyMissingOpeningBracket",
			input:  "env:HOST}env:PORT}?os=env:OS}",
			output: "env:HOST}env:PORT}?os=env:OS}",
		},
		{
			name:            "NoMatchOnlyMissingOpeningBracketDefaultProvider",
			input:           "env:HOST}env:PORT}?os=env:OS}",
			output:          "env:HOST}env:PORT}?os=env:OS}",
			defaultProvider: true,
		},
		{
			name:   "NoMatchCloseBeforeOpen",
			input:  "env:HOST}${env:PORT",
			output: "env:HOST}${env:PORT",
		},
		{
			name:            "NoMatchCloseBeforeOpenDefaultProvider",
			input:           "env:HOST}${env:PORT",
			output:          "env:HOST}${env:PORT",
			defaultProvider: true,
		},
		{
			name:   "NoMatchOldStyleNested",
			input:  "${test:localhost:${PORT}}",
			output: "${test:localhost:${PORT}}",
		},
		// Partial expand.
		{
			name:   "PartialMatchMissingOpeningBracketFirst",
			input:  "env:HOST}${env:PORT}",
			output: "env:HOST}3044",
		},
		{
			name:            "PartialMatchMissingOpeningBracketFirstDefaultProvider",
			input:           "env:HOST}${PORT}",
			output:          "env:HOST}3044",
			defaultProvider: true,
		},
		{
			name:   "PartialMatchMissingOpeningBracketLast",
			input:  "${env:HOST}env:PORT}",
			output: "localhostenv:PORT}",
		},
		{
			name:   "PartialMatchMissingClosingBracketFirst",
			input:  "${env:HOST${env:PORT}",
			output: "${env:HOST3044",
		},
		{
			name:   "PartialMatchMissingClosingBracketLast",
			input:  "${env:HOST}${env:PORT",
			output: "localhost${env:PORT",
		},
		{
			name:   "PartialMatchMultipleMissingOpen",
			input:  "env:HOST}env:PORT}?os=${env:OS}",
			output: "env:HOST}env:PORT}?os=ubuntu",
		},
		{
			name:   "PartialMatchMultipleMissingClosing",
			input:  "${env:HOST${env:PORT?os=${env:OS}",
			output: "${env:HOST${env:PORT?os=ubuntu",
		},
		{
			name:   "PartialMatchMoreClosingBrackets",
			input:  "${env:HOST}}}}}}${env:PORT?os=${env:OS}",
			output: "localhost}}}}}${env:PORT?os=ubuntu",
		},
		{
			name:   "PartialMatchMoreOpeningBrackets",
			input:  "${env:HOST}${${${${${env:PORT}?os=${env:OS}",
			output: "localhost${${${${3044?os=ubuntu",
		},
		{
			name:   "PartialMatchAlternatingMissingOpening",
			input:  "env:HOST}${env:PORT}?os=env:OS}&pr=${env:PR}",
			output: "env:HOST}3044?os=env:OS}&pr=amd",
		},
		{
			name:   "PartialMatchAlternatingMissingClosing",
			input:  "${env:HOST${env:PORT}?os=${env:OS&pr=${env:PR}",
			output: "${env:HOST3044?os=${env:OS&pr=amd",
		},
		{
			name:   "SchemeAfterNoSchemeIsExpanded",
			input:  "${HOST}${env:PORT}",
			output: "${HOST}3044",
		},
		{
			name:            "SchemeAfterNoSchemeIsExpandedDefaultProvider",
			input:           "${HOST}${env:PORT}",
			output:          "localhost3044",
			defaultProvider: true,
		},
		{
			name:   "SchemeBeforeNoSchemeIsExpanded",
			input:  "${env:HOST}${PORT}",
			output: "localhost${PORT}",
		},
		{
			name:            "SchemeBeforeNoSchemeIsExpandedDefaultProvider",
			input:           "${env:HOST}${PORT}",
			output:          "localhost3044",
			defaultProvider: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
				return NewRetrieved(map[string]any{tt.name: tt.input})
			})

			testProvider := newFakeProvider("test", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
				return NewRetrieved(uri[5:])
			})

			envProvider := newEnvProvider()
			set := ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, envProvider, testProvider}, ConverterFactories: nil}
			if tt.defaultProvider {
				set.DefaultScheme = "env"
			}
			resolver, err := NewResolver(set)
			require.NoError(t, err)

			cfgMap, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			assert.Equal(t, map[string]any{tt.name: tt.output}, cfgMap.ToStringMap())
		})
	}
}

func newEnvProvider() ProviderFactory {
	return newFakeProvider("env", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
		// When using `env` as the default scheme for tests, the uri will not include `env:`.
		// Instead of duplicating the switch cases, the scheme is added instead.
		if uri[0:4] != "env:" {
			uri = "env:" + uri
		}
		switch uri {
		case "env:COMPLEX_VALUE":
			return NewRetrievedFromYAML([]byte("[localhost:3042]"))
		case "env:HOST":
			return NewRetrievedFromYAML([]byte("localhost"))
		case "env:TIMESTAMP":
			return NewRetrievedFromYAML([]byte("2023-03-20T03:17:55.432328Z"))
		case "env:OS":
			return NewRetrievedFromYAML([]byte("ubuntu"))
		case "env:PR":
			return NewRetrievedFromYAML([]byte("amd"))
		case "env:PORT":
			return NewRetrievedFromYAML([]byte("3044"))
		case "env:INT":
			return NewRetrievedFromYAML([]byte("1"))
		case "env:INT32":
			return NewRetrieved(int32(32), withStringRepresentation("32"))
		case "env:INT64":
			return NewRetrieved(int64(64), withStringRepresentation("64"))
		case "env:FLOAT32":
			return NewRetrieved(float32(3.25), withStringRepresentation("3.25"))
		case "env:FLOAT64":
			return NewRetrieved(float64(6.4), withStringRepresentation("6.4"))
		case "env:BOOL":
			return NewRetrievedFromYAML([]byte("true"))
		}
		return nil, errors.New("impossible")
	})
}

func TestResolverExpandReturnError(t *testing.T) {
	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "string_value",
			input: "${test:VALUE}",
		},
		{
			name:  "slice_value",
			input: []any{"${test:VALUE}"},
		},
		{
			name:  "map_value",
			input: map[string]any{"test": "${test:VALUE}"},
		},
		{
			name:  "string_embedded_value",
			input: "https://${test:HOST}:3045",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
				return NewRetrieved(map[string]any{tt.name: tt.input})
			})

			myErr := errors.New(tt.name)
			testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
				return nil, myErr
			})

			resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
			require.NoError(t, err)

			_, err = resolver.Resolve(context.Background())
			assert.ErrorIs(t, err, myErr)
		})
	}
}

func TestResolverInfiniteExpand(t *testing.T) {
	const receiverValue = "${test:VALUE}"
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": receiverValue})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.ErrorIs(t, err, errTooManyRecursiveExpansions)
}

func TestResolverExpandInvalidScheme(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": "${g_c_s:VALUE}"})
	})

	testProvider := newFakeProvider("g_c_s", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		panic("must not be called")
	})

	_, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
	assert.ErrorContains(t, err, "invalid 'confmap.Provider' scheme")
}

func TestResolverExpandInvalidOpaqueValue(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": []any{map[string]any{"test": "${test:$VALUE}"}}})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		panic("must not be called")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `the uri "test:$VALUE" contains unsupported characters ('$')`)
}

func TestResolverExpandUnsupportedScheme(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": "${unsupported:VALUE}"})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		panic("must not be called")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, testProvider}, ConverterFactories: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `scheme "unsupported" is not supported for uri "unsupported:VALUE"`)
}

func TestResolverDefaultProviderExpand(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"foo": "${HOST}"})
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, ProviderFactories: []ProviderFactory{provider, newEnvProvider()}, DefaultScheme: "env", ConverterFactories: nil})
	require.NoError(t, err)

	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"foo": "localhost"}, cfgMap.ToStringMap())
}
