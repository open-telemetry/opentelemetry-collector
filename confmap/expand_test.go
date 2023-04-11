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
	expectedCfgMap := map[string]any{"test.1": "${EXTRA}", "test.2": "$EXTRA", "test.3": "${EXTRA}:${EXTRA}"}
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
		return NewRetrieved(map[string]any{
			"test_map":   map[string]any{"recv": "${test:MAP_VALUE}"},
			"test_slice": []any{"${test:MAP_VALUE}"}})
	})

	const receiverExtraMapValue = "some map value"
	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(receiverExtraMapValue)
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	cfgMap, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	expectedMap := map[string]any{
		"test_map":   map[string]any{"recv": receiverExtraMapValue},
		"test_slice": []any{receiverExtraMapValue}}
	assert.Equal(t, expectedMap, cfgMap.ToStringMap())
}

func TestResolverExpandStringValues(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output any
	}{
		// Embedded.
		{
			name:   "test_no_match_old",
			input:  "${HOST}:${PORT}",
			output: "${HOST}:${PORT}",
		},
		{
			name:   "test_no_match_old_no_brackets",
			input:  "${HOST}:$PORT",
			output: "${HOST}:$PORT",
		},
		{
			name:   "test_match_value",
			input:  "${env:COMPLEX_VALUE}",
			output: []any{"localhost:3042"},
		},
		{
			name:   "test_match_embedded",
			input:  "${env:HOST}:3043",
			output: "localhost:3043",
		},
		{
			name:   "test_match_embedded_multi",
			input:  "${env:HOST}:${env:PORT}",
			output: "localhost:3044",
		},
		{
			name:   "test_match_embedded_concat",
			input:  "https://${env:HOST}:3045",
			output: "https://localhost:3045",
		},
		{
			name:   "test_match_embedded_new_and_old",
			input:  "${env:HOST}:${PORT}",
			output: "localhost:${PORT}",
		},
		{
			name:   "test_match_int",
			input:  "test_${env:INT}",
			output: "test_1",
		},
		{
			name:   "test_match_int32",
			input:  "test_${env:INT32}",
			output: "test_32",
		},
		{
			name:   "test_match_int64",
			input:  "test_${env:INT64}",
			output: "test_64",
		},
		{
			name:   "test_match_float32",
			input:  "test_${env:FLOAT32}",
			output: "test_3.25",
		},
		{
			name:   "test_match_float64",
			input:  "test_${env:FLOAT64}",
			output: "test_6.4",
		},
		{
			name:   "test_match_bool",
			input:  "test_${env:BOOL}",
			output: "test_true",
		},
		// Nested.
		{
			name:   "test_nested",
			input:  "${test:localhost:${env:PORT}}",
			output: "localhost:3044",
		},
		{
			name:   "test_embedded_in_nested",
			input:  "${test:${env:HOST}:${env:PORT}}",
			output: "localhost:3044",
		},
		{
			name:   "test_embedded_and_nested",
			input:  "${test:localhost:${env:PORT}}?os=${env:OS}",
			output: "localhost:3044?os=ubuntu",
		},
		{
			name:   "test_nested_multiple",
			input:  "${test:1${test:2${test:3${test:4${test:5${test:6}}}}}}",
			output: "123456",
		},
		// No expand.
		{
			name:   "no_opening_bracket",
			input:  "env:HOST}",
			output: "env:HOST}",
		},
		{
			name:   "no_closing_bracket",
			input:  "${HOST",
			output: "${HOST",
		},
		{
			name:   "brackets_without_$",
			input:  "HO{ST}",
			output: "HO{ST}",
		},
		{
			name:   "only_missing_closing",
			input:  "${env:HOST${env:PORT?os=${env:OS",
			output: "${env:HOST${env:PORT?os=${env:OS",
		},
		{
			name:   "only_missing_opening",
			input:  "env:HOST}env:PORT}?os=env:OS}",
			output: "env:HOST}env:PORT}?os=env:OS}",
		},
		{
			name:   "close_before_open",
			input:  "env:HOST}${env:PORT",
			output: "env:HOST}${env:PORT",
		},
		{
			name:   "old_env_var_nested",
			input:  "${test:localhost:${PORT}}",
			output: "${test:localhost:${PORT}}",
		},
		// Partial expand.
		{
			name:   "missing_opening_bracket_first",
			input:  "env:HOST}${env:PORT}",
			output: "env:HOST}3044",
		},
		{
			name:   "missing_opening_bracket_last",
			input:  "${env:HOST}env:PORT}",
			output: "localhostenv:PORT}",
		},
		{
			name:   "missing_closing_bracket_first",
			input:  "${env:HOST${env:PORT}",
			output: "${env:HOST3044",
		},
		{
			name:   "missing_closing_bracket_last",
			input:  "${env:HOST}${env:PORT",
			output: "localhost${env:PORT",
		},
		{
			name:   "multiple_missing_open",
			input:  "env:HOST}env:PORT}?os=${env:OS}",
			output: "env:HOST}env:PORT}?os=ubuntu",
		},
		{
			name:   "multiple_missing_closing",
			input:  "${env:HOST${env:PORT?os=${env:OS}",
			output: "${env:HOST${env:PORT?os=ubuntu",
		},
		{
			name:   "more_closing_brackets",
			input:  "${env:HOST}}}}}}${env:PORT?os=${env:OS}",
			output: "localhost}}}}}${env:PORT?os=ubuntu",
		},
		{
			name:   "more_opening_brackets",
			input:  "${env:HOST}${${${${${env:PORT}?os=${env:OS}",
			output: "localhost${${${${3044?os=ubuntu",
		},
		{
			name:   "alternating_missing_opening",
			input:  "env:HOST}${env:PORT}?os=env:OS}&pr=${env:PR}",
			output: "env:HOST}3044?os=env:OS}&pr=amd",
		},
		{
			name:   "alternating_missing_closing",
			input:  "${env:HOST${env:PORT}?os=${env:OS&pr=${env:PR}",
			output: "${env:HOST3044?os=${env:OS&pr=amd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
				return NewRetrieved(map[string]any{tt.name: tt.input})
			})

			envProvider := newFakeProvider("env", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
				switch uri {
				case "env:COMPLEX_VALUE":
					return NewRetrieved([]any{"localhost:3042"})
				case "env:HOST":
					return NewRetrieved("localhost")
				case "env:OS":
					return NewRetrieved("ubuntu")
				case "env:PR":
					return NewRetrieved("amd")
				case "env:PORT":
					return NewRetrieved(3044)
				case "env:INT":
					return NewRetrieved(1)
				case "env:INT32":
					return NewRetrieved(32)
				case "env:INT64":
					return NewRetrieved(64)
				case "env:FLOAT32":
					return NewRetrieved(float32(3.25))
				case "env:FLOAT64":
					return NewRetrieved(float64(6.4))
				case "env:BOOL":
					return NewRetrieved(true)
				}
				return nil, errors.New("impossible")
			})

			testProvider := newFakeProvider("test", func(_ context.Context, uri string, _ WatcherFunc) (*Retrieved, error) {
				return NewRetrieved(uri[5:])
			})

			resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, envProvider, testProvider), Converters: nil})
			require.NoError(t, err)

			cfgMap, err := resolver.Resolve(context.Background())
			require.NoError(t, err)
			assert.Equal(t, map[string]any{tt.name: tt.output}, cfgMap.ToStringMap())
		})
	}
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

			resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
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

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.ErrorIs(t, err, errTooManyRecursiveExpansions)
}

func TestResolverExpandURILimit(t *testing.T) {
	input := "${env:HOST}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}${env:PORT}"
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": input})
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.ErrorIs(t, err, errURILimit)
}

func TestResolverExpandInvalidScheme(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": "${g_c_s:VALUE}"})
	})

	testProvider := newFakeProvider("g_c_s", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		panic("must not be called")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())

	assert.EqualError(t, err, `invalid uri: "g_c_s:VALUE"`)
}

func TestResolverExpandInvalidOpaqueValue(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": []any{map[string]any{"test": "${test:$VALUE}"}}})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		panic("must not be called")
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
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

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `scheme "unsupported" is not supported for uri "unsupported:VALUE"`)
}

func TestResolverExpandStringValueInvalidReturnValue(t *testing.T) {
	provider := newFakeProvider("input", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved(map[string]any{"test": "localhost:${test:PORT}"})
	})

	testProvider := newFakeProvider("test", func(context.Context, string, WatcherFunc) (*Retrieved, error) {
		return NewRetrieved([]any{1243})
	})

	resolver, err := NewResolver(ResolverSettings{URIs: []string{"input:"}, Providers: makeMapProvidersMap(provider, testProvider), Converters: nil})
	require.NoError(t, err)

	_, err = resolver.Resolve(context.Background())
	assert.EqualError(t, err, `expanding ${test:PORT}, expected string value type, got []interface {}`)
}
