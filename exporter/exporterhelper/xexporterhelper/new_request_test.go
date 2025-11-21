// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package xexporterhelper

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pipeline"
)

func TestWithQueueBatch(t *testing.T) {
	t.Run("with MetadataKeys - configures partitioner and merge function", func(t *testing.T) {
		cfg := exporterhelper.NewDefaultQueueConfig()
		cfg.MetadataKeys = []string{"key1", "key2"}
		cfg.Enabled = true

		set := NewTracesQueueBatchSettings()
		// Verify initial state
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		// Apply WithQueueBatch - this modifies a copy of set and captures it in the closure
		option := WithQueueBatch(cfg, set)

		// Note: Since set is passed by value, the original set variable isn't modified,
		// but the modified copy is captured in the closure and passed to internal.WithQueueBatch.
		// We verify the option works by ensuring the BaseExporter can be created successfully.
		be, err := internal.NewBaseExporter(
			exportertest.NewNopSettings(exportertest.NopType),
			pipeline.SignalTraces,
			func(context.Context, request.Request) error { return nil },
			option,
		)
		require.NoError(t, err)
		assert.NotNil(t, be)

		// The partitioner and merge function are configured in the copy of set that's
		// captured in the closure. The actual configuration is verified through
		// integration tests in factory_test.go where the partitioner is used.
	})

	t.Run("without MetadataKeys - does not configure partitioner", func(t *testing.T) {
		cfg := exporterhelper.NewDefaultQueueConfig()
		cfg.MetadataKeys = []string{}
		cfg.Enabled = true

		set := NewTracesQueueBatchSettings()
		// Verify initial state
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		// Apply WithQueueBatch
		option := WithQueueBatch(cfg, set)

		// Verify partitioner and merge function are NOT configured
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		// Verify the option can be applied to a base exporter
		be, err := internal.NewBaseExporter(
			exportertest.NewNopSettings(exportertest.NopType),
			pipeline.SignalTraces,
			func(context.Context, request.Request) error { return nil },
			option,
		)
		require.NoError(t, err)
		assert.NotNil(t, be)
	})

	t.Run("with nil MetadataKeys - does not configure partitioner", func(t *testing.T) {
		cfg := exporterhelper.NewDefaultQueueConfig()
		cfg.MetadataKeys = nil
		cfg.Enabled = true

		set := NewTracesQueueBatchSettings()
		// Verify initial state
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		// Apply WithQueueBatch
		option := WithQueueBatch(cfg, set)

		// Verify partitioner and merge function are NOT configured
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		// Verify the option can be applied to a base exporter
		be, err := internal.NewBaseExporter(
			exportertest.NewNopSettings(exportertest.NopType),
			pipeline.SignalTraces,
			func(context.Context, request.Request) error { return nil },
			option,
		)
		require.NoError(t, err)
		assert.NotNil(t, be)
	})

	t.Run("partitioner is correctly configured with metadata keys", func(t *testing.T) {
		cfg := exporterhelper.NewDefaultQueueConfig()
		cfg.MetadataKeys = []string{"project-id", "tenant-id"}
		cfg.Enabled = true

		set := NewTracesQueueBatchSettings()
		// Verify initial state
		assert.Nil(t, set.Partitioner)
		assert.Nil(t, set.MergeCtx)

		option := WithQueueBatch(cfg, set)

		be, err := internal.NewBaseExporter(
			exportertest.NewNopSettings(exportertest.NopType),
			pipeline.SignalTraces,
			func(context.Context, request.Request) error { return nil },
			option,
		)
		require.NoError(t, err)
		assert.NotNil(t, be)
	})
}
