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

package pipelines

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/internal/testdata"
	"go.opentelemetry.io/collector/service/internal/configunmarshaler"
	"go.opentelemetry.io/collector/service/internal/testcomponents"
)

func TestBuild(t *testing.T) {
	tests := []struct {
		name             string
		receiverIDs      []component.ID
		processorIDs     []component.ID
		exporterIDs      []component.ID
		expectedRequests int
	}{
		{
			name:             "pipelines_simple.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver")},
			processorIDs:     []component.ID{component.NewID("exampleprocessor")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_simple_multi_proc.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver")},
			processorIDs:     []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_simple_no_proc.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter")},
			expectedRequests: 1,
		},
		{
			name:             "pipelines_multi.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
			processorIDs:     []component.ID{component.NewID("exampleprocessor"), component.NewIDWithName("exampleprocessor", "1")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
			expectedRequests: 2,
		},
		{
			name:             "pipelines_multi_no_proc.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver"), component.NewIDWithName("examplereceiver", "1")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter"), component.NewIDWithName("exampleexporter", "1")},
			expectedRequests: 2,
		},
		{
			name:             "pipelines_exporter_multi_pipeline.yaml",
			receiverIDs:      []component.ID{component.NewID("examplereceiver")},
			exporterIDs:      []component.ID{component.NewID("exampleexporter")},
			expectedRequests: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factories, err := testcomponents.ExampleComponents()
			assert.NoError(t, err)

			cfg := loadConfig(t, filepath.Join("testdata", test.name), factories)

			// Build the pipeline
			pipelines, err := Build(context.Background(), toSettings(factories, cfg))
			assert.NoError(t, err)

			assert.NoError(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))

			// Verify exporters created, started and empty.
			for _, expID := range test.exporterIDs {
				traceExporter := pipelines.GetExporters()[component.DataTypeTraces][expID].(*testcomponents.ExampleExporter)
				assert.True(t, traceExporter.Started)
				assert.Equal(t, len(traceExporter.Traces), 0)

				// Validate metrics.
				metricsExporter := pipelines.GetExporters()[component.DataTypeMetrics][expID].(*testcomponents.ExampleExporter)
				assert.True(t, metricsExporter.Started)
				assert.Zero(t, len(metricsExporter.Traces))

				// Validate logs.
				logsExporter := pipelines.GetExporters()[component.DataTypeLogs][expID].(*testcomponents.ExampleExporter)
				assert.True(t, logsExporter.Started)
				assert.Zero(t, len(logsExporter.Traces))
			}

			// Verify processors created in the given order and started.
			for i, procID := range test.processorIDs {
				traceProcessor := pipelines.pipelines[component.NewID(component.DataTypeTraces)].processors[i]
				assert.Equal(t, procID, traceProcessor.id)
				assert.True(t, traceProcessor.comp.(*testcomponents.ExampleProcessor).Started)

				// Validate metrics.
				metricsProcessor := pipelines.pipelines[component.NewID(component.DataTypeMetrics)].processors[i]
				assert.Equal(t, procID, metricsProcessor.id)
				assert.True(t, metricsProcessor.comp.(*testcomponents.ExampleProcessor).Started)

				// Validate logs.
				logsProcessor := pipelines.pipelines[component.NewID(component.DataTypeLogs)].processors[i]
				assert.Equal(t, procID, logsProcessor.id)
				assert.True(t, logsProcessor.comp.(*testcomponents.ExampleProcessor).Started)
			}

			// Verify receivers created, started and send data to confirm pipelines correctly connected.
			for _, recvID := range test.receiverIDs {
				traceReceiver := pipelines.allReceivers[component.DataTypeTraces][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, traceReceiver.Started)
				// Send traces.
				assert.NoError(t, traceReceiver.ConsumeTraces(context.Background(), testdata.GenerateTraces(1)))

				metricsReceiver := pipelines.allReceivers[component.DataTypeMetrics][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, metricsReceiver.Started)
				// Send metrics.
				assert.NoError(t, metricsReceiver.ConsumeMetrics(context.Background(), testdata.GenerateMetrics(1)))

				logsReceiver := pipelines.allReceivers[component.DataTypeLogs][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, logsReceiver.Started)
				// Send logs.
				assert.NoError(t, logsReceiver.ConsumeLogs(context.Background(), testdata.GenerateLogs(1)))
			}

			assert.NoError(t, pipelines.ShutdownAll(context.Background()))

			// Verify receivers shutdown.
			for _, recvID := range test.receiverIDs {
				traceReceiver := pipelines.allReceivers[component.DataTypeTraces][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, traceReceiver.Stopped)

				metricsReceiver := pipelines.allReceivers[component.DataTypeMetrics][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, metricsReceiver.Stopped)

				logsReceiver := pipelines.allReceivers[component.DataTypeLogs][recvID].(*testcomponents.ExampleReceiver)
				assert.True(t, logsReceiver.Stopped)
			}

			// Verify processors shutdown.
			for i := range test.processorIDs {
				traceProcessor := pipelines.pipelines[component.NewID(component.DataTypeTraces)].processors[i]
				assert.True(t, traceProcessor.comp.(*testcomponents.ExampleProcessor).Stopped)

				// Validate metrics.
				metricsProcessor := pipelines.pipelines[component.NewID(component.DataTypeMetrics)].processors[i]
				assert.True(t, metricsProcessor.comp.(*testcomponents.ExampleProcessor).Stopped)

				// Validate logs.
				logsProcessor := pipelines.pipelines[component.NewID(component.DataTypeLogs)].processors[i]
				assert.True(t, logsProcessor.comp.(*testcomponents.ExampleProcessor).Stopped)
			}

			// Now verify that exporters received data, and are shutdown.
			for _, expID := range test.exporterIDs {
				// Validate traces.
				traceExporter := pipelines.GetExporters()[component.DataTypeTraces][expID].(*testcomponents.ExampleExporter)
				require.Len(t, traceExporter.Traces, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateTraces(1), traceExporter.Traces[0])
				assert.True(t, traceExporter.Stopped)

				// Validate metrics.
				metricsExporter := pipelines.GetExporters()[component.DataTypeMetrics][expID].(*testcomponents.ExampleExporter)
				require.Len(t, metricsExporter.Metrics, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateMetrics(1), metricsExporter.Metrics[0])
				assert.True(t, metricsExporter.Stopped)

				// Validate logs.
				logsExporter := pipelines.GetExporters()[component.DataTypeLogs][expID].(*testcomponents.ExampleExporter)
				require.Len(t, logsExporter.Logs, test.expectedRequests)
				assert.EqualValues(t, testdata.GenerateLogs(1), logsExporter.Logs[0])
				assert.True(t, logsExporter.Stopped)
			}
		})
	}
}

func TestBuildErrors(t *testing.T) {
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()
	badReceiverFactory := newBadReceiverFactory()
	badProcessorFactory := newBadProcessorFactory()
	badExporterFactory := newBadExporterFactory()

	tests := []struct {
		configFile string
	}{
		{configFile: "not_supported_exporter_logs.yaml"},
		{configFile: "not_supported_exporter_metrics.yaml"},
		{configFile: "not_supported_exporter_traces.yaml"},
		{configFile: "not_supported_processor_logs.yaml"},
		{configFile: "not_supported_processor_metrics.yaml"},
		{configFile: "not_supported_processor_traces.yaml"},
		{configFile: "not_supported_receiver_traces.yaml"},
		{configFile: "not_supported_receiver_metrics.yaml"},
		{configFile: "not_supported_receiver_traces.yaml"},
		{configFile: "unknown_exporter_config.yaml"},
		{configFile: "unknown_exporter_factory.yaml"},
		{configFile: "unknown_processor_config.yaml"},
		{configFile: "unknown_processor_factory.yaml"},
		{configFile: "unknown_receiver_config.yaml"},
		{configFile: "unknown_receiver_factory.yaml"},
	}

	for _, test := range tests {
		t.Run(test.configFile, func(t *testing.T) {
			factories := component.Factories{
				Receivers: map[component.Type]component.ReceiverFactory{
					nopReceiverFactory.Type(): nopReceiverFactory,
					"unknown":                 nopReceiverFactory,
					badReceiverFactory.Type(): badReceiverFactory,
				},
				Processors: map[component.Type]component.ProcessorFactory{
					nopProcessorFactory.Type(): nopProcessorFactory,
					"unknown":                  nopProcessorFactory,
					badProcessorFactory.Type(): badProcessorFactory,
				},
				Exporters: map[component.Type]component.ExporterFactory{
					nopExporterFactory.Type(): nopExporterFactory,
					"unknown":                 nopExporterFactory,
					badExporterFactory.Type(): badExporterFactory,
				},
			}

			// Need the unknown factories to do unmarshalling.
			cfg := loadConfig(t, filepath.Join("testdata", test.configFile), factories)

			// Remove the unknown factories, so they are NOT available during building.
			delete(factories.Exporters, "unknown")
			delete(factories.Processors, "unknown")
			delete(factories.Receivers, "unknown")

			_, err := Build(context.Background(), toSettings(factories, cfg))
			assert.Error(t, err)
		})
	}
}

func TestFailToStartAndShutdown(t *testing.T) {
	errReceiverFactory := newErrReceiverFactory()
	errProcessorFactory := newErrProcessorFactory()
	errExporterFactory := newErrExporterFactory()
	nopReceiverFactory := componenttest.NewNopReceiverFactory()
	nopProcessorFactory := componenttest.NewNopProcessorFactory()
	nopExporterFactory := componenttest.NewNopExporterFactory()

	set := Settings{
		Telemetry: componenttest.NewNopTelemetrySettings(),
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiverFactories: map[component.Type]component.ReceiverFactory{
			nopReceiverFactory.Type(): nopReceiverFactory,
			errReceiverFactory.Type(): errReceiverFactory,
		},
		ReceiverConfigs: map[component.ID]component.ReceiverConfig{
			component.NewID(nopReceiverFactory.Type()): nopReceiverFactory.CreateDefaultConfig(),
			component.NewID(errReceiverFactory.Type()): errReceiverFactory.CreateDefaultConfig(),
		},
		ProcessorFactories: map[component.Type]component.ProcessorFactory{
			nopProcessorFactory.Type(): nopProcessorFactory,
			errProcessorFactory.Type(): errProcessorFactory,
		},
		ProcessorConfigs: map[component.ID]component.ProcessorConfig{
			component.NewID(nopProcessorFactory.Type()): nopProcessorFactory.CreateDefaultConfig(),
			component.NewID(errProcessorFactory.Type()): errProcessorFactory.CreateDefaultConfig(),
		},
		ExporterFactories: map[component.Type]component.ExporterFactory{
			nopExporterFactory.Type(): nopExporterFactory,
			errExporterFactory.Type(): errExporterFactory,
		},
		ExporterConfigs: map[component.ID]component.ExporterConfig{
			component.NewID(nopExporterFactory.Type()): nopExporterFactory.CreateDefaultConfig(),
			component.NewID(errExporterFactory.Type()): errExporterFactory.CreateDefaultConfig(),
		},
	}

	for _, dt := range []component.DataType{component.DataTypeTraces, component.DataTypeMetrics, component.DataTypeLogs} {
		t.Run(string(dt)+"/receiver", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*config.Pipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop"), component.NewID("err")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/processor", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*config.Pipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop"), component.NewID("err")},
					Exporters:  []component.ID{component.NewID("nop")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})

		t.Run(string(dt)+"/exporter", func(t *testing.T) {
			set.PipelineConfigs = map[component.ID]*config.Pipeline{
				component.NewID(dt): {
					Receivers:  []component.ID{component.NewID("nop")},
					Processors: []component.ID{component.NewID("nop")},
					Exporters:  []component.ID{component.NewID("nop"), component.NewID("err")},
				},
			}
			pipelines, err := Build(context.Background(), set)
			assert.NoError(t, err)
			assert.Error(t, pipelines.StartAll(context.Background(), componenttest.NewNopHost()))
			assert.Error(t, pipelines.ShutdownAll(context.Background()))
		})
	}
}

func newBadReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory("bf", func() component.ReceiverConfig {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(component.NewID("bf")),
		}
	})
}

func newBadProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory("bf", func() component.ProcessorConfig {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(component.NewID("bf")),
		}
	})
}

func newBadExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory("bf", func() component.ExporterConfig {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(component.NewID("bf")),
		}
	})
}

func newErrReceiverFactory() component.ReceiverFactory {
	return component.NewReceiverFactory("err", func() component.ReceiverConfig {
		return &struct {
			config.ReceiverSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ReceiverSettings: config.NewReceiverSettings(component.NewID("bf")),
		}
	},
		component.WithTracesReceiver(func(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Traces) (component.TracesReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsReceiver(func(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Logs) (component.LogsReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsReceiver(func(context.Context, component.ReceiverCreateSettings, component.ReceiverConfig, consumer.Metrics) (component.MetricsReceiver, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrProcessorFactory() component.ProcessorFactory {
	return component.NewProcessorFactory("err", func() component.ProcessorConfig {
		return &struct {
			config.ProcessorSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ProcessorSettings: config.NewProcessorSettings(component.NewID("bf")),
		}
	},
		component.WithTracesProcessor(func(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Traces) (component.TracesProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsProcessor(func(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Logs) (component.LogsProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsProcessor(func(context.Context, component.ProcessorCreateSettings, component.ProcessorConfig, consumer.Metrics) (component.MetricsProcessor, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func newErrExporterFactory() component.ExporterFactory {
	return component.NewExporterFactory("err", func() component.ExporterConfig {
		return &struct {
			config.ExporterSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct
		}{
			ExporterSettings: config.NewExporterSettings(component.NewID("bf")),
		}
	},
		component.WithTracesExporter(func(context.Context, component.ExporterCreateSettings, component.ExporterConfig) (component.TracesExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithLogsExporter(func(context.Context, component.ExporterCreateSettings, component.ExporterConfig) (component.LogsExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
		component.WithMetricsExporter(func(context.Context, component.ExporterCreateSettings, component.ExporterConfig) (component.MetricsExporter, error) {
			return &errComponent{}, nil
		}, component.StabilityLevelUndefined),
	)
}

func toSettings(factories component.Factories, cfg *configSettings) Settings {
	return Settings{
		Telemetry:          componenttest.NewNopTelemetrySettings(),
		BuildInfo:          component.NewDefaultBuildInfo(),
		ReceiverFactories:  factories.Receivers,
		ReceiverConfigs:    cfg.Receivers.GetReceivers(),
		ProcessorFactories: factories.Processors,
		ProcessorConfigs:   cfg.Processors.GetProcessors(),
		ExporterFactories:  factories.Exporters,
		ExporterConfigs:    cfg.Exporters.GetExporters(),
		PipelineConfigs:    cfg.Service.Pipelines,
	}
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

// TODO: Remove this by not reading the input from the files, or by providing something similar outside service package.
type configSettings struct {
	Receivers  *configunmarshaler.Receivers  `mapstructure:"receivers"`
	Processors *configunmarshaler.Processors `mapstructure:"processors"`
	Exporters  *configunmarshaler.Exporters  `mapstructure:"exporters"`
	Service    *serviceSettings              `mapstructure:"service"`
}

type serviceSettings struct {
	Pipelines map[component.ID]*config.Pipeline `mapstructure:"pipelines"`
}

func loadConfig(t *testing.T, fileName string, factories component.Factories) *configSettings {
	// Read yaml config from file
	conf, err := confmaptest.LoadConf(fileName)
	require.NoError(t, err)
	cfg := &configSettings{
		Receivers:  configunmarshaler.NewReceivers(factories.Receivers),
		Processors: configunmarshaler.NewProcessors(factories.Processors),
		Exporters:  configunmarshaler.NewExporters(factories.Exporters),
	}
	require.NoError(t, conf.Unmarshal(cfg, confmap.WithErrorUnused()))
	return cfg
}
