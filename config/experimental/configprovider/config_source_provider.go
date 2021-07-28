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
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configparser"
	"go.opentelemetry.io/collector/service/parserprovider"
	"go.uber.org/zap"
)

type errDuplicatedConfigSourceFactory struct{ error }

type configSourceParserProvider struct {
	logger       *zap.Logger
	csm          *Manager
	configServer *configServer
	pp           parserprovider.ParserProvider
	buildInfo    component.BuildInfo
	factories    []Factory
}

// NewConfigSourceParserProvider creates a ParserProvider that uses config sources.
func NewConfigSourceParserProvider(pp parserprovider.ParserProvider, logger *zap.Logger, buildInfo component.BuildInfo, factories ...Factory) parserprovider.ParserProvider {
	if pp == nil {
		pp = parserprovider.Default()
	}
	return &configSourceParserProvider{
		pp:        pp,
		logger:    logger,
		factories: factories,
		buildInfo: buildInfo,
	}
}

// Get returns a config.Parser that wraps the parserprovider.Default() with a parser
// that can load and inject data from config sources. If there are no config sources
// in the configuration the returned parser behaves like the parserprovider.Default().
func (c *configSourceParserProvider) Get() (*configparser.Parser, error) {
	defaultParser, err := c.pp.Get()
	if err != nil {
		return nil, err
	}

	factories, err := makeFactoryMap(c.factories)
	if err != nil {
		return nil, err
	}

	csm, err := NewManager(defaultParser, c.logger, c.buildInfo, factories)
	if err != nil {
		return nil, err
	}

	parser, err := csm.Resolve(context.Background(), defaultParser)
	if err != nil {
		return nil, err
	}

	c.configServer = newConfigServer(c.logger, defaultParser.ToStringMap(), parser.ToStringMap())
	if err = c.configServer.start(); err != nil {
		return nil, err
	}

	c.csm = csm
	return parser, nil
}

// WatchForUpdate is used to monitor for updates on configuration values that
// were retrieved from config sources.
func (c *configSourceParserProvider) WatchForUpdate() error {
	return c.csm.WatchForUpdate()
}

// Close ends the watch for updates and closes the parser provider and respective
// config sources.
func (c *configSourceParserProvider) Close(ctx context.Context) error {
	if c.configServer != nil {
		_ = c.configServer.shutdown()
	}
	return c.csm.Close(ctx)
}

func makeFactoryMap(factories []Factory) (Factories, error) {
	fMap := make(Factories)
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return nil, &errDuplicatedConfigSourceFactory{fmt.Errorf("duplicate config source factory %q", f.Type())}
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
