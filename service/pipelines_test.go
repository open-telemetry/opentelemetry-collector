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

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
)

func newBadReceiverFactory() receiver.Factory {
	return receiver.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadProcessorFactory() processor.Factory {
	return processor.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadExporterFactory() exporter.Factory {
	return exporter.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newBadConnectorFactory() connector.Factory {
	return connector.NewFactory("bf", func() component.Config {
		return &struct{}{}
	})
}

func newErrReceiverFactory() receiver.Factory {
	return receiver.NewFactory("err",
		func() component.Config { return &struct{}{} },
		receiver.WithTraces(func(context.Context, receiver.CreateSettings, component.Config, consumer.Traces) (receiver.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithLogs(func(context.Context, receiver.CreateSettings, component.Config, consumer.Logs) (receiver.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		receiver.WithMetrics(func(context.Context, receiver.CreateSettings, component.Config, consumer.Metrics) (receiver.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() processor.Factory {
	return processor.NewFactory("err",
		func() component.Config { return &struct{}{} },
		processor.WithTraces(func(context.Context, processor.CreateSettings, component.Config, consumer.Traces) (processor.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithLogs(func(context.Context, processor.CreateSettings, component.Config, consumer.Logs) (processor.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		processor.WithMetrics(func(context.Context, processor.CreateSettings, component.Config, consumer.Metrics) (processor.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() exporter.Factory {
	return exporter.NewFactory("err",
		func() component.Config { return &struct{}{} },
		exporter.WithTraces(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithLogs(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		exporter.WithMetrics(func(context.Context, exporter.CreateSettings, component.Config) (exporter.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrConnectorFactory() connector.Factory {
	return connector.NewFactory("err", func() component.Config {
		return &struct{}{}
	},
		connector.WithTracesToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithTracesToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Traces, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithMetricsToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithMetricsToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Metrics, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),

		connector.WithLogsToTraces(func(context.Context, connector.CreateSettings, component.Config, consumer.Traces) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToMetrics(func(context.Context, connector.CreateSettings, component.Config, consumer.Metrics) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
		connector.WithLogsToLogs(func(context.Context, connector.CreateSettings, component.Config, consumer.Logs) (connector.Logs, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUnmaintained),
	)
}

type errComponent struct {
	consumertest.Consumer
}

func (e errComponent) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (e errComponent) Start(context.Context, component.Host) error {
	return errors.New("my error")
}

func (e errComponent) Shutdown(context.Context) error {
	return errors.New("my error")
}
