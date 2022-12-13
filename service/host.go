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

package service // import "go.opentelemetry.io/collector/service"

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service/extensions"
)

var _ component.Host = (*serviceHost)(nil)

type serviceHost struct {
	asyncErrorChannel chan error
	receivers         *receiver.Builder
	processors        *processor.Builder
	exporters         *exporter.Builder
	connectors        *connector.Builder
	extensions        *extension.Builder

	buildInfo component.BuildInfo

	pipelines         *builtPipelines
	serviceExtensions *extensions.Extensions
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (host *serviceHost) ReportFatalError(err error) {
	host.asyncErrorChannel <- err
}

func (host *serviceHost) GetFactory(kind component.Kind, componentType component.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return host.receivers.Factory(componentType)
	case component.KindProcessor:
		return host.processors.Factory(componentType)
	case component.KindExporter:
		return host.exporters.Factory(componentType)
	case component.KindConnector:
		return host.connectors.Factory(componentType)
	case component.KindExtension:
		return host.extensions.Factory(componentType)
	}
	return nil
}

func (host *serviceHost) GetExtensions() map[component.ID]component.Component {
	return host.serviceExtensions.GetExtensions()
}

func (host *serviceHost) GetExporters() map[component.DataType]map[component.ID]component.Component {
	return host.pipelines.GetExporters()
}
