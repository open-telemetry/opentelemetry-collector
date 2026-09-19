// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/featuregate"
)

func TestSendingQueueValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  SendingQueue
		wantErr string
	}{
		{
			name:   "default",
			config: SendingQueue{Support: SendingQueueSupportDefault},
		},
		{
			name: "default with overrides",
			config: SendingQueue{
				Support:   SendingQueueSupportDefault,
				Overrides: map[string]any{"num_consumers": 1},
			},
			wantErr: "sending_queue.overrides requires support to be has_overrides",
		},
		{
			name: "has overrides",
			config: SendingQueue{
				Support: SendingQueueSupportHasOverrides,
				Overrides: map[string]any{
					"num_consumers":     1,
					"wait_for_result":   true,
					"block_on_overflow": true,
				},
			},
		},
		{
			name:    "has overrides without overrides",
			config:  SendingQueue{Support: SendingQueueSupportHasOverrides},
			wantErr: "sending_queue.overrides is required when support is has_overrides",
		},
		{
			name: "disabled with rationale",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Rationale: "Preserve existing behavior.",
				Overrides: map[string]any{"enabled": false},
			},
		},
		{
			name: "disabled without rationale",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"enabled": false},
			},
			wantErr: "sending_queue.rationale is required when overrides disable the queue",
		},
		{
			name: "omitted",
			config: SendingQueue{
				Support:   SendingQueueSupportOmitted,
				Rationale: "No sender.",
			},
		},
		{
			name:    "omitted without rationale",
			config:  SendingQueue{Support: SendingQueueSupportOmitted},
			wantErr: "sending_queue.rationale is required when support is omitted",
		},
		{
			name: "omitted with overrides",
			config: SendingQueue{
				Support:   SendingQueueSupportOmitted,
				Rationale: "No sender.",
				Overrides: map[string]any{"num_consumers": 1},
			},
			wantErr: "sending_queue.overrides cannot be set when support is omitted",
		},
		{
			name:    "invalid support",
			config:  SendingQueue{Support: "sometimes"},
			wantErr: "sending_queue.support must be one of",
		},
		{
			name: "unknown override",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"unknown": true},
			},
			wantErr: "invalid sending_queue.overrides",
		},
		{
			name: "invalid enabled override",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"enabled": "false"},
			},
			wantErr: "enabled must be a boolean",
		},
		{
			name: "invalid override value",
			config: SendingQueue{
				Support:   SendingQueueSupportHasOverrides,
				Overrides: map[string]any{"num_consumers": 0},
			},
			wantErr: "`num_consumers` must be positive",
		},
		{
			name: "invalid nested batch override",
			config: SendingQueue{
				Support: SendingQueueSupportHasOverrides,
				Overrides: map[string]any{
					"batch": map[string]any{
						"min_size": 12,
						"max_size": 10,
					},
				},
			},
			wantErr: "`max_size` (10) must be greater or equal to `min_size` (12)",
		},
		{
			name: "invalid nested batch enabled override",
			config: SendingQueue{
				Support: SendingQueueSupportHasOverrides,
				Overrides: map[string]any{
					"batch": map[string]any{"enabled": "false"},
				},
			},
			wantErr: "sending_queue.overrides.batch: enabled must be a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}

func TestSendingQueueAssignments(t *testing.T) {
	config := SendingQueue{
		Overrides: map[string]any{
			"enabled":       false,
			"num_consumers": 1,
			"batch": map[string]any{
				"enabled":  false,
				"min_size": int64(12),
			},
		},
	}

	actual, err := config.QueueAssignments()
	require.NoError(t, err)
	require.Equal(t, []string{"cfg.NumConsumers = 1"}, actual)

	actualBatch, err := config.BatchAssignments()
	require.NoError(t, err)
	require.Equal(t, []string{"batchCfg.MinSize = 12"}, actualBatch)

	actualYAML, err := config.IndentedYAMLQueueOverrides()
	require.NoError(t, err)
	require.NotContains(t, actualYAML, "\n  enabled: false")
	require.Contains(t, actualYAML, "batch:\n      enabled: false")
	require.Contains(t, actualYAML, "num_consumers: 1")
}

func TestSendingQueueAssignmentsAllFields(t *testing.T) {
	config := SendingQueue{
		Overrides: map[string]any{
			"wait_for_result":   true,
			"sizer":             "items",
			"queue_size":        int64(2048),
			"block_on_overflow": true,
			"storage":           "file_storage/queue",
			"num_consumers":     2,
			"batch": map[string]any{
				"flush_timeout": "3s",
				"sizer":         "bytes",
				"min_size":      int64(10),
				"max_size":      int64(20),
				"partition": map[string]any{
					"metadata_keys": []any{"tenant", "region"},
				},
			},
		},
	}

	queueAssignments, err := config.QueueAssignments()
	require.NoError(t, err)
	require.Equal(t, []string{
		"cfg.WaitForResult = true",
		"cfg.Sizer = exporterhelper.RequestSizerTypeItems",
		"cfg.QueueSize = 2048",
		"cfg.BlockOnOverflow = true",
		`storageID := component.MustNewIDWithName("file_storage", "queue")`,
		"cfg.StorageID = &storageID",
		"cfg.NumConsumers = 2",
	}, queueAssignments)

	batchAssignments, err := config.BatchAssignments()
	require.NoError(t, err)
	require.Equal(t, []string{
		"batchCfg.FlushTimeout = time.Duration(3000000000)",
		"batchCfg.Sizer = exporterhelper.RequestSizerTypeBytes",
		"batchCfg.MinSize = 10",
		"batchCfg.MaxSize = 20",
		`batchCfg.Partition.MetadataKeys = []string{"tenant", "region"}`,
	}, batchAssignments)
	require.True(t, config.HasStorageOverride())
	require.True(t, config.HasBatchFlushTimeoutOverride())
}

func TestSendingQueueOverridesApply(t *testing.T) {
	standard := exporterhelper.NewDefaultQueueConfig()
	standardBatchEnabled := standard.Batch.HasValue()
	standardBatch := *standard.Batch.GetOrInsertDefault()

	tests := []struct {
		name             string
		overrides        SendingQueueOverrides
		queueEnabled     bool
		batchEnabled     bool
		wantNumConsumers int
		wantBatchMinSize int64
	}{
		{
			name:             "no overrides",
			queueEnabled:     true,
			batchEnabled:     standardBatchEnabled,
			wantNumConsumers: standard.NumConsumers,
			wantBatchMinSize: standardBatch.MinSize,
		},
		{
			name:             "queue disabled",
			overrides:        SendingQueueOverrides{"enabled": false},
			queueEnabled:     false,
			batchEnabled:     standardBatchEnabled,
			wantNumConsumers: standard.NumConsumers,
			wantBatchMinSize: standardBatch.MinSize,
		},
		{
			name: "batch disabled preserves overridden values",
			overrides: SendingQueueOverrides{
				"batch": map[string]any{
					"enabled":  false,
					"min_size": int64(123),
				},
			},
			queueEnabled:     true,
			batchEnabled:     false,
			wantNumConsumers: standard.NumConsumers,
			wantBatchMinSize: 123,
		},
		{
			name: "batch fields enable batch",
			overrides: SendingQueueOverrides{
				"batch": map[string]any{"min_size": int64(123)},
			},
			queueEnabled:     true,
			batchEnabled:     true,
			wantNumConsumers: standard.NumConsumers,
			wantBatchMinSize: 123,
		},
		{
			name: "queue and batch overrides",
			overrides: SendingQueueOverrides{
				"num_consumers": 1,
				"batch": map[string]any{
					"enabled":  true,
					"min_size": int64(123),
				},
			},
			queueEnabled:     true,
			batchEnabled:     true,
			wantNumConsumers: 1,
			wantBatchMinSize: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.overrides.Apply(exporterhelper.NewDefaultQueueConfig())
			require.NoError(t, err)
			require.Equal(t, tt.queueEnabled, actual.HasValue())

			queueConfig := actual.GetOrInsertDefault()
			require.Equal(t, tt.wantNumConsumers, queueConfig.NumConsumers)
			require.Equal(t, tt.batchEnabled, queueConfig.Batch.HasValue())

			batchConfig := queueConfig.Batch.GetOrInsertDefault()
			require.Equal(t, tt.wantBatchMinSize, batchConfig.MinSize)
			require.Equal(t, standardBatch.FlushTimeout, batchConfig.FlushTimeout)
			require.Equal(t, standardBatch.Sizer, batchConfig.Sizer)
		})
	}
}

func TestDisabledBatchCanBeEnabledWithPreservedDefaults(t *testing.T) {
	overrides := SendingQueueOverrides{
		"batch": map[string]any{"enabled": false},
	}
	actual, err := overrides.Apply(exporterhelper.NewDefaultQueueConfig())
	require.NoError(t, err)

	queueConfig := actual.GetOrInsertDefault()
	require.False(t, queueConfig.Batch.HasValue())

	err = confmap.NewFromStringMap(map[string]any{"enabled": true}).Unmarshal(&queueConfig.Batch)
	require.NoError(t, err)
	require.True(t, queueConfig.Batch.HasValue())
	require.Equal(t, int64(8192), queueConfig.Batch.Get().MinSize)
	require.NotZero(t, queueConfig.Batch.Get().FlushTimeout)
}

func TestSendingQueueOverridesPreserveFeatureGateDefaults(t *testing.T) {
	const queueBatchFeatureGate = "pkg.exporterhelper.queueBatchEnabled"
	require.NoError(t, featuregate.GlobalRegistry().Set(queueBatchFeatureGate, false))
	t.Cleanup(func() {
		require.NoError(t, featuregate.GlobalRegistry().Set(queueBatchFeatureGate, false))
	})

	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			require.NoError(t, featuregate.GlobalRegistry().Set(queueBatchFeatureGate, enabled))
			standard := exporterhelper.NewDefaultQueueConfig()

			actual, err := (SendingQueueOverrides{}).Apply(exporterhelper.NewDefaultQueueConfig())
			require.NoError(t, err)
			require.Equal(t, configoptional.Some(standard), actual)

			actual, err = (SendingQueueOverrides{"enabled": false}).Apply(exporterhelper.NewDefaultQueueConfig())
			require.NoError(t, err)
			require.Equal(t, configoptional.Default(standard), actual)
		})
	}
}
