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
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
)

func TestConfigSourceManager_Simple(t *testing.T) {
	ctx := context.Background()
	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {Value: "test_value"},
			},
		},
	})

	originalCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"int":    1,
			"cfgsrc": "$tstcfgsrc:test_selector",
		},
	}
	expectedCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"int":    1,
			"cfgsrc": "test_value",
		},
	}

	cp := confmap.NewFromStringMap(originalCfg)

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedCfg, res.ToStringMap())

	doneCh := make(chan struct{})
	var errWatcher error
	go func() {
		defer close(doneCh)
		errWatcher = manager.WatchForUpdate()
	}()

	manager.WaitForWatcher()
	assert.NoError(t, manager.Close(ctx))
	<-doneCh
	assert.ErrorIs(t, errWatcher, configsource.ErrSessionClosed)
}

func TestConfigSourceManager_ResolveRemoveConfigSourceSection(t *testing.T) {
	cfg := map[string]interface{}{
		"config_sources": map[string]interface{}{
			"testcfgsrc": nil,
		},
		"another_section": map[string]interface{}{
			"int": 42,
		},
	}

	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{},
	})

	res, err := manager.Resolve(context.Background(), confmap.NewFromStringMap(cfg))
	require.NoError(t, err)
	require.NotNil(t, res)

	delete(cfg, "config_sources")
	assert.Equal(t, cfg, res.ToStringMap())
}

func TestConfigSourceManager_ResolveErrors(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	tests := []struct {
		config          map[string]interface{}
		configSourceMap map[string]configsource.ConfigSource
		name            string
	}{
		{
			name: "incorrect_cfgsrc_ref",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector?{invalid}",
			},
			configSourceMap: map[string]configsource.ConfigSource{
				"tstcfgsrc": &testConfigSource{},
			},
		},
		{
			name: "error_on_retrieve",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector",
			},
			configSourceMap: map[string]configsource.ConfigSource{
				"tstcfgsrc": &testConfigSource{ErrOnRetrieve: testErr},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := newManager(tt.configSourceMap)

			res, err := manager.Resolve(ctx, confmap.NewFromStringMap(tt.config))
			require.Error(t, err)
			require.Nil(t, res)
			require.NoError(t, manager.Close(ctx))
		})
	}
}

func TestConfigSourceManager_YAMLInjection(t *testing.T) {
	ctx := context.Background()
	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"valid_yaml_str": {Value: `
bool: true
int: 42
source: string
map:
  k0: v0
  k1: v1
`},
				"invalid_yaml_str": {Value: ":"},
				"valid_yaml_byte_slice": {Value: []byte(`
bool: true
int: 42
source: "[]byte"
map:
  k0: v0
  k1: v1
`)},
				"invalid_yaml_byte_slice": {Value: []byte(":")},
			},
		},
	})

	file := filepath.Join("testdata", "yaml_injection.yaml")
	cp, err := confmaptest.LoadConf(file)
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "yaml_injection_expected.yaml")
	expectedConfigMap, err := confmaptest.LoadConf(expectedFile)
	require.NoError(t, err)
	expectedCfg := expectedConfigMap.ToStringMap()

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	actualCfg := res.ToStringMap()
	assert.Equal(t, expectedCfg, actualCfg)
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_ArraysAndMaps(t *testing.T) {
	ctx := context.Background()
	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"elem0": {Value: "elem0_value"},
				"elem1": {Value: "elem1_value"},
				"k0":    {Value: "k0_value"},
				"k1":    {Value: "k1_value"},
			},
		},
	})

	file := filepath.Join("testdata", "arrays_and_maps.yaml")
	cp, err := confmaptest.LoadConf(file)
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "arrays_and_maps_expected.yaml")
	expectedConfigMap, err := confmaptest.LoadConf(expectedFile)
	require.NoError(t, err)

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedConfigMap.ToStringMap(), res.ToStringMap())
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_ParamsHandling(t *testing.T) {
	ctx := context.Background()
	tstCfgSrc := testConfigSource{
		ValueMap: map[string]valueEntry{
			"elem0": {Value: nil},
			"elem1": {
				Value: map[string]interface{}{
					"p0": true,
					"p1": "a string with spaces",
					"p3": 42,
				},
			},
			"k0": {Value: nil},
			"k1": {
				Value: map[string]interface{}{
					"p0": true,
					"p1": "a string with spaces",
					"p2": map[string]interface{}{
						"p2_0": "a nested map0",
						"p2_1": true,
					},
				},
			},
		},
	}

	// Set OnRetrieve to check if the parameters were parsed as expected.
	tstCfgSrc.OnRetrieve = func(ctx context.Context, selector string, paramsConfigMap *confmap.Conf) error {
		paramsValue := (interface{})(nil)
		if paramsConfigMap != nil {
			paramsValue = paramsConfigMap.ToStringMap()
		}
		assert.Equal(t, tstCfgSrc.ValueMap[selector].Value, paramsValue)
		return nil
	}

	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &tstCfgSrc,
	})

	file := filepath.Join("testdata", "params_handling.yaml")
	cp, err := confmaptest.LoadConf(file)
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "params_handling_expected.yaml")
	expectedConfigMap, err := confmaptest.LoadConf(expectedFile)
	require.NoError(t, err)

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedConfigMap.ToStringMap(), res.ToStringMap())
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_WatchForUpdate(t *testing.T) {
	ctx := context.Background()
	watchForUpdateCh := make(chan error, 1)

	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {
					Value: "test_value",
					WatchForUpdateFn: func() error {
						return <-watchForUpdateCh
					},
				},
			},
		},
	})

	originalCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"var0": "$tstcfgsrc:test_selector",
		},
	}

	cp := confmap.NewFromStringMap(originalCfg)
	_, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)

	doneCh := make(chan struct{})
	var errWatcher error
	go func() {
		defer close(doneCh)
		errWatcher = manager.WatchForUpdate()
	}()

	manager.WaitForWatcher()
	watchForUpdateCh <- configsource.ErrValueUpdated

	<-doneCh
	assert.ErrorIs(t, errWatcher, configsource.ErrValueUpdated)
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_MultipleWatchForUpdate(t *testing.T) {
	ctx := context.Background()

	watchDoneCh := make(chan struct{})
	const watchForUpdateChSize int = 2
	watchForUpdateCh := make(chan error, watchForUpdateChSize)
	watchForUpdateFn := func() error {
		select {
		case errFromWatchForUpdate := <-watchForUpdateCh:
			return errFromWatchForUpdate
		case <-watchDoneCh:
			return configsource.ErrSessionClosed
		}
	}

	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {
					Value:            "test_value",
					WatchForUpdateFn: watchForUpdateFn,
				},
			},
		},
	})

	originalCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"var0": "$tstcfgsrc:test_selector",
			"var1": "$tstcfgsrc:test_selector",
			"var2": "$tstcfgsrc:test_selector",
			"var3": "$tstcfgsrc:test_selector",
		},
	}

	cp := confmap.NewFromStringMap(originalCfg)
	_, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)

	doneCh := make(chan struct{})
	var errWatcher error
	go func() {
		defer close(doneCh)
		errWatcher = manager.WatchForUpdate()
	}()

	manager.WaitForWatcher()

	for i := 0; i < watchForUpdateChSize; i++ {
		watchForUpdateCh <- configsource.ErrValueUpdated
	}

	<-doneCh
	assert.ErrorIs(t, errWatcher, configsource.ErrValueUpdated)
	close(watchForUpdateCh)
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_EnvVarHandling(t *testing.T) {
	t.Setenv("envvar", "envvar_value")

	ctx := context.Background()
	tstCfgSrc := testConfigSource{
		ValueMap: map[string]valueEntry{
			"int_key": {Value: 42},
		},
	}

	// Intercept "params_key" and create an entry with the params themselves.
	tstCfgSrc.OnRetrieve = func(ctx context.Context, selector string, paramsConfigMap *confmap.Conf) error {
		if selector == "params_key" {
			tstCfgSrc.ValueMap[selector] = valueEntry{Value: paramsConfigMap.ToStringMap()}
		}
		return nil
	}

	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &tstCfgSrc,
	})

	file := filepath.Join("testdata", "envvar_cfgsrc_mix.yaml")
	cp, err := confmaptest.LoadConf(file)
	require.NoError(t, err)

	expectedFile := filepath.Join("testdata", "envvar_cfgsrc_mix_expected.yaml")
	expectedConfigMap, err := confmaptest.LoadConf(expectedFile)
	require.NoError(t, err)

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedConfigMap.ToStringMap(), res.ToStringMap())
	assert.NoError(t, manager.Close(ctx))
}

func TestManager_parseStringValue(t *testing.T) {
	ctx := context.Background()
	manager := newManager(map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"str_key": {Value: "test_value"},
				"int_key": {Value: 1},
				"nil_key": {Value: nil},
			},
		},
		"tstcfgsrc/named": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"int_key": {Value: 42},
			},
		},
	})

	t.Setenv("envvar", "envvar_value")
	t.Setenv("envvar_str_key", "str_key")

	tests := []struct {
		want    interface{}
		wantErr error
		name    string
		input   string
	}{
		{
			name:  "literal_string",
			input: "literal_string",
			want:  "literal_string",
		},
		{
			name:  "escaped_$",
			input: "$$tstcfgsrc:int_key$$envvar",
			want:  "$tstcfgsrc:int_key$envvar",
		},
		{
			name:  "cfgsrc_int",
			input: "$tstcfgsrc:int_key",
			want:  1,
		},
		{
			name:  "concatenate_cfgsrc_string",
			input: "prefix-$tstcfgsrc:str_key",
			want:  "prefix-test_value",
		},
		{
			name:  "concatenate_cfgsrc_non_string",
			input: "prefix-$tstcfgsrc:int_key",
			want:  "prefix-1",
		},
		{
			name:  "envvar",
			input: "$envvar",
			want:  "envvar_value",
		},
		{
			name:  "prefixed_envvar",
			input: "prefix-$envvar",
			want:  "prefix-envvar_value",
		},
		{
			name:    "envvar_treated_as_cfgsrc",
			input:   "$envvar:suffix",
			wantErr: &errUnknownConfigSource{},
		},
		{
			name:  "cfgsrc_using_envvar",
			input: "$tstcfgsrc:$envvar_str_key",
			want:  "test_value",
		},
		{
			name:  "envvar_cfgsrc_using_envvar",
			input: "$envvar/$tstcfgsrc:$envvar_str_key",
			want:  "envvar_value/test_value",
		},
		{
			name:  "delimited_cfgsrc",
			input: "${tstcfgsrc:int_key}",
			want:  1,
		},
		{
			name:    "unknown_delimited_cfgsrc",
			input:   "${cfgsrc:int_key}",
			wantErr: &errUnknownConfigSource{},
		},
		{
			name:  "delimited_cfgsrc_with_spaces",
			input: "${ tstcfgsrc: int_key }",
			want:  1,
		},
		{
			name:  "interpolated_and_delimited_cfgsrc",
			input: "0/${ tstcfgsrc: $envvar_str_key }/2/${tstcfgsrc:int_key}",
			want:  "0/test_value/2/1",
		},
		{
			name:  "named_config_src",
			input: "$tstcfgsrc/named:int_key",
			want:  42,
		},
		{
			name:  "named_config_src_bracketed",
			input: "${tstcfgsrc/named:int_key}",
			want:  42,
		},
		{
			name:  "envvar_name_separator",
			input: "$envvar/test/test",
			want:  "envvar_value/test/test",
		},
		{
			name:    "envvar_treated_as_cfgsrc",
			input:   "$envvar/test:test",
			wantErr: &errUnknownConfigSource{},
		},
		{
			name:  "retrieved_nil",
			input: "${tstcfgsrc:nil_key}",
		},
		{
			name:  "retrieved_nil_on_string",
			input: "prefix-${tstcfgsrc:nil_key}-suffix",
			want:  "prefix--suffix",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.parseStringValue(ctx, tt.input)
			require.IsType(t, tt.wantErr, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseCfgSrcInvocation(t *testing.T) {
	tests := []struct {
		params     interface{}
		name       string
		str        string
		cfgSrcName string
		selector   string
		wantErr    bool
	}{
		{
			name:       "basic",
			str:        "cfgsrc:selector",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
		},
		{
			name:    "missing_selector",
			str:     "cfgsrc",
			wantErr: true,
		},
		{
			name:       "params",
			str:        "cfgsrc:selector?p0=1&p1=a_string&p2=true",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
			params: map[string]interface{}{
				"p0": 1,
				"p1": "a_string",
				"p2": true,
			},
		},
		{
			name:       "query_pass_nil",
			str:        "cfgsrc:selector?p0&p1&p2",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
			params: map[string]interface{}{
				"p0": nil,
				"p1": nil,
				"p2": nil,
			},
		},
		{
			name:       "array_in_params",
			str:        "cfgsrc:selector?p0=0&p0=1&p0=2&p1=done",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
			params: map[string]interface{}{
				"p0": []interface{}{0, 1, 2},
				"p1": "done",
			},
		},
		{
			name:       "empty_param",
			str:        "cfgsrc:selector?no_closing=",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
			params: map[string]interface{}{
				"no_closing": interface{}(nil),
			},
		},
		{
			name:       "use_url_encode",
			str:        "cfgsrc:selector?p0=contains+%3D+and+%26+too",
			cfgSrcName: "cfgsrc",
			selector:   "selector",
			params: map[string]interface{}{
				"p0": "contains = and & too",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgSrcName, selector, paramsConfigMap, err := parseCfgSrcInvocation(tt.str)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.cfgSrcName, cfgSrcName)
			assert.Equal(t, tt.selector, selector)

			paramsValue := (interface{})(nil)
			if paramsConfigMap != nil {
				paramsValue = paramsConfigMap.ToStringMap()
			}
			assert.Equal(t, tt.params, paramsValue)
		})
	}
}

func newManager(configSources map[string]configsource.ConfigSource) *Manager {
	manager, _ := NewManager(nil)
	manager.configSources = configSources
	return manager
}

// testConfigSource a ConfigSource to be used in tests.
type testConfigSource struct {
	ValueMap map[string]valueEntry

	ErrOnRetrieve error
	ErrOnClose    error

	OnRetrieve func(ctx context.Context, selector string, paramsConfigMap *confmap.Conf) error
}

type valueEntry struct {
	Value            interface{}
	WatchForUpdateFn func() error
}

var _ configsource.ConfigSource = (*testConfigSource)(nil)

func (t *testConfigSource) Retrieve(ctx context.Context, selector string, paramsConfigMap *confmap.Conf) (configsource.Retrieved, error) {
	if t.OnRetrieve != nil {
		if err := t.OnRetrieve(ctx, selector, paramsConfigMap); err != nil {
			return nil, err
		}
	}

	if t.ErrOnRetrieve != nil {
		return nil, t.ErrOnRetrieve
	}

	entry, ok := t.ValueMap[selector]
	if !ok {
		return nil, fmt.Errorf("no value for selector %q", selector)
	}

	if entry.WatchForUpdateFn != nil {
		return &watchableRetrieved{
			retrieved: retrieved{
				value: entry.Value,
			},
			watchForUpdateFn: entry.WatchForUpdateFn,
		}, nil
	}

	return &retrieved{
		value: entry.Value,
	}, nil
}

func (t *testConfigSource) Close(context.Context) error {
	return t.ErrOnClose
}

type retrieved struct {
	value interface{}
}

var _ configsource.Retrieved = (*retrieved)(nil)

func (r *retrieved) Value() interface{} {
	return r.value
}

type watchableRetrieved struct {
	retrieved
	watchForUpdateFn func() error
}

func (r *watchableRetrieved) WatchForUpdate() error {
	return r.watchForUpdateFn()
}
