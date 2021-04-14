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

package parserprovider

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/experimental/configsource"
	"go.opentelemetry.io/collector/experimental/configsource/envvarconfigsource"
	"go.opentelemetry.io/collector/experimental/configsource/fileconfigsource"
	"go.opentelemetry.io/collector/experimental/configsource/vaultconfigsource"
)

type configSourceParserProvider struct {
	pp           ParserProvider
	logger       *zap.Logger
	appStartInfo component.ApplicationStartInfo
	csm          *configsource.Manager
}

// NewConfigSourceParserProvider creates a ParserProvider that uses config sources.
func NewConfigSourceParserProvider(logger *zap.Logger, appStartInfo component.ApplicationStartInfo) ParserProvider {
	return &configSourceParserProvider{
		pp:           Default(),
		logger:       logger,
		appStartInfo: appStartInfo,
	}
}

func (c *configSourceParserProvider) Get() (*config.Parser, error) {
	defaultParser, err := c.pp.Get()
	if err != nil {
		return nil, err
	}

	factories, err := makeFactoryMap(
		vaultconfigsource.NewFactory(),
		fileconfigsource.NewFactory(),
		envvarconfigsource.NewFactory(),
	)
	if err != nil {
		return nil, err
	}

	csm, err := configsource.NewManager(defaultParser, c.logger, c.appStartInfo, factories)
	if err != nil {
		return nil, err
	}

	parser, err := csm.Resolve(context.Background(), defaultParser)
	if err != nil {
		return nil, err
	}

	c.csm = csm
	return parser, nil
}

func (c *configSourceParserProvider) WatchForUpdate() error {
	return c.csm.WatchForUpdate()
}

func (c *configSourceParserProvider) Close(ctx context.Context) error {
	return c.csm.Close(ctx)
}

func makeFactoryMap(factories ...configsource.Factory) (configsource.Factories, error) {
	fMap := make(configsource.Factories)
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return nil, fmt.Errorf("duplicate config source factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
