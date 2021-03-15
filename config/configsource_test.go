package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
)

func Test_applyConfigSources_no_error(t *testing.T) {
	cfgSources := map[string]component.ConfigSource{
		"cfgsrc": &cfgSrcMock{
			keys: map[string]interface{}{
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
		{
			name: "nested",
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
			dstCfg: map[string]interface{}{
				"a": map[string]interface{}{
					cfgSrcKey("cfgsrc"): map[string]interface{}{
						"b": true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcV := NewViper()
			require.NoError(t, srcV.MergeConfigMap(tt.srcCfg))

			dstV := NewViper()
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
			keys: map[string]interface{}{
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
		{
			name: "nested",
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
			dstCfg: map[string]interface{}{
				"a": map[string]interface{}{
					cfgSrcKey("cfgsrc"): map[string]interface{}{
						"b": true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srcV := NewViper()
			require.NoError(t, srcV.MergeConfigMap(tt.srcCfg))

			done, err := applyConfigSources(context.Background(), srcV, NewViper(), cfgSources)
			assert.NoError(t, err)
			assert.False(t, done)
			// TODO: Check error
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
		key       string
		dstKey    string
		cfgSrc    string
		paramsKey string
	}{
		{
			key: concatKeys("a"),
		},
		{
			key: concatKeys("a", "b"),
		},
		{
			key:       concatKeys("simple", cfgSrcKey("cfgsrc")),
			dstKey:    "simple",
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("simple", cfgSrcKey("cfgsrc")),
		},
		{
			key:       concatKeys("simple_a", cfgSrcKey("cfgsrc"), "a"),
			dstKey:    "simple_a",
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("simple_a", cfgSrcKey("cfgsrc")),
		},
		{
			key:       concatKeys("nested", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
			dstKey:    concatKeys("nested", cfgSrcKey("top"), "a"),
			cfgSrc:    "deepest",
			paramsKey: concatKeys("nested", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
		},
		{
			key:       concatKeys("nested_a", cfgSrcKey("top"), "a", cfgSrcKey("deepest"), "a"),
			dstKey:    concatKeys("nested_a", cfgSrcKey("top"), "a"),
			cfgSrc:    "deepest",
			paramsKey: concatKeys("nested_a", cfgSrcKey("top"), "a", cfgSrcKey("deepest")),
		},
		{
			key:       concatKeys(cfgSrcKey("cfgSrcTopLvl")),
			cfgSrc:    "cfgSrcTopLvl",
			paramsKey: cfgSrcKey("cfgSrcTopLvl"),
		},
		{
			key:       concatKeys(cfgSrcKey("cfgSrcTopLvl_a"), "a"),
			cfgSrc:    "cfgSrcTopLvl_a",
			paramsKey: cfgSrcKey("cfgSrcTopLvl_a"),
		},
		{
			key:       concatKeys("typical", "a", "b", cfgSrcKey("cfgsrc")),
			dstKey:    concatKeys("typical", "a", "b"),
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("typical", "a", "b", cfgSrcKey("cfgsrc")),
		},
		{
			key:       concatKeys("typical_a", "a", "b", cfgSrcKey("cfgsrc"), "a", "b"),
			dstKey:    concatKeys("typical_a", "a", "b"),
			cfgSrc:    "cfgsrc",
			paramsKey: concatKeys("typical_a", "a", "b", cfgSrcKey("cfgsrc")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			dstKey, cfgSrc, paramsKey := extractCfgSrcInvocation(tt.key)

			assert.Equal(t, tt.dstKey, dstKey)
			assert.Equal(t, tt.cfgSrc, cfgSrc)
			assert.Equal(t, tt.paramsKey, paramsKey)
		})
	}
}

func concatKeys(keys ...string) (s string) {
	s = keys[0]
	for _, key := range keys[1:] {
		s += ViperDelimiter + key
	}
	return s
}

func cfgSrcKey(key string) string {
	return ConfigSourcePrefix + key
}

type cfgSrcMock struct {
	keys map[string]interface{}
}

func (c cfgSrcMock) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (c cfgSrcMock) Shutdown(_ context.Context) error {
	return nil
}

func (c cfgSrcMock) BeginSession(_ context.Context, _ component.SessionParams) error {
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
		return c.keys[v["key"].(string)], nil
	}
	return nil, nil
}

func (c cfgSrcMock) EndSession(_ context.Context, _ component.SessionParams) {
}
