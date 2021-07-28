// Copyright Splunk, Inc.
// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configprovider

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/config/experimental/configsource"
	"go.opentelemetry.io/collector/service/parserprovider"
	"go.uber.org/zap"
)

func TestConfigSourceParserProvider(t *testing.T) {
	tests := []struct {
		parserProvider parserprovider.ParserProvider
		wantErr        error
		name           string
		factories      []Factory
	}{
		{
			name: "success",
		},
		{
			name: "wrapped_parser_provider_get_error",
			parserProvider: &mockParserProvider{
				ErrOnGet: true,
			},
			wantErr: &errOnParserProviderGet{},
		},
		{
			name: "duplicated_factory_type",
			factories: []Factory{
				&mockCfgSrcFactory{},
				&mockCfgSrcFactory{},
			},
			wantErr: &errDuplicatedConfigSourceFactory{},
		},
		{
			name: "new_manager_builder_error",
			factories: []Factory{
				&mockCfgSrcFactory{
					ErrOnCreateConfigSource: errors.New("new_manager_builder_error forced error"),
				},
			},
			parserProvider: &fileParserProvider{
				FileName: path.Join("testdata", "basic_config.yaml"),
			},
			wantErr: &errConfigSourceCreation{},
		},
		{
			name: "manager_resolve_error",
			parserProvider: &fileParserProvider{
				FileName: path.Join("testdata", "manager_resolve_error.yaml"),
			},
			wantErr: fmt.Errorf("error not wrapped by specific error type: %w", configsource.ErrSessionClosed),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factories := tt.factories
			if factories == nil {
				factories = []Factory{
					&mockCfgSrcFactory{},
				}
			}

			pp := NewConfigSourceParserProvider(
				parserprovider.Default(),
				zap.NewNop(),
				component.DefaultBuildInfo(),
				factories...,
			)
			require.NotNil(t, pp)

			// Do not use the parserprovider.Default() to simplify the test setup.
			cspp := pp.(*configSourceParserProvider)
			cspp.pp = tt.parserProvider
			if cspp.pp == nil {
				cspp.pp = &mockParserProvider{}
			}

			cp, err := pp.Get()
			if tt.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, cp)
			} else {
				assert.IsType(t, tt.wantErr, err)
				assert.Nil(t, cp)
				return
			}

			var watchForUpdatedError error
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				watchForUpdatedError = pp.(parserprovider.Watchable).WatchForUpdate()
			}()
			cspp.csm.WaitForWatcher()

			closeErr := pp.(parserprovider.Closeable).Close(context.Background())
			assert.NoError(t, closeErr)

			wg.Wait()
			assert.Equal(t, configsource.ErrSessionClosed, watchForUpdatedError)
		})
	}
}

type mockParserProvider struct {
	ErrOnGet bool
}

var _ (parserprovider.ParserProvider) = (*mockParserProvider)(nil)

func (mpp *mockParserProvider) Get() (*configparser.Parser, error) {
	if mpp.ErrOnGet {
		return nil, &errOnParserProviderGet{errors.New("mockParserProvider.Get() forced test error")}
	}
	return configparser.NewParser(), nil
}

type errOnParserProviderGet struct{ error }

type fileParserProvider struct {
	FileName string
}

var _ (parserprovider.ParserProvider) = (*fileParserProvider)(nil)

func (fpp *fileParserProvider) Get() (*configparser.Parser, error) {
	return configparser.NewParserFromFile(fpp.FileName)
}
