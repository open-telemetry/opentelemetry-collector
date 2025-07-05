// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package internal // import "go.opentelemetry.io/collector/exporter/exporterhelper/internal"

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queue"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/queuebatch"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/request"
	"go.opentelemetry.io/collector/exporter/exporterhelper/internal/sender"
)

// QueueBatchSettings is a subset of the queuebatch.Settings that are needed when used within an Exporter.
type QueueBatchSettings[T any] struct {
	Encoding    queue.Encoding[T]
	ItemsSizer  request.Sizer[T]
	BytesSizer  request.Sizer[T]
	Partitioner queuebatch.Partitioner[T]
}

// NewDefaultQueueConfig returns the default config for queuebatch.Config.
// By default, the queue stores 1000 requests of telemetry and is non-blocking when full.
func NewDefaultQueueConfig() queuebatch.Config {
	return queuebatch.Config{
		Enabled:      true,
		Sizer:        request.SizerTypeRequests,
		NumConsumers: 10,
		// By default, batches are 8192 spans, for a total of up to 8 million spans in the queue
		// This can be estimated at 1-4 GB worth of maximum memory usage
		// This default is probably still too high, and may be adjusted further down in a future release
		QueueSize:       1_000,
		BlockOnOverflow: false,
		StorageID:       nil,
		Batch:           nil,
	}
}

func NewQueueSender(
	qSet queuebatch.Settings[request.Request],
	qCfg queuebatch.Config,
	bCfg BatcherConfig,
	exportFailureMessage string,
	next sender.Sender[request.Request],
) (sender.Sender[request.Request], error) {
	exportFunc := func(ctx context.Context, req request.Request) error {
		// Have to read the number of items before sending the request since the request can
		// be modified by the downstream components like the batcher.
		itemsCount := req.ItemsCount()
		if errSend := next.Send(ctx, req); errSend != nil {
			qSet.Telemetry.Logger.Error("Exporting failed. Dropping data."+exportFailureMessage,
				zap.Error(errSend), zap.Int("dropped_items", itemsCount))
			return errSend
		}
		return nil
	}

	// TODO: Remove this when WithBatcher is removed.
	if bCfg.Enabled {
		return queuebatch.NewQueueBatchLegacyBatcher(qSet, newQueueBatchConfig(qCfg, bCfg), exportFunc)
	}
	return queuebatch.NewQueueBatch(qSet, newQueueBatchConfig(qCfg, bCfg), exportFunc)
}

func newQueueBatchConfig(qCfg queuebatch.Config, bCfg BatcherConfig) queuebatch.Config {
	// Overwrite configuration with the legacy BatcherConfig configured via WithBatcher.
	// TODO: Remove this when WithBatcher is removed.
	if !bCfg.Enabled {
		return qCfg
	}

	// User configured queueing, copy all config.
	if qCfg.Enabled {
		// Overwrite configuration with the legacy BatcherConfig configured via WithBatcher.
		// TODO: Remove this when WithBatcher is removed.
		qCfg.Batch = &queuebatch.BatchConfig{
			FlushTimeout: bCfg.FlushTimeout,
			MinSize:      bCfg.MinSize,
			MaxSize:      bCfg.MaxSize,
		}
		return qCfg
	}

	// This can happen only if the deprecated way to configure batching is used with a "disabled" queue.
	// TODO: Remove this when WithBatcher is removed.
	return queuebatch.Config{
		Enabled:         true,
		WaitForResult:   true,
		Sizer:           request.SizerTypeRequests,
		QueueSize:       math.MaxInt,
		NumConsumers:    runtime.NumCPU(),
		BlockOnOverflow: true,
		StorageID:       nil,
		Batch: &queuebatch.BatchConfig{
			FlushTimeout: bCfg.FlushTimeout,
			MinSize:      bCfg.MinSize,
			MaxSize:      bCfg.MaxSize,
		},
	}
}

// BatcherConfig defines a configuration for batching requests based on a timeout and a minimum number of items.
// Experimental: This API is at the early stage of development and may change without backward compatibility
// until https://github.com/open-telemetry/opentelemetry-collector/issues/8122 is resolved.
type BatcherConfig struct {
	// Enabled indicates whether to not enqueue batches before sending to the consumerSender.
	Enabled bool `mapstructure:"enabled"`

	// FlushTimeout sets the time after which a batch will be sent regardless of its size.
	FlushTimeout time.Duration `mapstructure:"flush_timeout"`

	// SizeConfig sets the size limits for a batch.
	SizeConfig `mapstructure:",squash"`
}

// SizeConfig sets the size limits for a batch.
type SizeConfig struct {
	Sizer request.SizerType `mapstructure:"sizer"`

	// MinSize defines the configuration for the minimum size of a batch.
	MinSize int64 `mapstructure:"min_size"`
	// MaxSize defines the configuration for the maximum size of a batch.
	MaxSize int64 `mapstructure:"max_size"`
}

func (c *BatcherConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.FlushTimeout <= 0 {
		return errors.New("`flush_timeout` must be greater than zero")
	}

	if c.Sizer != request.SizerTypeItems {
		return fmt.Errorf("unsupported sizer type: %q", c.Sizer)
	}
	if c.MinSize < 0 {
		return errors.New("`min_size` must be greater than or equal to zero")
	}
	if c.MaxSize < 0 {
		return errors.New("`max_size` must be greater than or equal to zero")
	}
	if c.MaxSize != 0 && c.MaxSize < c.MinSize {
		return errors.New("`max_size` must be greater than or equal to mix_size")
	}
	return nil
}

func NewDefaultBatcherConfig() BatcherConfig {
	return BatcherConfig{
		Enabled:      true,
		FlushTimeout: 200 * time.Millisecond,
		SizeConfig: SizeConfig{
			Sizer:   request.SizerTypeItems,
			MinSize: 8192,
		},
	}
}
