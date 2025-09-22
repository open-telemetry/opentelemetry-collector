// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package queuebatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"
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

	// Confirm Validate doesn't return error with invalid config when feature is disabled
	cfg.Enabled = false
	assert.NoError(t, xconfmap.Validate(cfg))
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
