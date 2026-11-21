// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configstorage"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
)

type wrappedLogsProcessor struct {
	processor.Logs
}

// queueProcessorFactory builds processors on exporterhelper, as the queue_batch processor does.
func queueProcessorFactory(typ component.Type, storage, wrapped bool) processor.Factory {
	create := func(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
		qCfg := exporterhelper.NewDefaultQueueConfig()
		if storage {
			qCfg.StorageID = configoptional.Some(configstorage.ID(component.MustNewID("file_storage")))
		}
		p, err := exporterhelper.NewLogs(ctx,
			exporter.Settings{ID: set.ID, TelemetrySettings: set.TelemetrySettings, BuildInfo: set.BuildInfo},
			cfg, next.ConsumeLogs, exporterhelper.WithQueue(configoptional.Some(qCfg)))
		if err != nil || !wrapped {
			return p, err
		}
		return &wrappedLogsProcessor{Logs: p}, nil
	}
	return processor.NewFactory(typ, func() component.Config { return struct{}{} },
		processor.WithLogs(create, component.StabilityLevelAlpha))
}

func TestProcessorPersistentQueueSharedByPipelines(t *testing.T) {
	procType := component.MustNewType("queueproc")
	shared := component.NewID(procType)
	procA := component.NewIDWithName(procType, "a")
	procB := component.NewIDWithName(procType, "b")
	logsA := pipeline.NewIDWithName(pipeline.SignalLogs, "tenant_a")
	logsB := pipeline.NewIDWithName(pipeline.SignalLogs, "tenant_b")
	logsC := pipeline.NewIDWithName(pipeline.SignalLogs, "tenant_c")

	for _, tt := range []struct {
		name       string
		storage    bool
		wrapped    bool
		processors map[pipeline.ID]component.ID
		wantErr    string
	}{
		{
			name:       "shared_with_storage",
			storage:    true,
			processors: map[pipeline.ID]component.ID{logsA: shared, logsB: shared, logsC: procA},
			wantErr: `processor "queueproc" has a persistent queue and is used in pipelines ["logs/tenant_a" "logs/tenant_b"], ` +
				`which would share one stored queue: configure a separate instance for each pipeline, for example "queueproc/<name>"`,
		},
		{
			name:       "separate_instances_with_storage",
			storage:    true,
			processors: map[pipeline.ID]component.ID{logsA: procA, logsB: procB},
		},
		{
			name:       "single_pipeline_with_storage",
			storage:    true,
			processors: map[pipeline.ID]component.ID{logsA: shared},
		},
		{
			name:       "shared_without_storage",
			processors: map[pipeline.ID]component.ID{logsA: shared, logsB: shared},
		},
		{
			// A wrapper that hides the exporterhelper component is not detected.
			name:       "shared_with_storage_opaque_wrapper",
			storage:    true,
			wrapped:    true,
			processors: map[pipeline.ID]component.ID{logsA: shared, logsB: shared},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			nop := component.MustNewID("nop")
			procFactory := queueProcessorFactory(procType, tt.storage, tt.wrapped)
			procCfgs := map[component.ID]component.Config{}
			pipelineCfgs := pipelines.Config{}
			for pipelineID, procID := range tt.processors {
				procCfgs[procID] = procFactory.CreateDefaultConfig()
				pipelineCfgs[pipelineID] = &pipelines.PipelineConfig{
					Receivers:  []component.ID{nop},
					Processors: []component.ID{procID},
					Exporters:  []component.ID{nop},
				}
			}
			rcvrFactory := receivertest.NewNopFactory()
			exprFactory := exportertest.NewNopFactory()

			g, err := Build(context.Background(), Settings{
				Telemetry: componenttest.NewNopTelemetrySettings(),
				BuildInfo: component.NewDefaultBuildInfo(),
				ReceiverBuilder: builders.NewReceiver(
					map[component.ID]component.Config{nop: rcvrFactory.CreateDefaultConfig()},
					map[component.Type]receiver.Factory{nop.Type(): rcvrFactory},
				),
				ProcessorBuilder: builders.NewProcessor(
					procCfgs,
					map[component.Type]processor.Factory{procType: procFactory},
				),
				ExporterBuilder: builders.NewExporter(
					map[component.ID]component.Config{nop: exprFactory.CreateDefaultConfig()},
					map[component.Type]exporter.Factory{nop.Type(): exprFactory},
				),
				ConnectorBuilder: builders.NewConnector(nil, map[component.Type]connector.Factory{}),
				PipelineConfigs:  pipelineCfgs,
			})
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				// The graph is returned so that the caller can shut down what was built.
				require.NotNil(t, g)
				return
			}
			require.NoError(t, err)
		})
	}
}
