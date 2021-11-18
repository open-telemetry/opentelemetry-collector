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

package service

import (
	"context"
	"errors"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configmapprovider"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"go.opentelemetry.io/collector/config/experimental/configsource"
)

type errConfigMapProvider struct {
	ret *errRetrieved
	err error
}

func (ecmp *errConfigMapProvider) Retrieve(_ context.Context, onChange func(*configmapprovider.ChangeEvent)) (configmapprovider.Retrieved, error) {
	if ecmp.ret != nil {
		ecmp.ret.onChange = onChange
	}
	return ecmp.ret, ecmp.err
}

func (ecmp *errConfigMapProvider) Shutdown(context.Context) error {
	return nil
}

type errConfigUnmarshaler struct {
	err error
}

func (ecu *errConfigUnmarshaler) Unmarshal(*config.Map, component.Factories) (*config.Config, error) {
	return nil, ecu.err
}

type errRetrieved struct {
	retM     *config.Map
	errW     error
	errC     error
	onChange func(event *configmapprovider.ChangeEvent)
}

func (er *errRetrieved) Get(ctx context.Context) (*config.Map, error) {
	er.onChange(&configmapprovider.ChangeEvent{Error: er.errW})
	return er.retM, nil
}

func (er *errRetrieved) Close(context.Context) error {
	return er.errC
}

func TestConfigWatcher(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)

	tests := []struct {
		name              string
		parserProvider    configmapprovider.Provider
		configUnmarshaler configunmarshaler.ConfigUnmarshaler
		expectNewErr      bool
		expectWatchErr    bool
		expectCloseErr    bool
	}{
		{
			name:              "retrieve_err",
			parserProvider:    &errConfigMapProvider{err: errors.New("retrieve_err")},
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name:              "retrieve_ok_unmarshal_err",
			parserProvider:    configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")),
			configUnmarshaler: &errConfigUnmarshaler{err: errors.New("retrieve_ok_unmarshal_err")},
			expectNewErr:      true,
		},
		{
			name:              "validation_err",
			parserProvider:    configmapprovider.NewFile(path.Join("testdata", "otelcol-invalid.yaml")),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectNewErr:      true,
		},
		{
			name: "watch_err",
			parserProvider: func() configmapprovider.Provider {
				ret, err := configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")).Retrieve(context.Background(), onChange)
				require.NoError(t, err)
				m, err := ret.Get(context.Background())
				require.NoError(t, err)
				return &errConfigMapProvider{ret: &errRetrieved{retM: m, errW: errors.New("watch_err")}}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectWatchErr:    true,
		},
		{
			name: "close_err",
			parserProvider: func() configmapprovider.Provider {
				ret, err := configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")).Retrieve(context.Background(), onChange)
				require.NoError(t, err)
				m, err := ret.Get(context.Background())
				require.NoError(t, err)
				return &errConfigMapProvider{ret: &errRetrieved{retM: m, errC: errors.New("close_err")}}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
			expectCloseErr:    true,
		},
		{
			name: "ok",
			parserProvider: func() configmapprovider.Provider {
				// Use errRetrieved with nil errors to have Watchable interface implemented.
				ret, err := configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")).Retrieve(context.Background(), onChange)
				require.NoError(t, err)
				m, err := ret.Get(context.Background())
				require.NoError(t, err)
				return &errConfigMapProvider{ret: &errRetrieved{retM: m}}
			}(),
			configUnmarshaler: configunmarshaler.NewDefault(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := CollectorSettings{
				ConfigMapProvider: tt.parserProvider,
				ConfigUnmarshaler: tt.configUnmarshaler,
				Factories:         factories,
			}

			cfgW, _ := newConfigWatcher(context.Background(), set)
			_, errN := cfgW.get()

			if tt.expectNewErr {
				assert.Error(t, errN)
				return
			}
			assert.NoError(t, errN)
			cfgW.close()

			errW := <-cfgW.watcher
			if tt.expectWatchErr {
				assert.Error(t, errW)
				return
			}
			assert.NoError(t, errW)
		})
	}
}

func TestConfigWatcherNoWatcher(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)
	set := CollectorSettings{
		ConfigMapProvider: configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")),
		ConfigUnmarshaler: configunmarshaler.NewDefault(),
		Factories:         factories,
	}

	watcherWG := sync.WaitGroup{}
	cfgW, errN := newConfigWatcher(context.Background(), set)
	assert.NoError(t, errN)

	watcherWG.Add(1)
	go func() {
		errW, ok := <-cfgW.watcher
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	cfgW.close()
	watcherWG.Wait()
}

func TestConfigWatcherWhenClosed(t *testing.T) {
	factories, errF := componenttest.NopFactories()
	require.NoError(t, errF)
	set := CollectorSettings{
		ConfigMapProvider: func() configmapprovider.Provider {
			// Use errRetrieved with nil errors to have Watchable interface implemented.
			ret, err := configmapprovider.NewFile(path.Join("testdata", "otelcol-nop.yaml")).Retrieve(context.Background(), onChange)
			require.NoError(t, err)
			m, err := ret.Get(context.Background())
			require.NoError(t, err)
			return &errConfigMapProvider{ret: &errRetrieved{retM: m, errW: configsource.ErrSessionClosed}}
		}(),
		ConfigUnmarshaler: configunmarshaler.NewDefault(),
		Factories:         factories,
	}

	watcherWG := sync.WaitGroup{}
	cfgW, errN := newConfigWatcher(context.Background(), set)
	assert.NoError(t, errN)

	cfg, err := cfgW.get()
	assert.NotNil(t, cfg)
	assert.Nil(t, err)

	watcherWG.Add(1)
	go func() {
		errW, ok := <-cfgW.watcher
		// Channel is closed, no exception
		assert.False(t, ok)
		assert.NoError(t, errW)
		watcherWG.Done()
	}()

	cfgW.close()
	watcherWG.Wait()
}

func onChange(event *configmapprovider.ChangeEvent) {}
