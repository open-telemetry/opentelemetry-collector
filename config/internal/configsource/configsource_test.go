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

package configsource

import (
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	publiccomponent "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/internal/configsource/component"
)

func TestApplyConfigSources(t *testing.T) {
	tests := []struct {
		name            string
		cfgSrcErrorCode cfgSrcErrorCode
		applyParams     ApplyConfigSourcesParams
	}{
		{
			name: "nil_param",
		},
		{
			name: "simple",
		},
		{
			name:            "nested",
			cfgSrcErrorCode: errNestedCfgSrc,
		},
		{
			name:            "err_begin_session",
			cfgSrcErrorCode: errCfgSrcBeginSession,
		},
		{
			name:            "recursion",
			cfgSrcErrorCode: errCfgSrcChainTooLong,
			applyParams: ApplyConfigSourcesParams{
				MaxRecursionDepth: 2,
			},
		},
		{
			name:            "multiple",
			cfgSrcErrorCode: errCfgSrcChainTooLong,
		},
		{
			name: "multiple",
			applyParams: ApplyConfigSourcesParams{
				MaxRecursionDepth: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := config.NewViper()
			v.SetConfigFile(path.Join("testdata", tt.name+".yaml"))
			require.NoError(t, v.ReadInConfig())

			tt.applyParams.ConfigSources = loadCfgSrcs(t, path.Join("testdata", tt.name+"_sources.yaml"))
			dst, err := ApplyConfigSources(context.Background(), v, tt.applyParams)

			if tt.cfgSrcErrorCode != cfgSrcErrorCode(0) {
				assert.Nil(t, dst)
				assert.Equal(t, tt.cfgSrcErrorCode, err.(*cfgSrcError).code)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, dst)

			expectedViper := config.NewViper()
			expectedViper.SetConfigFile(path.Join("testdata", tt.name+"_expected.yaml"))
			require.NoError(t, expectedViper.ReadInConfig())
			assert.Equal(t, expectedViper.AllSettings(), dst.AllSettings())
		})
	}
}

func Test_applyConfigSources_no_error(t *testing.T) {
	cfgSources := map[string]component.ConfigSource{
		"cfgsrc": &cfgSrcMock{
			Keys: map[string]interface{}{
				"int":  1,
				"str":  "str",
				"bool": true,
				"interface": map[string]interface{}{
					"i0":   0,
					"str1": "1",
				},
				"recurse": "$cfgsrc",
			},
		},
	}

	tests := []struct {
		name   string
		srcCfg map[string]interface{}
		dstCfg map[string]interface{}
		done   bool
	}{
		{
			name:   "done",
			srcCfg: map[string]interface{}{"a": 0},
			dstCfg: map[string]interface{}{"a": 0},
			done:   true,
		},
		{
			name: "root",
			srcCfg: map[string]interface{}{
				cfgSrcKey("cfgsrc"): map[string]interface{}{
					"key": "interface",
				},
			},
			dstCfg: map[string]interface{}{
				"i0":   0,
				"str1": "1",
			},
		},
		{
			name: "basic",
			srcCfg: map[string]interface{}{
				"a": map[string]interface{}{
					"a": "b",
					"c": map[string]interface{}{cfgSrcKey("cfgsrc"): nil},
				},
			},
			dstCfg: map[string]interface{}{
				"a": map[string]interface{}{
					"a": "b",
					"c": "nil_param",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcV := config.NewViper()
			require.NoError(t, srcV.MergeConfigMap(tt.srcCfg))

			dstV := config.NewViper()
			done, err := applyConfigSources(context.Background(), srcV, dstV, cfgSources)
			assert.NoError(t, err)
			assert.Equal(t, tt.done, done)
			assert.Equal(t, tt.dstCfg, dstV.AllSettings())
		})
	}
}

func Test_applyConfigSources_error(t *testing.T) {
	cfgSources := map[string]component.ConfigSource{
		"cfgsrc": &cfgSrcMock{
			Keys: map[string]interface{}{
				"bool": true,
			},
		},
	}

	tests := []struct {
		name    string
		srcCfg  map[string]interface{}
		errCode cfgSrcErrorCode
	}{
		{
			name: "err_missing_cfgsrc",
			srcCfg: map[string]interface{}{
				cfgSrcKey("cfgsrc/1"): map[string]interface{}{
					"key": "interface",
				},
			},
			errCode: errCfgSrcNotFound,
		},
		{
			name: "err_cfgsrc_apply",
			srcCfg: map[string]interface{}{
				"root": map[string]interface{}{
					cfgSrcKey("cfgsrc"): map[string]interface{}{
						"missing_key_param": "",
					},
				},
			},
			errCode: errCfgSrcApply,
		},
		{
			name: "err_only_map_at_root",
			srcCfg: map[string]interface{}{
				cfgSrcKey("cfgsrc"): map[string]interface{}{
					"key": "bool",
				},
			},
			errCode: errOnlyMapAtRootLevel,
		},
		{
			name: "nested_cfgsrc",
			srcCfg: map[string]interface{}{
				"a": map[string]interface{}{
					cfgSrcKey("cfgsrc"): map[string]interface{}{
						"b": map[string]interface{}{
							cfgSrcKey("cfgsrc"): map[string]interface{}{
								"key": "bool",
								"int": 1,
								"map": map[string]interface{}{
									"str": "force_already_visied_cfgsrc",
								},
							},
						},
					},
				},
			},
			errCode: errNestedCfgSrc,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcV := config.NewViper()
			require.NoError(t, srcV.MergeConfigMap(tt.srcCfg))

			done, err := applyConfigSources(context.Background(), srcV, config.NewViper(), cfgSources)
			assert.False(t, done)
			require.Error(t, err)
			assert.Equal(t, tt.errCode, err.(*cfgSrcError).code)
		})
	}
}

func Test_deepestConfigSourcesFirst(t *testing.T) {
	tests := []struct {
		name     string
		keys     []string
		expected bool
	}{
		{
			name: "left_cfgsrc",
			keys: []string{
				concatKeys("a", "b", "c", cfgSrcKey("d"), "e"),
				concatKeys("a", "b", "c", "f", "g", "h"),
			},
			expected: true,
		},
		{
			name: "right_cfgsrc",
			keys: []string{
				concatKeys("a", "b", "c", "f", "g", "h"),
				concatKeys("a", "b", "c", cfgSrcKey("d"), "e"),
			},
			expected: false,
		},
		{
			name: "left_cfgsrc_deepest",
			keys: []string{
				concatKeys("a", "b", "c", "d", cfgSrcKey("e")),
				concatKeys("a", "b", "c", cfgSrcKey("f"), "g"),
			},
			expected: true,
		},
		{
			name: "right_cfgsrc_deepest",
			keys: []string{
				concatKeys("a", "b", "c", cfgSrcKey("f"), "g"),
				concatKeys("a", "b", "c", "d", cfgSrcKey("e")),
			},
			expected: false,
		},
		{
			name: "left_nested_cfgsrc",
			keys: []string{
				concatKeys("a", cfgSrcKey("b"), "c", "d", cfgSrcKey("e")),
				concatKeys("a", cfgSrcKey("b"), "c", "f", "g"),
			},
			expected: true,
		},
		{
			name: "right_nested_cfgsrc",
			keys: []string{
				concatKeys("a", cfgSrcKey("b"), "c", "f", "g"),
				concatKeys("a", cfgSrcKey("b"), "c", "d", cfgSrcKey("e")),
			},
			expected: false,
		},
		{
			name: "left_deepest_not_by_length",
			keys: []string{
				concatKeys("a", strings.Repeat("a", 100), cfgSrcKey("b")),
				concatKeys("a", "a", "a", cfgSrcKey("b")),
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lessFn := deepestConfigSourcesFirst(tt.keys)
			assert.Equal(t, tt.expected, lessFn(0, 1))
		})
	}
}

func Test_extractCfgSrcInvocation(t *testing.T) {
	tests := []struct {
		name      string
		dstKey    string
		cfgSrc    string
		paramsKey string
	}{
		{
			name: concatKeys("a"),
		},
		{
			name: concatKeys("a", "b"),
		},
		{
			name:      concatKeys("simple", cfgSrcKey("cfgsrc")),
			dstKey:    "simple",
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("simple", cfgSrcKey("cfgsrc")),
		},
		{
			name:      concatKeys("simple_a", cfgSrcKey("cfgsrc"), "a"),
			dstKey:    "simple_a",
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("simple_a", cfgSrcKey("cfgsrc")),
		},
		{
			name:      concatKeys("nested", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
			dstKey:    concatKeys("nested", cfgSrcKey("top"), "a"),
			cfgSrc:    "deepest",
			paramsKey: concatKeys("nested", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
		},
		{
			name:      concatKeys("nested_a", cfgSrcKey("top"), "a", cfgSrcKey("deepest"), "a"),
			dstKey:    concatKeys("nested_a", cfgSrcKey("top"), "a"),
			cfgSrc:    "deepest",
			paramsKey: concatKeys("nested_a", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
		},
		{
			name:      concatKeys(cfgSrcKey("cfgSrcTopLvl")),
			cfgSrc:    "cfgSrcTopLvl",
			paramsKey: cfgSrcKey("cfgSrcTopLvl"),
		},
		{
			name:      concatKeys(cfgSrcKey("cfgSrcTopLvl_a"), "a"),
			cfgSrc:    "cfgSrcTopLvl_a",
			paramsKey: cfgSrcKey("cfgSrcTopLvl_a"),
		},
		{
			name:      concatKeys("typical", "a", "b", cfgSrcKey("cfgsrc")),
			dstKey:    concatKeys("typical", "a", "b"),
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("typical", "a", "b", cfgSrcKey("cfgsrc")),
		},
		{
			name:      concatKeys("typical_a", "a", "b", cfgSrcKey("cfgsrc"), "a", "b"),
			dstKey:    concatKeys("typical_a", "a", "b"),
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("typical_a", "a", "b", cfgSrcKey("cfgsrc")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstKey, cfgSrc, paramsKey := extractCfgSrcInvocation(tt.name)

			assert.Equal(t, tt.dstKey, dstKey)
			assert.Equal(t, tt.cfgSrc, cfgSrc)
			assert.Equal(t, tt.paramsKey, paramsKey)
		})
	}
}

func concatKeys(keys ...string) (s string) {
	s = keys[0]
	for _, key := range keys[1:] {
		s += config.ViperDelimiter + key
	}
	return s
}

func cfgSrcKey(key string) string {
	return ConfigSourcePrefix + key
}

type cfgSrcMock struct {
	ErrOnBeginSession bool                   `mapstructure:"err_begin_session"`
	Keys              map[string]interface{} `mapstructure:"keys"`
}

func (c cfgSrcMock) Start(context.Context, publiccomponent.Host) error {
	return nil
}

func (c cfgSrcMock) Shutdown(context.Context) error {
	return nil
}

func (c cfgSrcMock) BeginSession(context.Context) error {
	if c.ErrOnBeginSession {
		return errors.New("request to error on BeginSession")
	}
	return nil
}

func (c cfgSrcMock) Apply(_ context.Context, params interface{}) (interface{}, error) {
	if params == nil {
		return "nil_param", nil
	}
	switch v := params.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		keyVal, ok := v["key"]
		if !ok {
			return nil, errors.New("missing 'key' parameter")
		}
		return c.Keys[keyVal.(string)], nil
	}
	return nil, nil
}

func (c cfgSrcMock) EndSession(context.Context) {
}

// TODO: Create proper factories and methods to load generic ConfigSources, temporary test helper.
func loadCfgSrcs(t *testing.T, file string) map[string]component.ConfigSource {
	v := config.NewViper()
	v.SetConfigFile(file)
	require.NoError(t, v.ReadInConfig())

	cfgSrcMap := make(map[string]component.ConfigSource)
	for cfgSrcName := range v.AllSettings() {
		cfgSrc := &cfgSrcMock{}
		require.NoError(t, v.Sub(cfgSrcName).UnmarshalExact(cfgSrc))
		cfgSrcMap[cfgSrcName] = cfgSrc
	}

	return cfgSrcMap
}
