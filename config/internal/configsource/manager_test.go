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
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

func TestConfigSourceManager_Simple(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(nil)
	require.NoError(t, err)
	manager.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {Value: "test_value"},
			},
		},
	}

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

	cp := configparser.NewParserFromStringMap(originalCfg)

	actualParser, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedCfg, actualParser.ToStringMap())

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

func TestConfigSourceManager_ResolveErrors(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	tests := []struct {
		name            string
		config          map[string]interface{}
		configSourceMap map[string]configsource.ConfigSource
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
		{
			name: "error_on_retrieve_end",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector",
			},
			configSourceMap: map[string]configsource.ConfigSource{
				"tstcfgsrc": &testConfigSource{
					ErrOnRetrieveEnd: testErr,
					ValueMap: map[string]valueEntry{
						"selector": {Value: "test_value"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewManager(nil)
			require.NoError(t, err)
			manager.configSources = tt.configSourceMap

			res, err := manager.Resolve(ctx, configparser.NewParserFromStringMap(tt.config))
			require.Error(t, err)
			require.Nil(t, res)
			require.NoError(t, manager.Close(ctx))
		})
	}
}

func TestConfigSourceManager_ArraysAndMaps(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(nil)
	require.NoError(t, err)
	manager.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"elem0": {Value: "elem0_value"},
				"elem1": {Value: "elem1_value"},
				"k0":    {Value: "k0_value"},
				"k1":    {Value: "k1_value"},
			},
		},
	}

	file := path.Join("testdata", "arrays_and_maps.yaml")
	cp, err := configparser.NewParserFromFile(file)
	require.NoError(t, err)

	expectedFile := path.Join("testdata", "arrays_and_maps_expected.yaml")
	expectedParser, err := configparser.NewParserFromFile(expectedFile)
	require.NoError(t, err)

	actualParser, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedParser.ToStringMap(), actualParser.ToStringMap())
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
	tstCfgSrc.OnRetrieve = func(ctx context.Context, selector string, params interface{}) error {
		assert.Equal(t, tstCfgSrc.ValueMap[selector].Value, params)
		return nil
	}

	manager, err := NewManager(nil)
	require.NoError(t, err)
	manager.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &tstCfgSrc,
	}

	file := path.Join("testdata", "params_handling.yaml")
	cp, err := configparser.NewParserFromFile(file)
	require.NoError(t, err)

	expectedFile := path.Join("testdata", "params_handling_expected.yaml")
	expectedParser, err := configparser.NewParserFromFile(expectedFile)
	require.NoError(t, err)

	actualParser, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedParser.ToStringMap(), actualParser.ToStringMap())
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_WatchForUpdate(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(nil)
	require.NoError(t, err)

	watchForUpdateCh := make(chan error, 1)
	manager.configSources = map[string]configsource.ConfigSource{
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
	}

	originalCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"var0": "$tstcfgsrc:test_selector",
		},
	}

	cp := configparser.NewParserFromStringMap(originalCfg)
	_, err = manager.Resolve(ctx, cp)
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
	manager, err := NewManager(nil)
	require.NoError(t, err)

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

	manager.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {
					Value:            "test_value",
					WatchForUpdateFn: watchForUpdateFn,
				},
			},
		},
	}

	originalCfg := map[string]interface{}{
		"top0": map[string]interface{}{
			"var0": "$tstcfgsrc:test_selector",
			"var1": "$tstcfgsrc:test_selector",
			"var2": "$tstcfgsrc:test_selector",
			"var3": "$tstcfgsrc:test_selector",
		},
	}

	cp := configparser.NewParserFromStringMap(originalCfg)
	_, err = manager.Resolve(ctx, cp)
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
	require.NoError(t, os.Setenv("envvar", "envvar_value"))
	defer func() {
		assert.NoError(t, os.Unsetenv("envvar"))
	}()

	ctx := context.Background()
	tstCfgSrc := testConfigSource{
		ValueMap: map[string]valueEntry{
			"int_key": {Value: 42},
		},
	}

	// Intercept "params_key" and create an entry with the params themselves.
	tstCfgSrc.OnRetrieve = func(ctx context.Context, selector string, params interface{}) error {
		if selector == "params_key" {
			tstCfgSrc.ValueMap[selector] = valueEntry{Value: params}
		}
		return nil
	}

	manager, err := NewManager(nil)
	require.NoError(t, err)
	manager.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &tstCfgSrc,
	}

	file := path.Join("testdata", "envvar_cfgsrc_mix.yaml")
	cp, err := configparser.NewParserFromFile(file)
	require.NoError(t, err)

	expectedFile := path.Join("testdata", "envvar_cfgsrc_mix_expected.yaml")
	expectedParser, err := configparser.NewParserFromFile(expectedFile)
	require.NoError(t, err)

	actualParser, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	assert.Equal(t, expectedParser.ToStringMap(), actualParser.ToStringMap())
	assert.NoError(t, manager.Close(ctx))
}

func TestManager_expandString(t *testing.T) {
	ctx := context.Background()
	csp, err := NewManager(nil)
	require.NoError(t, err)
	csp.configSources = map[string]configsource.ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"str_key": {Value: "test_value"},
				"int_key": {Value: 1},
			},
		},
	}

	require.NoError(t, os.Setenv("envvar", "envvar_value"))
	defer func() {
		assert.NoError(t, os.Unsetenv("envvar"))
	}()
	require.NoError(t, os.Setenv("envvar_str_key", "str_key"))
	defer func() {
		assert.NoError(t, os.Unsetenv("envvar_str_key"))
	}()

	tests := []struct {
		name    string
		input   string
		want    interface{}
		wantErr error
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := csp.expandString(ctx, tt.input)
			require.IsType(t, tt.wantErr, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_parseCfgSrc(t *testing.T) {
	tests := []struct {
		name       string
		str        string
		cfgSrcName string
		selector   string
		params     interface{}
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
			cfgSrcName, selector, params, err := parseCfgSrc(tt.str)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.cfgSrcName, cfgSrcName)
			assert.Equal(t, tt.selector, selector)
			assert.Equal(t, tt.params, params)
		})
	}
}

// testConfigSource a ConfigSource to be used in tests.
type testConfigSource struct {
	ValueMap map[string]valueEntry

	ErrOnRetrieve    error
	ErrOnRetrieveEnd error
	ErrOnClose       error

	OnRetrieve func(ctx context.Context, selector string, params interface{}) error
}

type valueEntry struct {
	Value            interface{}
	WatchForUpdateFn func() error
}

var _ configsource.ConfigSource = (*testConfigSource)(nil)

func (t *testConfigSource) Retrieve(ctx context.Context, selector string, params interface{}) (configsource.Retrieved, error) {
	if t.OnRetrieve != nil {
		if err := t.OnRetrieve(ctx, selector, params); err != nil {
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

func (t *testConfigSource) RetrieveEnd(context.Context) error {
	return t.ErrOnRetrieveEnd
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
