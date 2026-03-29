// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func TestConfig_Validate(t *testing.T) {
	cfg := newTestConfig()
	require.NoError(t, xconfmap.Validate(cfg))

	cfg.NumConsumers = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`num_consumers` must be positive")

	cfg = newTestConfig()
	cfg.QueueSize = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`queue_size` must be positive")

	cfg = newTestConfig()
	cfg.QueueSize = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`queue_size` must be positive")

	storageID := component.MustNewID("test")
	cfg = newTestConfig()
	cfg.WaitForResult = true
	cfg.StorageID = &storageID
	require.EqualError(t, xconfmap.Validate(cfg), "`wait_for_result` is not supported with a persistent queue configured with `storage`")

	cfg = newTestConfig()
	cfg.QueueSize = cfg.Batch.Get().MinSize - 1
	require.EqualError(t, xconfmap.Validate(cfg), "`min_size` must be less than or equal to `queue_size`")

	cfg = newTestConfig()
	cfg.Batch.Get().Sizer = request.SizerType{}
	require.EqualError(t, xconfmap.Validate(cfg), "batch: `batch` supports only `items` or `bytes` sizer, found \"\"")

	cfg = newTestConfig()
	cfg.Sizer = request.SizerTypeBytes
	require.NoError(t, xconfmap.Validate(cfg))
}

func TestBatchConfig_Validate_MetadataKeys(t *testing.T) {
	t.Run("no duplicates - valid", func(t *testing.T) {
		cfg := newTestBatchConfig()
		cfg.Partition.MetadataKeys = []string{"key1", "key2", "key3"}
		require.NoError(t, xconfmap.Validate(cfg))
	})

	t.Run("duplicate keys mixed case - invalid", func(t *testing.T) {
		cfg := newTestBatchConfig()
		cfg.Partition.MetadataKeys = []string{"Key1", "kEy1", "key2"}
		err := xconfmap.Validate(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate entry in metadata_keys")
		assert.Contains(t, err.Error(), "key1")
		assert.Contains(t, err.Error(), "case-insensitive")
	})

	t.Run("empty metadata_keys - valid", func(t *testing.T) {
		cfg := newTestBatchConfig()
		cfg.Partition.MetadataKeys = []string{}
		require.NoError(t, xconfmap.Validate(cfg))
	})

	t.Run("nil metadata_keys - valid", func(t *testing.T) {
		cfg := newTestBatchConfig()
		cfg.Partition.MetadataKeys = nil
		require.NoError(t, xconfmap.Validate(cfg))
	})

	t.Run("multiple duplicates - reports first duplicate", func(t *testing.T) {
		cfg := newTestBatchConfig()
		cfg.Partition.MetadataKeys = []string{"key1", "key2", "key1", "key2"}
		err := xconfmap.Validate(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate entry in metadata_keys")
		assert.Contains(t, err.Error(), "key1")
	})
}

func TestBatchConfig_Validate(t *testing.T) {
	cfg := newTestBatchConfig()
	require.NoError(t, xconfmap.Validate(cfg))

	cfg = newTestBatchConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`flush_timeout` must be positive, found 0")

	cfg = newTestBatchConfig()
	cfg.MinSize = -1
	require.EqualError(t, xconfmap.Validate(cfg), "`min_size` must be non-negative, found -1")

	cfg = newTestBatchConfig()
	cfg.MaxSize = -1
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` must be non-negative, found -1")

	cfg = newTestBatchConfig()
	cfg.Sizer = request.SizerTypeRequests
	require.EqualError(t, xconfmap.Validate(cfg), "`batch` supports only `items` or `bytes` sizer, found \"requests\"")

	cfg = newTestBatchConfig()
	cfg.Sizer = request.SizerType{}
	require.EqualError(t, xconfmap.Validate(cfg), "`batch` supports only `items` or `bytes` sizer, found \"\"")

	cfg = newTestBatchConfig()
	cfg.MinSize = 2048
	cfg.MaxSize = 1024
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` (1024) must be greater or equal to `min_size` (2048)")
}

func newTestBatchConfig() BatchConfig {
	return BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      2048,
		MaxSize:      0,
	}
}

func TestUnmarshal(t *testing.T) {
	newBaseCfg := func() configoptional.Optional[Config] {
		return configoptional.Some(Config{
			Sizer:        request.SizerTypeRequests,
			NumConsumers: 10,
			QueueSize:    10000,
			Batch: configoptional.Default(BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
				Sizer:        request.SizerTypeItems,
				MinSize:      8192,
			}),
		})
	}
	tests := []struct {
		path        string
		expectedErr string
		expectedCfg func() configoptional.Optional[Config]
	}{
		{
			path: "batch_set_empty_explicit_sizer_bytes.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 10000
				// Batch sizer inherited from queue sizer (bytes)
				cfg.Get().Batch.GetOrInsertDefault()
				cfg.Get().Batch.Get().Sizer = request.SizerTypeBytes
				return cfg
			},
		},
		{
			path: "batch_set_empty_explicit_sizer_requests.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeRequests
				cfg.Get().QueueSize = 10000
				// Queue sizer is "requests" (not valid for batch), so batch defaults to "items"
				cfg.Get().Batch.GetOrInsertDefault()
				cfg.Get().Batch.Get().Sizer = request.SizerTypeItems
				return cfg
			},
		},
		{
			path: "batch_set_empty_no_explicit_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				// Queue sizer is "requests" (not valid for batch), so batch defaults to "items"
				cfg.Get().Batch.GetOrInsertDefault()
				cfg.Get().Batch.Get().Sizer = request.SizerTypeItems
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_explicit_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Batch sizer inherited from queue sizer (bytes)
					Sizer:   request.SizerTypeBytes,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_no_explicit_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Queue sizer is "requests" (not valid for batch), so batch defaults to "items"
					Sizer:   request.SizerTypeItems,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_unset.yaml",
			// Batch remains unset, sizer override does not apply.
			expectedCfg: newBaseCfg,
		},
		{
			path: "batch_set_nonempty_queue_bytes_no_batch_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Batch sizer inherited from queue sizer (bytes)
					Sizer:   request.SizerTypeBytes,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_queue_items_no_batch_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeItems
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Batch sizer inherited from queue sizer (items)
					Sizer:   request.SizerTypeItems,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_queue_requests_no_batch_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeRequests
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Queue sizer is "requests" (not valid for batch), so batch defaults to "items"
					Sizer:   request.SizerTypeItems,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_explicit_batch_sizer_items.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Explicit batch sizer (items) must NOT be overridden by queue sizer (bytes)
					Sizer:   request.SizerTypeItems,
					MinSize: 100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_explicit_batch_sizer_bytes.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeItems
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					// Explicit batch sizer (bytes) must NOT be overridden by queue sizer (items)
					Sizer:   request.SizerTypeBytes,
					MinSize: 100,
				})
				return cfg
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.path))
			require.NoError(t, err)

			cfg := newBaseCfg()
			err = cm.Unmarshal(&cfg)
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedCfg(), cfg)
			assert.NoError(t, xconfmap.Validate(cfg))
		})
	}
}
