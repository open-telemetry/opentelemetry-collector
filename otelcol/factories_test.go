// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelcol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	"go.opentelemetry.io/collector/service/telemetry"
)

func nopFactories() (Factories, error) {
	var factories Factories
	var err error

	if factories.Connectors, err = MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
	for _, con := range factories.Connectors {
		factories.ConnectorModules[con.Type()] = "go.opentelemetry.io/collector/connector/connectortest v1.2.3"
	}

	if factories.Extensions, err = MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
	for _, ext := range factories.Extensions {
		factories.ExtensionModules[ext.Type()] = "go.opentelemetry.io/collector/extension/extensiontest v1.2.3"
	}

	if factories.Receivers, err = MakeFactoryMap(receivertest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
	for _, rec := range factories.Receivers {
		factories.ReceiverModules[rec.Type()] = "go.opentelemetry.io/collector/receiver/receivertest v1.2.3"
	}

	if factories.Exporters, err = MakeFactoryMap(exportertest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
	for _, exp := range factories.Exporters {
		factories.ExporterModules[exp.Type()] = "go.opentelemetry.io/collector/exporter/exportertest v1.2.3"
	}

	if factories.Processors, err = MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
	for _, proc := range factories.Processors {
		factories.ProcessorModules[proc.Type()] = "go.opentelemetry.io/collector/processor/processortest v1.2.3"
	}

	factories.Telemetry = telemetry.NewFactory(func() component.Config {
		return fakeTelemetryConfig{}
	})

	return factories, err
}

func TestMakeFactoryMap(t *testing.T) {
	type testCase struct {
		name string
		in   []component.Factory
		out  map[component.Type]component.Factory
	}

	fRec := receiver.NewFactory(component.MustNewType("rec"), nil)
	fRec2 := receiver.NewFactory(component.MustNewType("rec"), nil)
	fRec3 := xreceiver.NewFactory(component.MustNewType("new_rec"), nil, xreceiver.WithDeprecatedTypeAlias(component.MustNewType("rec")))
	fPro := processor.NewFactory(component.MustNewType("pro"), nil)
	fCon := connector.NewFactory(component.MustNewType("con"), nil)
	fExp := exporter.NewFactory(component.MustNewType("exp"), nil)
	fExt := extension.NewFactory(component.MustNewType("ext"), nil, nil, component.StabilityLevelUndefined)
	testCases := []testCase{
		{
			name: "different names",
			in:   []component.Factory{fRec, fPro, fCon, fExp, fExt},
			out: map[component.Type]component.Factory{
				fRec.Type(): fRec,
				fPro.Type(): fPro,
				fCon.Type(): fCon,
				fExp.Type(): fExp,
				fExt.Type(): fExt,
			},
		},
		{
			name: "same name",
			in:   []component.Factory{fRec, fPro, fCon, fExp, fExt, fRec2},
		},
		{
			name: "with deprecated alias",
			in:   []component.Factory{fRec3},
			out: map[component.Type]component.Factory{
				fRec3.Type():                 fRec3,
				component.MustNewType("rec"): fRec3,
			},
		},
		{
			name: "conflicting alias name",
			in:   []component.Factory{fRec, fRec3},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			out, err := MakeFactoryMap(tt.in...)
			if tt.out == nil {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.out, out)
		})
	}
}

// configurableConfig is a config struct with many fields to stress reflect.DeepEqual
// during config comparison in partial reload benchmarks.
type configurableConfig struct {
	Endpoint    string            `mapstructure:"endpoint"`
	Timeout     int               `mapstructure:"timeout"`
	RetryCount  int               `mapstructure:"retry_count"`
	BatchSize   int               `mapstructure:"batch_size"`
	Enabled     bool              `mapstructure:"enabled"`
	Compression string            `mapstructure:"compression"`
	Headers     map[string]string `mapstructure:"headers"`
	Tags        []string          `mapstructure:"tags"`
	Metadata    struct {
		Name        string `mapstructure:"name"`
		Version     string `mapstructure:"version"`
		Environment string `mapstructure:"environment"`
		Region      string `mapstructure:"region"`
	} `mapstructure:"metadata"`
	Advanced struct {
		BufferSize     int    `mapstructure:"buffer_size"`
		FlushInterval  int    `mapstructure:"flush_interval"`
		MaxConnections int    `mapstructure:"max_connections"`
		KeepAlive      bool   `mapstructure:"keep_alive"`
		Protocol       string `mapstructure:"protocol"`
	} `mapstructure:"advanced"`
}

var configurableType = component.MustNewType("configurable")

// newConfigurableReceiverFactory creates a receiver factory with a config that has many fields.
func newConfigurableReceiverFactory() receiver.Factory {
	return receiver.NewFactory(
		configurableType,
		func() component.Config { return &configurableConfig{} },
		receiver.WithTraces(func(_ context.Context, _ receiver.Settings, _ component.Config, _ consumer.Traces) (receiver.Traces, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
		receiver.WithMetrics(func(_ context.Context, _ receiver.Settings, _ component.Config, _ consumer.Metrics) (receiver.Metrics, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
		receiver.WithLogs(func(_ context.Context, _ receiver.Settings, _ component.Config, _ consumer.Logs) (receiver.Logs, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
	)
}

// newConfigurableProcessorFactory creates a processor factory with a config that has many fields.
func newConfigurableProcessorFactory() processor.Factory {
	return processor.NewFactory(
		configurableType,
		func() component.Config { return &configurableConfig{} },
		processor.WithTraces(func(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Traces) (processor.Traces, error) {
			return &nopProcessor{Traces: next}, nil
		}, component.StabilityLevelStable),
		processor.WithMetrics(func(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Metrics) (processor.Metrics, error) {
			return &nopProcessor{Metrics: next}, nil
		}, component.StabilityLevelStable),
		processor.WithLogs(func(_ context.Context, _ processor.Settings, _ component.Config, next consumer.Logs) (processor.Logs, error) {
			return &nopProcessor{Logs: next}, nil
		}, component.StabilityLevelStable),
	)
}

// newConfigurableExporterFactory creates an exporter factory with a config that has many fields.
func newConfigurableExporterFactory() exporter.Factory {
	return exporter.NewFactory(
		configurableType,
		func() component.Config { return &configurableConfig{} },
		exporter.WithTraces(func(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Traces, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
		exporter.WithMetrics(func(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Metrics, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
		exporter.WithLogs(func(_ context.Context, _ exporter.Settings, _ component.Config) (exporter.Logs, error) {
			return &nopComponent{}, nil
		}, component.StabilityLevelStable),
	)
}

// newConfigurableConnectorFactory creates a connector factory with a config that has many fields.
func newConfigurableConnectorFactory() connector.Factory {
	return connector.NewFactory(
		configurableType,
		func() component.Config { return &configurableConfig{} },
		connector.WithTracesToTraces(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Traces) (connector.Traces, error) {
			return &nopProcessor{Traces: next}, nil
		}, component.StabilityLevelStable),
		connector.WithTracesToMetrics(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Metrics) (connector.Traces, error) {
			return &nopConnector{Metrics: next}, nil
		}, component.StabilityLevelStable),
		connector.WithTracesToLogs(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Logs) (connector.Traces, error) {
			return &nopConnector{Logs: next}, nil
		}, component.StabilityLevelStable),
		connector.WithMetricsToTraces(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Traces) (connector.Metrics, error) {
			return &nopConnector{Traces: next}, nil
		}, component.StabilityLevelStable),
		connector.WithMetricsToMetrics(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Metrics) (connector.Metrics, error) {
			return &nopProcessor{Metrics: next}, nil
		}, component.StabilityLevelStable),
		connector.WithMetricsToLogs(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Logs) (connector.Metrics, error) {
			return &nopConnector{Logs: next}, nil
		}, component.StabilityLevelStable),
		connector.WithLogsToTraces(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Traces) (connector.Logs, error) {
			return &nopConnector{Traces: next}, nil
		}, component.StabilityLevelStable),
		connector.WithLogsToMetrics(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Metrics) (connector.Logs, error) {
			return &nopConnector{Metrics: next}, nil
		}, component.StabilityLevelStable),
		connector.WithLogsToLogs(func(_ context.Context, _ connector.Settings, _ component.Config, next consumer.Logs) (connector.Logs, error) {
			return &nopProcessor{Logs: next}, nil
		}, component.StabilityLevelStable),
	)
}

// nopComponent is a minimal component that does nothing.
type nopComponent struct {
	component.StartFunc
	component.ShutdownFunc
}

func (n *nopComponent) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (n *nopComponent) ConsumeTraces(context.Context, ptrace.Traces) error   { return nil }
func (n *nopComponent) ConsumeMetrics(context.Context, pmetric.Metrics) error { return nil }
func (n *nopComponent) ConsumeLogs(context.Context, plog.Logs) error          { return nil }

// nopProcessor passes data through to the next consumer.
type nopProcessor struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

func (n *nopProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

// nopConnector converts between signal types (does nothing, just satisfies interface).
type nopConnector struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Traces
	consumer.Metrics
	consumer.Logs
}

func (n *nopConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{}
}

func (n *nopConnector) ConsumeTraces(context.Context, ptrace.Traces) error   { return nil }
func (n *nopConnector) ConsumeMetrics(context.Context, pmetric.Metrics) error { return nil }
func (n *nopConnector) ConsumeLogs(context.Context, plog.Logs) error          { return nil }

// configurableFactories returns factories that use configs with many fields,
// suitable for benchmarking reflect.DeepEqual performance.
func configurableFactories() (Factories, error) {
	var factories Factories
	var err error

	if factories.Connectors, err = MakeFactoryMap(newConfigurableConnectorFactory()); err != nil {
		return Factories{}, err
	}
	factories.ConnectorModules = make(map[component.Type]string, len(factories.Connectors))
	for _, con := range factories.Connectors {
		factories.ConnectorModules[con.Type()] = "go.opentelemetry.io/collector/connector/connectortest v1.2.3"
	}

	if factories.Extensions, err = MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExtensionModules = make(map[component.Type]string, len(factories.Extensions))
	for _, ext := range factories.Extensions {
		factories.ExtensionModules[ext.Type()] = "go.opentelemetry.io/collector/extension/extensiontest v1.2.3"
	}

	if factories.Receivers, err = MakeFactoryMap(newConfigurableReceiverFactory()); err != nil {
		return Factories{}, err
	}
	factories.ReceiverModules = make(map[component.Type]string, len(factories.Receivers))
	for _, rec := range factories.Receivers {
		factories.ReceiverModules[rec.Type()] = "go.opentelemetry.io/collector/receiver/receivertest v1.2.3"
	}

	if factories.Exporters, err = MakeFactoryMap(newConfigurableExporterFactory()); err != nil {
		return Factories{}, err
	}
	factories.ExporterModules = make(map[component.Type]string, len(factories.Exporters))
	for _, exp := range factories.Exporters {
		factories.ExporterModules[exp.Type()] = "go.opentelemetry.io/collector/exporter/exportertest v1.2.3"
	}

	if factories.Processors, err = MakeFactoryMap(newConfigurableProcessorFactory()); err != nil {
		return Factories{}, err
	}
	factories.ProcessorModules = make(map[component.Type]string, len(factories.Processors))
	for _, proc := range factories.Processors {
		factories.ProcessorModules[proc.Type()] = "go.opentelemetry.io/collector/processor/processortest v1.2.3"
	}

	factories.Telemetry = telemetry.NewFactory(func() component.Config {
		return fakeTelemetryConfig{}
	})

	return factories, err
}
