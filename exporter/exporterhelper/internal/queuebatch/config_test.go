// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
)

func TestConfig_Validate(t *testing.T) {
	cfg := newTestConfig()
	require.NoError(t, cfg.Validate())

	cfg.NumConsumers = 0
	require.EqualError(t, cfg.Validate(), "`num_consumers` must be positive")

	cfg = newTestConfig()
	cfg.QueueSize = 0
	require.EqualError(t, cfg.Validate(), "`queue_size` must be positive")

	cfg = newTestConfig()
	cfg.QueueSize = 0
	require.EqualError(t, cfg.Validate(), "`queue_size` must be positive")

	storageID := component.MustNewID("test")
	cfg = newTestConfig()
	cfg.WaitForResult = true
	cfg.StorageID = &storageID
	require.EqualError(t, cfg.Validate(), "`wait_for_result` is not supported with a persistent queue configured with `storage`")

	cfg = newTestConfig()
	cfg.Sizer = request.SizerTypeBytes
	cfg.StorageID = &storageID
	require.EqualError(t, cfg.Validate(), "persistent queue configured with `storage` only supports `requests` sizer")

	cfg = newTestConfig()
	cfg.Sizer = request.SizerTypeItems
	cfg.StorageID = &storageID
	require.EqualError(t, cfg.Validate(), "persistent queue configured with `storage` only supports `requests` sizer")

	cfg = newTestConfig()
	cfg.Sizer = request.SizerTypeRequests
	require.EqualError(t, cfg.Validate(), "`batch` supports only `items` or `bytes` sizer")

	cfg = newTestConfig()
	cfg.QueueSize = cfg.Batch.MinSize - 1
	require.EqualError(t, cfg.Validate(), "`min_size` must be less than or equal to `queue_size`")

	cfg = newTestConfig()
	cfg.Sizer = request.SizerTypeBytes
	require.NoError(t, cfg.Validate())

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	cfg.Enabled = false
	assert.NoError(t, cfg.Validate())
}

func TestConfigDeprecatedBlockingUnmarshal(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"enabled":       true,
		"num_consumers": 2,
		"queue_size":    100,
		"blocking":      true,
	})

	qCfg := Config{}
	assert.False(t, qCfg.BlockOnOverflow)
	require.NoError(t, conf.Unmarshal(&qCfg))
	assert.True(t, qCfg.BlockOnOverflow)
	assert.True(t, qCfg.hasBlocking)
}

func TestBatchConfig_Validate(t *testing.T) {
	cfg := newTestBatchConfig()
	require.NoError(t, cfg.Validate())

	cfg = newTestBatchConfig()
	cfg.FlushTimeout = 0
	require.EqualError(t, cfg.Validate(), "`flush_timeout` must be positive")

	cfg = newTestBatchConfig()
	cfg.MinSize = -1
	require.EqualError(t, cfg.Validate(), "`min_size` must be non-negative")

	cfg = newTestBatchConfig()
	cfg.MaxSize = -1
	require.EqualError(t, cfg.Validate(), "`max_size` must be non-negative")

	cfg = newTestBatchConfig()
	cfg.MinSize = 2048
	cfg.MaxSize = 1024
	require.EqualError(t, cfg.Validate(), "`max_size` must be greater or equal to `min_size`")
}

func newTestBatchConfig() BatchConfig {
	return BatchConfig{
		FlushTimeout: 200 * time.Millisecond,
		MinSize:      2048,
		MaxSize:      0,
	}
}
