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
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config"
)

func TestConfigSourceManager_Simple(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(nil)
	require.NoError(t, err)
	manager.configSources = map[string]ConfigSource{
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

	cp := config.NewParserFromStringMap(originalCfg)

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	actualCfg := res.Viper().AllSettings()
	assert.Equal(t, expectedCfg, actualCfg)

	doneCh := make(chan struct{})
	var errWatcher error
	go func() {
		defer close(doneCh)
		errWatcher = manager.WatchForUpdate()
	}()

	manager.WaitForWatcher()
	assert.NoError(t, manager.Close(ctx))
	<-doneCh
	assert.ErrorIs(t, errWatcher, ErrSessionClosed)
}

func TestConfigSourceManager_ResolveErrors(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("test error")

	tests := []struct {
		name            string
		config          map[string]interface{}
		configSourceMap map[string]ConfigSource
	}{
		{
			name: "not_found_config_source",
			config: map[string]interface{}{
				"cfgsrc": "$unknown:test_selector",
			},
			configSourceMap: map[string]ConfigSource{
				"tstcfgsrc": &testConfigSource{},
			},
		},
		{
			name: "incorrect_cfgsrc_ref",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector?{invalid}",
			},
			configSourceMap: map[string]ConfigSource{
				"tstcfgsrc": &testConfigSource{},
			},
		},
		{
			name: "error_on_new_session",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector",
			},
			configSourceMap: map[string]ConfigSource{
				"tstcfgsrc": &testConfigSource{ErrOnNewSession: testErr},
			},
		},
		{
			name: "error_on_retrieve",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector",
			},
			configSourceMap: map[string]ConfigSource{
				"tstcfgsrc": &testConfigSource{ErrOnRetrieve: testErr},
			},
		},
		{
			name: "error_on_retrieve_end",
			config: map[string]interface{}{
				"cfgsrc": "$tstcfgsrc:selector",
			},
			configSourceMap: map[string]ConfigSource{
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

			res, err := manager.Resolve(ctx, config.NewParserFromStringMap(tt.config))
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
	manager.configSources = map[string]ConfigSource{
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
	cp, err := config.NewParserFromFile(file)
	require.NoError(t, err)

	expectedFile := path.Join("testdata", "arrays_and_maps_expected.yaml")
	expectedParser, err := config.NewParserFromFile(expectedFile)
	require.NoError(t, err)
	expectedCfg := expectedParser.Viper().AllSettings()

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	actualCfg := res.Viper().AllSettings()
	assert.Equal(t, expectedCfg, actualCfg)
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
	manager.configSources = map[string]ConfigSource{
		"tstcfgsrc": &tstCfgSrc,
	}

	file := path.Join("testdata", "params_handling.yaml")
	cp, err := config.NewParserFromFile(file)
	require.NoError(t, err)

	expectedFile := path.Join("testdata", "params_handling_expected.yaml")
	expectedParser, err := config.NewParserFromFile(expectedFile)
	require.NoError(t, err)
	expectedCfg := expectedParser.Viper().AllSettings()

	res, err := manager.Resolve(ctx, cp)
	require.NoError(t, err)
	actualCfg := res.Viper().AllSettings()
	assert.Equal(t, expectedCfg, actualCfg)
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_WatchForUpdate(t *testing.T) {
	ctx := context.Background()
	manager, err := NewManager(nil)
	require.NoError(t, err)

	watchForUpdateCh := make(chan error, 1)
	manager.configSources = map[string]ConfigSource{
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

	cp := config.NewParserFromStringMap(originalCfg)
	_, err = manager.Resolve(ctx, cp)
	require.NoError(t, err)

	doneCh := make(chan struct{})
	var errWatcher error
	go func() {
		defer close(doneCh)
		errWatcher = manager.WatchForUpdate()
	}()

	manager.WaitForWatcher()
	watchForUpdateCh <- ErrValueUpdated

	<-doneCh
	assert.ErrorIs(t, errWatcher, ErrValueUpdated)
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
			return ErrSessionClosed
		}
	}

	manager.configSources = map[string]ConfigSource{
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

	cp := config.NewParserFromStringMap(originalCfg)
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
		watchForUpdateCh <- ErrValueUpdated
	}

	<-doneCh
	assert.ErrorIs(t, errWatcher, ErrValueUpdated)
	close(watchForUpdateCh)
	assert.NoError(t, manager.Close(ctx))
}

func TestConfigSourceManager_expandConfigSources(t *testing.T) {
	ctx := context.Background()
	csp, err := NewManager(nil)
	require.NoError(t, err)
	csp.configSources = map[string]ConfigSource{
		"tstcfgsrc": &testConfigSource{
			ValueMap: map[string]valueEntry{
				"test_selector": {Value: "test_value"},
			},
		},
	}

	v, err := csp.expandConfigSources(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, "", v)

	v, err = csp.expandConfigSources(ctx, "no_cfgsrc")
	assert.NoError(t, err)
	assert.Equal(t, "no_cfgsrc", v)

	// Not found config source.
	v, err = csp.expandConfigSources(ctx, "$cfgsrc:selector")
	assert.Error(t, err)
	assert.Nil(t, v)

	v, err = csp.expandConfigSources(ctx, "$tstcfgsrc:test_selector")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", v)

	v, err = csp.expandConfigSources(ctx, "$tstcfgsrc:invalid_selector")
	assert.Error(t, err)
	assert.Nil(t, v)
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

	ErrOnNewSession  error
	ErrOnRetrieve    error
	ErrOnRetrieveEnd error
	ErrOnClose       error

	OnRetrieve func(ctx context.Context, selector string, params interface{}) error
}

type valueEntry struct {
	Value            interface{}
	WatchForUpdateFn func() error
}

var _ (ConfigSource) = (*testConfigSource)(nil)
var _ (Session) = (*testConfigSource)(nil)

func (t *testConfigSource) NewSession(context.Context) (Session, error) {
	if t.ErrOnNewSession != nil {
		return nil, t.ErrOnNewSession
	}
	return t, nil
}

func (t *testConfigSource) Retrieve(ctx context.Context, selector string, params interface{}) (Retrieved, error) {
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

	watchForUpdateFn := func() error {
		return ErrWatcherNotSupported
	}

	if entry.WatchForUpdateFn != nil {
		watchForUpdateFn = entry.WatchForUpdateFn
	}

	return &retrieved{
		value:            entry.Value,
		watchForUpdateFn: watchForUpdateFn,
	}, nil
}

func (t *testConfigSource) RetrieveEnd(context.Context) error {
	return t.ErrOnRetrieveEnd
}

func (t *testConfigSource) Close(context.Context) error {
	return t.ErrOnClose
}

type retrieved struct {
	value            interface{}
	watchForUpdateFn func() error
}

var _ (Retrieved) = (*retrieved)(nil)

func (r *retrieved) Value() interface{} {
	return r.value
}

func (r *retrieved) WatchForUpdate() error {
	return r.watchForUpdateFn()
}
