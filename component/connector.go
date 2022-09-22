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

package component // import "go.opentelemetry.io/collector/component"

import (
	"go.opentelemetry.io/collector/config"
)

// ConnectorFactory is factory interface for connectors.
//
// This interface cannot be directly implemented. Implementations must
// use the NewConnectorFactory to implement it.
type ConnectorFactory interface {
	Factory

	NewExporterFactory() ExporterFactory
	NewReceiverFactory() ReceiverFactory

	// CreateDefaultConfig creates the default configuration for the Connector.
	// This method can be called multiple times depending on the pipeline
	// configuration and should not cause side-effects that prevent the creation
	// of multiple instances of the Connector.
	// The object returned by this method needs to pass the checks implemented by
	// 'configtest.CheckConfigStruct'. It is recommended to have these checks in the
	// tests of any implementation of the Factory interface.
	CreateDefaultConfig() config.Connector
}

// ConnectorCreateDefaultConfigFunc is the equivalent of ConnectorFactory.CreateDefaultConfig().
type ConnectorCreateDefaultConfigFunc func() config.Connector

// CreateDefaultConfig implements ConnectorFactory.CreateDefaultConfig().
func (f ConnectorCreateDefaultConfigFunc) CreateDefaultConfig() config.Connector {
	return f()
}
func (f ConnectorCreateDefaultConfigFunc) createDefaultReceiverConfig() config.Receiver {
	return f()
}
func (f ConnectorCreateDefaultConfigFunc) CreateDefaultExporterConfig() config.Exporter {
	return f()
}

type connectorFactory struct {
	baseFactory
	exporterFactoryOptions []ExporterFactoryOption
	receiverFactoryOptions []ReceiverFactoryOption
	ConnectorCreateDefaultConfigFunc
}

// NewConnectorFactory returns a ConnectorFactory.
func NewConnectorFactory(
	cfgType config.Type,
	createDefaultConfig ConnectorCreateDefaultConfigFunc,
	exporterFactoryOptions []ExporterFactoryOption,
	receiverFactoryOptions []ReceiverFactoryOption,
) ConnectorFactory {
	return &connectorFactory{
		baseFactory:                      baseFactory{cfgType: cfgType},
		exporterFactoryOptions:           exporterFactoryOptions,
		receiverFactoryOptions:           receiverFactoryOptions,
		ConnectorCreateDefaultConfigFunc: createDefaultConfig,
	}
}

func (f *connectorFactory) NewExporterFactory() ExporterFactory {
	return NewExporterFactory(f.cfgType, f.CreateDefaultExporterConfig, f.exporterFactoryOptions...)
}

func (f *connectorFactory) NewReceiverFactory() ReceiverFactory {
	return NewReceiverFactory(f.cfgType, f.createDefaultReceiverConfig, f.receiverFactoryOptions...)
}

// TODO Implement and enforce ConnectorFactoryOptions that enumerate valid signal combos.
//
// Example: nopconnector
// func AsLogsToLogsConnector() ConnectorFactoryOption
// func AsMetricsToMetricsConnector() ConnectorFactoryOption
// func AsTracesToTracesConnector() ConnectorFactoryOption
//
// Example: countconnector
// func AsLogsToMetricsConnector() ConnectorFactoryOption
// func AsMetricsToMetricsConnector() ConnectorFactoryOption
// func AsTracesToMetricsConnector() ConnectorFactoryOption
