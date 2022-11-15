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
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/id"
	"go.opentelemetry.io/collector/component/status"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/internal/components"
	"go.opentelemetry.io/collector/service/internal/pipelines"
)

var _ component.Host = (*serviceHost)(nil)

type serviceHost struct {
	asyncErrorChannel   chan error
	factories           component.Factories
	buildInfo           component.BuildInfo
	telemetry           component.TelemetrySettings
	statusNotifications *components.Notifications

	pipelines  *pipelines.Pipelines
	extensions *extensions.Extensions
}

func newServiceHost(set *settings, telemetrySettings component.TelemetrySettings) *serviceHost {
	return &serviceHost{
		factories:           set.Factories,
		buildInfo:           set.BuildInfo,
		asyncErrorChannel:   set.AsyncErrorChannel,
		statusNotifications: components.NewNotifications(),
		telemetry:           telemetrySettings,
	}
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
// Deprecated: [0.65.0] Use ReportComponentStatus instead (with an event of type status.ComponentError).
func (host *serviceHost) ReportFatalError(err error) {
	host.asyncErrorChannel <- err
}

func (host *serviceHost) GetFactory(kind component.Kind, componentType id.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return host.factories.Receivers[componentType]
	case component.KindProcessor:
		return host.factories.Processors[componentType]
	case component.KindExporter:
		return host.factories.Exporters[componentType]
	case component.KindExtension:
		return host.factories.Extensions[componentType]
	}
	return nil
}

func (host *serviceHost) GetExtensions() map[id.ID]component.Extension {
	return host.extensions.GetExtensions()
}

func (host *serviceHost) GetExporters() map[component.DataType]map[id.ID]component.Exporter {
	return host.pipelines.GetExporters()
}

func (host *serviceHost) RegisterStatusListener(options ...status.ListenerOption) component.StatusListenerUnregisterFunc {
	return host.statusNotifications.RegisterListener(options...)
}

// ReportComponentStatus is an implementation of Host.ReportComponentStatus.
func (host *serviceHost) ReportComponentStatus(event *status.ComponentEvent) {
	if err := host.statusNotifications.SetComponentStatus(event); err != nil {
		host.telemetry.Logger.Warn("Service failed to report status", zap.Error(err))
	}
}
