// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pipeline"
	"go.opentelemetry.io/collector/service/internal/builders"
	"go.opentelemetry.io/collector/service/pipelines"
)

// ---------- minimal config shapes for reflection tests ----------

type fakeBatchSubConfig struct{}

type fakeSendingQueueConfig struct {
	Batch configoptional.Optional[fakeBatchSubConfig] `mapstructure:"batch"`
}

type batchingExporterConfig struct {
	SendingQueue configoptional.Optional[fakeSendingQueueConfig] `mapstructure:"sending_queue"`
}

func (*batchingExporterConfig) Validate() error { return nil }

type noBatchExporterConfig struct {
	SendingQueue configoptional.Optional[fakeSendingQueueConfig] `mapstructure:"sending_queue"`
}

func (*noBatchExporterConfig) Validate() error { return nil }

type noQueueExporterConfig struct{}

func (*noQueueExporterConfig) Validate() error { return nil }

func newBatchingExporterConfig() component.Config {
	return &batchingExporterConfig{
		SendingQueue: configoptional.Some(fakeSendingQueueConfig{
			Batch: configoptional.Some(fakeBatchSubConfig{}),
		}),
	}
}

func newNoBatchExporterConfig() component.Config {
	return &noBatchExporterConfig{
		SendingQueue: configoptional.Some(fakeSendingQueueConfig{
			Batch: configoptional.None[fakeBatchSubConfig](),
		}),
	}
}

func newNoQueueExporterConfig() component.Config {
	return &noQueueExporterConfig{}
}

func observedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zap.WarnLevel)
	return zap.New(core), logs
}

// newTestExporterBuilder creates an ExporterBuilder backed only by configs and no
// real factories.  warnDoubleBatching only reads configs so no factory calls happen.
func newTestExporterBuilder(cfgs map[component.ID]component.Config) *builders.ExporterBuilder {
	return builders.NewExporter(cfgs, map[component.Type]exporter.Factory{})
}

// ---------- unit tests for the reflection helper ----------

func TestExporterHasBatchingEnabled(t *testing.T) {
	assert.True(t, exporterHasBatchingEnabled(newBatchingExporterConfig()),
		"sending_queue.batch = Some → batching enabled")

	assert.False(t, exporterHasBatchingEnabled(newNoBatchExporterConfig()),
		"sending_queue.batch = None → batching not enabled")

	assert.False(t, exporterHasBatchingEnabled(newNoQueueExporterConfig()),
		"no sending_queue field → batching not enabled")

	assert.False(t, exporterHasBatchingEnabled(&batchingExporterConfig{
		SendingQueue: configoptional.None[fakeSendingQueueConfig](),
	}), "sending_queue = None → batching not enabled")
}

// ---------- integration-level tests for warnDoubleBatching ----------

// TestWarnDoubleBatching_SamePipeline verifies a warning is emitted when a
// batchprocessor and an exporter with batching enabled appear in the same pipeline.
func TestWarnDoubleBatching_SamePipeline(t *testing.T) {
	logger, logs := observedLogger()

	recvID := component.MustNewID("examplereceiver")
	batchProcID := component.MustNewID(batchProcessorType)
	exporterID := component.MustNewID("batchexporter")

	pipelineConfigs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{recvID},
			Processors: []component.ID{batchProcID},
			Exporters:  []component.ID{exporterID},
		},
	}

	warnDoubleBatching(logger, pipelineConfigs, newTestExporterBuilder(map[component.ID]component.Config{
		exporterID: newBatchingExporterConfig(),
	}))

	assert.Equal(t, 1, logs.Len(), "expected exactly one double-batching warning")

	entry := logs.All()[0]
	assert.Contains(t, entry.Message, "double batching detected")

	fields := logFields(entry)
	assert.Equal(t, pipeline.NewID(pipeline.SignalTraces).String(), fields["pipeline"])
	assert.Equal(t, exporterID.String(), fields["exporter"])

	path := fields["path"]
	assert.Contains(t, path, batchProcessorType, "path should name the batch processor")
	assert.Contains(t, path, exporterID.String(), "path should name the exporter")
}

// TestWarnDoubleBatching_DifferentPipelines is the critical negative case: when the
// batchprocessor and the batching-enabled exporter are on separate, non-overlapping
// pipelines/signal paths, no warning must be emitted.
func TestWarnDoubleBatching_DifferentPipelines(t *testing.T) {
	logger, logs := observedLogger()

	recvID := component.MustNewID("examplereceiver")
	batchProcID := component.MustNewID(batchProcessorType)
	plainExporterID := component.MustNewID("plainexporter") // no batching
	batchExporterID := component.MustNewID("batchexporter") // batching, but no batch processor in its pipeline

	pipelineConfigs := pipelines.Config{
		// batch processor present, but exporter has no batching.
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{recvID},
			Processors: []component.ID{batchProcID},
			Exporters:  []component.ID{plainExporterID},
		},
		// batching exporter, but no batch processor.
		pipeline.NewID(pipeline.SignalMetrics): {
			Receivers: []component.ID{recvID},
			Exporters: []component.ID{batchExporterID},
		},
	}

	warnDoubleBatching(logger, pipelineConfigs, newTestExporterBuilder(map[component.ID]component.Config{
		plainExporterID: newNoQueueExporterConfig(),
		batchExporterID: newBatchingExporterConfig(),
	}))

	assert.Equal(t, 0, logs.Len(), "no warning expected when both mechanisms are on distinct pipelines")
}

// TestWarnDoubleBatching_NoBatchProcessor ensures silence when there is no batch
// processor at all, even if the exporter has batching enabled.
func TestWarnDoubleBatching_NoBatchProcessor(t *testing.T) {
	logger, logs := observedLogger()

	pipelineConfigs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers: []component.ID{component.MustNewID("examplereceiver")},
			Exporters: []component.ID{component.MustNewID("batchexporter")},
		},
	}

	warnDoubleBatching(logger, pipelineConfigs, newTestExporterBuilder(map[component.ID]component.Config{
		component.MustNewID("batchexporter"): newBatchingExporterConfig(),
	}))

	assert.Equal(t, 0, logs.Len(), "no warning expected when batch processor is absent from pipeline")
}

// TestWarnDoubleBatching_MultipleExporters verifies that warnings are emitted per
// (batch-processor, batching-exporter) combination in a pipeline that fans out to
// multiple exporters, only for those that have batching enabled.
func TestWarnDoubleBatching_MultipleExporters(t *testing.T) {
	logger, logs := observedLogger()

	recvID := component.MustNewID("examplereceiver")
	batchProcID := component.MustNewID(batchProcessorType)
	batchExpID := component.MustNewID("batchexporter")
	plainExpID := component.MustNewID("plainexporter")

	pipelineConfigs := pipelines.Config{
		pipeline.NewID(pipeline.SignalTraces): {
			Receivers:  []component.ID{recvID},
			Processors: []component.ID{batchProcID},
			Exporters:  []component.ID{batchExpID, plainExpID},
		},
	}

	warnDoubleBatching(logger, pipelineConfigs, newTestExporterBuilder(map[component.ID]component.Config{
		batchExpID: newBatchingExporterConfig(),
		plainExpID: newNoQueueExporterConfig(),
	}))

	assert.Equal(t, 1, logs.Len(), "expected exactly one warning, only for the batching exporter")
	assert.Equal(t, batchExpID.String(), logFields(logs.All()[0])["exporter"])
}

// logFields converts a zap observed log entry's context fields to a string map.
func logFields(entry observer.LoggedEntry) map[string]string {
	m := make(map[string]string, len(entry.Context))
	for _, f := range entry.Context {
		m[f.Key] = f.String
	}
	return m
}
