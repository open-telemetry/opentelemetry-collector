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
	cfg := newTestConfigWithSizers()
	require.NoError(t, xconfmap.Validate(cfg))

	cfg.NumConsumers = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`num_consumers` must be positive")

	cfg = newTestConfigWithSizers()
	cfg.QueueSize = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`queue_size` must be positive")

	cfg = newTestConfigWithSizers()
	cfg.QueueSize = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`queue_size` must be positive")

	storageID := component.MustNewID("test")
	cfg = newTestConfigWithSizers()
	cfg.WaitForResult = true
	cfg.StorageID = &storageID
	require.EqualError(t, xconfmap.Validate(cfg), "`wait_for_result` is not supported with a persistent queue configured with `storage`")

	cfg = newTestConfigWithSizers()
	cfg.QueueSize = cfg.Batch.Get().Sizers[request.SizerTypeItems].MinSize - 1
	require.EqualError(t, xconfmap.Validate(cfg), "for sizer items, `min_size` (2048) must be less than or equal to `queue_size` (2047)")

	cfg = newTestConfigWithSizers()
	cfg.Batch.Get().Sizers = map[request.SizerType]SizerLimit{}
	cfg.Batch.Get().Sizer = request.SizerType{}
	require.EqualError(t, xconfmap.Validate(cfg), "batch: `batch` supports only `items` or `bytes` sizer, found \"\"")

	cfg = newTestConfigWithSizers()
	cfg.Sizer = request.SizerTypeBytes
	require.NoError(t, xconfmap.Validate(cfg))

	// Scenario 1: batch::sizer matches queue sizer, min_size > queue_size -> error
	cfg = newTestConfigWithSizers()
	cfg.Batch = configoptional.Some(BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      2048,
	})
	cfg.QueueSize = 2047
	require.EqualError(t, xconfmap.Validate(cfg), "for sizer items, `min_size` (2048) must be less than or equal to `queue_size` (2047)")

	// Scenario 3: batch::sizer does not match queue sizer, min_size > queue_size -> no error
	cfg = newTestConfigWithSizers()
	cfg.Sizer = request.SizerTypeBytes
	cfg.Batch = configoptional.Some(BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      2048,
	})
	cfg.QueueSize = 2047
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
	cfg.Sizer = request.SizerTypeItems
	require.EqualError(t, xconfmap.Validate(cfg), "both `sizer` and `sizers` are specified, but only one is allowed, `sizers` is preferred")

	cfg = newTestBatchConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, xconfmap.Validate(cfg), "`flush_timeout` must be positive, found 0")

	cfg = newTestBatchConfig()
	cfg.Sizers[request.SizerTypeItems] = SizerLimit{MinSize: -1}
	require.EqualError(t, xconfmap.Validate(cfg), "`min_size` must be non-negative for sizer \"items\", found -1")

	cfg = newTestBatchConfig()
	cfg.Sizers[request.SizerTypeItems] = SizerLimit{MaxSize: -1}
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` must be non-negative for sizer \"items\", found -1")

	cfg = newTestBatchConfig()
	cfg.Sizers = map[request.SizerType]SizerLimit{
		request.SizerTypeRequests: {MinSize: 1},
	}
	require.EqualError(t, xconfmap.Validate(cfg), "`batch` supports only `items` or `bytes` sizer, found \"requests\"")

	cfg = newTestBatchConfig()
	cfg.Sizers[request.SizerTypeItems] = SizerLimit{MinSize: 2048, MaxSize: 1024}
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` (1024) must be greater or equal to `min_size` (2048) for sizer \"items\"")

	// Test legacy fields when Sizers is empty
	cfg = BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      10,
	}
	require.NoError(t, xconfmap.Validate(cfg))

	cfg = BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeRequests,
	}
	require.EqualError(t, xconfmap.Validate(cfg), "`batch` supports only `items` or `bytes` sizer, found \"requests\"")

	cfg = BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      -1,
	}
	require.EqualError(t, xconfmap.Validate(cfg), "`min_size` must be non-negative, found -1")

	cfg = BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MaxSize:      -1,
	}
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` must be non-negative, found -1")

	cfg = BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizer:        request.SizerTypeItems,
		MinSize:      20,
		MaxSize:      10,
	}
	require.EqualError(t, xconfmap.Validate(cfg), "`max_size` (10) must be greater or equal to `min_size` (20)")
}

func newTestBatchConfig() BatchConfig {
	return BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		Sizers: map[request.SizerType]SizerLimit{
			request.SizerTypeItems: {MinSize: 2048},
		},
	}
}

func newTestConfigWithSizers() Config {
	return Config{
		Sizer:        request.SizerTypeItems,
		NumConsumers: 10,
		QueueSize:    100_000,
		Batch:        configoptional.Some(newTestBatchConfig()),
	}
}

func TestUnmarshal(t *testing.T) {
	newBaseCfg := func() configoptional.Optional[Config] {
		return configoptional.Some(Config{
			Sizer:        request.SizerTypeRequests,
			NumConsumers: 10,
			QueueSize:    1_000,
			Batch: configoptional.Default(BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
				Sizers: map[request.SizerType]SizerLimit{
					request.SizerTypeItems: {MinSize: 8192},
				},
			}),
		})
	}

	newLegacyBaseCfg := func() configoptional.Optional[Config] {
		return configoptional.Some(Config{
			Sizer:        request.SizerTypeRequests,
			NumConsumers: 10,
			QueueSize:    1_000,
			Batch: configoptional.Default(BatchConfig{
				FlushTimeout: 200 * time.Millisecond,
			}),
		})
	}

	tests := []struct {
		path        string
		expectedErr string
		expectedCfg func() configoptional.Optional[Config]
		baseCfg     func() configoptional.Optional[Config]
	}{
		{
			path: "batch_set_empty_explicit_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				// Batch is set, sizer is not overridden
				cfg.Get().Batch.GetOrInsertDefault()
				return cfg
			},
		},
		{
			path: "batch_set_empty_no_explicit_sizer.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Batch.GetOrInsertDefault()
				return cfg
			},
		},
		{
			path:    "batch_set_nonempty_explicit_sizer.yaml",
			baseCfg: newLegacyBaseCfg,
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newLegacyBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					Sizer:        request.SizerTypeBytes,
					MinSize:      100,
				})
				return cfg
			},
		},
		{
			path:    "batch_set_nonempty_explicit_batch_sizer.yaml",
			baseCfg: newLegacyBaseCfg,
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newLegacyBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 200 * time.Millisecond,
					Sizer:        request.SizerTypeItems,
					MinSize:      100,
				})
				return cfg
			},
		},
		{
			path: "batch_set_nonempty_multiple_sizers.yaml",
			expectedCfg: func() configoptional.Optional[Config] {
				cfg := newBaseCfg()
				cfg.Get().Sizer = request.SizerTypeBytes
				cfg.Get().QueueSize = 2000
				cfg.Get().Batch = configoptional.Some(BatchConfig{
					FlushTimeout: 2 * time.Second,
					Sizers: map[request.SizerType]SizerLimit{
						request.SizerTypeItems: {MinSize: 1, MaxSize: 12},
						request.SizerTypeBytes: {MinSize: 1000, MaxSize: 1000},
					},
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
					MinSize:      100,
					Sizers: map[request.SizerType]SizerLimit{
						request.SizerTypeItems: {MinSize: 8192},
					},
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
			path:        "batch_invalid_mixed_style.yaml",
			expectedErr: "cannot specify both `sizers` and legacy fields (`min_size`, `max_size`, `sizer`) in `batch`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			cm, err := confmaptest.LoadConf(filepath.Join("testdata", tt.path))
			require.NoError(t, err)

			cfg := newBaseCfg()
			if tt.baseCfg != nil {
				cfg = tt.baseCfg()
			}
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
