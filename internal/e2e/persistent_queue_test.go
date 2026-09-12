// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pdata/testdata"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/extensions"
	"go.opentelemetry.io/collector/service/pipelines"
	"go.opentelemetry.io/collector/service/telemetry/otelconftelemetry"
)

func TestPersistentQueueConnectorForwardsToSecondPipeline(t *testing.T) {
	receiverType := component.MustNewType("persistent_queue_receiver")
	connectorType := component.MustNewType("persistent_queue_connector")
	exporterType := component.MustNewType("persistent_queue_exporter")
	storageType := component.MustNewType("memory_storage")

	receiverID := component.NewID(receiverType)
	connectorID := component.NewID(connectorType)
	exporterID := component.NewID(exporterType)
	storageID := component.NewID(storageType)

	receiverFactory := receiver.NewFactory(
		receiverType,
		func() component.Config { return &struct{}{} },
		receiver.WithLogs(
			func(_ context.Context, _ receiver.Settings, _ component.Config, next consumer.Logs) (receiver.Logs, error) {
				return &logsGeneratorReceiver{next: next}, nil
			},
			component.StabilityLevelDevelopment,
		),
	)

	connectorFactory := connector.NewFactory(
		connectorType,
		func() component.Config { return &struct{}{} },
		connector.WithLogsToLogs(
			func(ctx context.Context, set connector.Settings, cfg component.Config, next consumer.Logs) (connector.Logs, error) {
				queueCfg := exporterhelper.NewDefaultQueueConfig()
				queueCfg.StorageID = &storageID
				queueCfg.NumConsumers = 1
				queueCfg.Batch = configoptional.None[exporterhelper.BatchConfig]()

				return exporterhelper.NewLogs(
					ctx,
					exporter.Settings{
						ID:                set.ID,
						TelemetrySettings: set.TelemetrySettings,
						BuildInfo:         set.BuildInfo,
					},
					cfg,
					next.ConsumeLogs,
					exporterhelper.WithQueue(configoptional.Some(queueCfg)),
					exporterhelper.WithTimeout(exporterhelper.TimeoutConfig{Timeout: 0}),
				)
			},
			component.StabilityLevelDevelopment,
		),
	)

	sink := &consumertest.LogsSink{}
	exporterFactory := exporter.NewFactory(
		exporterType,
		func() component.Config { return &struct{}{} },
		exporter.WithLogs(
			func(context.Context, exporter.Settings, component.Config) (exporter.Logs, error) {
				return &logsSinkExporter{Logs: sink}, nil
			},
			component.StabilityLevelDevelopment,
		),
	)

	storageExtension := &memoryStorageExtension{
		client: &memoryStorageClient{items: map[string][]byte{}},
	}
	storageFactory := extension.NewFactory(
		storageType,
		func() component.Config { return &struct{}{} },
		func(context.Context, extension.Settings, component.Config) (extension.Extension, error) {
			return storageExtension, nil
		},
		component.StabilityLevelDevelopment,
	)

	telemetryFactory := otelconftelemetry.NewFactory()
	telemetryCfg := telemetryFactory.CreateDefaultConfig().(*otelconftelemetry.Config)
	telemetryCfg.Logs.Level = zapcore.ErrorLevel
	telemetryCfg.Metrics = otelconftelemetry.MetricsConfig{Level: configtelemetry.LevelNone}

	set := service.Settings{
		BuildInfo: component.NewDefaultBuildInfo(),
		ReceiversConfigs: map[component.ID]component.Config{
			receiverID: receiverFactory.CreateDefaultConfig(),
		},
		ReceiversFactories: map[component.Type]receiver.Factory{
			receiverType: receiverFactory,
		},
		ExportersConfigs: map[component.ID]component.Config{
			exporterID: exporterFactory.CreateDefaultConfig(),
		},
		ExportersFactories: map[component.Type]exporter.Factory{
			exporterType: exporterFactory,
		},
		ConnectorsConfigs: map[component.ID]component.Config{
			connectorID: connectorFactory.CreateDefaultConfig(),
		},
		ConnectorsFactories: map[component.Type]connector.Factory{
			connectorType: connectorFactory,
		},
		ExtensionsConfigs: map[component.ID]component.Config{
			storageID: storageFactory.CreateDefaultConfig(),
		},
		ExtensionsFactories: map[component.Type]extension.Factory{
			storageType: storageFactory,
		},
		TelemetryFactory: telemetryFactory,
	}

	cfg := service.Config{
		Telemetry:  telemetryCfg,
		Extensions: extensions.Config{storageID},
		Pipelines: pipelines.Config{
			pipeline.NewIDWithName(pipeline.SignalLogs, "source"): {
				Receivers: []component.ID{receiverID},
				Exporters: []component.ID{connectorID},
			},
			pipeline.NewIDWithName(pipeline.SignalLogs, "destination"): {
				Receivers: []component.ID{connectorID},
				Exporters: []component.ID{exporterID},
			},
		},
	}

	srv, err := service.New(t.Context(), set, cfg)
	require.NoError(t, err)

	started := false
	t.Cleanup(func() {
		if started {
			assert.NoError(t, srv.Shutdown(context.Background()))
		}
	})

	require.NoError(t, srv.Start(t.Context()))
	started = true
	require.Eventually(t, func() bool {
		return sink.LogRecordCount() == 1
	}, 5*time.Second, 10*time.Millisecond)

	// Shutdown waits for the queue worker's final release of the unmarshaled request.
	require.NoError(t, srv.Shutdown(context.Background()))
	started = false
	require.Equal(t, 1, sink.LogRecordCount())
}

type logsGeneratorReceiver struct {
	next consumer.Logs
}

func (r *logsGeneratorReceiver) Start(ctx context.Context, _ component.Host) error {
	return r.next.ConsumeLogs(ctx, testdata.GenerateLogs(1))
}

func (*logsGeneratorReceiver) Shutdown(context.Context) error {
	return nil
}

type logsSinkExporter struct {
	component.StartFunc
	component.ShutdownFunc
	consumer.Logs
}

type memoryStorageExtension struct {
	component.StartFunc
	component.ShutdownFunc
	client storage.Client
}

func (e *memoryStorageExtension) GetClient(
	context.Context,
	component.Kind,
	component.ID,
	string,
) (storage.Client, error) {
	return e.client, nil
}

type memoryStorageClient struct {
	mu    sync.Mutex
	items map[string][]byte
}

func (c *memoryStorageClient) Get(ctx context.Context, key string) ([]byte, error) {
	op := storage.GetOperation(key)
	if err := c.Batch(ctx, op); err != nil {
		return nil, err
	}
	return op.Value, nil
}

func (c *memoryStorageClient) Set(ctx context.Context, key string, value []byte) error {
	return c.Batch(ctx, storage.SetOperation(key, value))
}

func (c *memoryStorageClient) Delete(ctx context.Context, key string) error {
	return c.Batch(ctx, storage.DeleteOperation(key))
}

func (c *memoryStorageClient) Batch(_ context.Context, ops ...*storage.Operation) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, op := range ops {
		switch op.Type {
		case storage.Get:
			op.Value = bytes.Clone(c.items[op.Key])
		case storage.Set:
			c.items[op.Key] = bytes.Clone(op.Value)
		case storage.Delete:
			delete(c.items, op.Key)
		default:
			return errors.New("unsupported storage operation")
		}
	}
	return nil
}

func (*memoryStorageClient) Close(context.Context) error {
	return nil
}
